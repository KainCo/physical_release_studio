//go:build windows

// Oberfläche des Launchers (klassische Windows-Steuerelemente, "Win98-Optik").
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	winW = 640 // Breite des Fensterinhalts

	idMain, idRe, idUn, idExit = 201, 202, 203, 204
	idEdit, idBrowse           = 211, 212
	idChkDesk, idChkStart      = 213, 214
	idChkRedist, idDoInstall   = 215, 216
	idBack, idBtnCancel        = 217, 218
	idComp                     = 219

	wmAppProg  = wmApp + 1
	wmAppDisc  = wmApp + 2
	wmAppRetry = wmApp + 3
	wmAppDone  = wmApp + 4
	wmAppState = wmApp + 5
	wmAppComp  = wmApp + 6
)

const (
	pageHome = iota
	pageOpts
	pageProg
)

type discReq struct {
	n, total int
	reply    chan bool
}

type retryReq struct {
	file  string
	err   error
	reply chan bool
}

type finishMsg struct {
	err          error
	redistFailed bool
}

type ui struct {
	hwnd     uintptr
	hInst    uintptr
	font     uintptr
	fontBold uintptr
	cfg      *Config
	startDir string
	known    map[int]string
	banner   *Banner
	top      int // Höhe des Kopfbereichs (Banner oder Titelband)
	page     int

	installed bool
	instDest  string

	// Steuerelemente
	lblStatus, lblSteam                                      uintptr
	btnMain, btnRe, btnUn, btnExit, btnComp                  uintptr
	lblDest, edDest, btnBrowse, chkDesk, chkStart, chkRedist uintptr
	lblSpace, btnDoInstall, btnBack                          uintptr
	lblCopy, pbar, lblPct, btnCancel                         uintptr

	// Installation im Hintergrund
	cancel     context.CancelFunc
	mu         sync.Mutex
	progDone   int64
	progTotal  int64
	progFile   string
	statusText string
	pendDisc   *discReq
	pendRetry  *retryReq
	fin        *finishMsg
}

var (
	app      *ui
	wndProcC uintptr
)

func init() { runtime.LockOSThread() } // Fensternachrichten müssen im selben Betriebssystem-Thread laufen

func main() {
	pSetErrorMode.Call(semFailCrit) // keine Systemdialoge bei leeren Laufwerken
	setLang(uiLanguage())
	coInit()
	icc := initCommonControlsEx{Size: 8, ICC: iccProgress}
	pInitCommonCtrlsEx.Call(uintptr(unsafe.Pointer(&icc)))

	dir, self := exeDir()
	base := strings.ToLower(filepath.Base(self))
	uninstall := base == "uninstall.exe"
	for _, a := range os.Args[1:] {
		if strings.EqualFold(a, "/uninstall") || strings.EqualFold(a, "-uninstall") {
			uninstall = true
		}
	}
	if uninstall {
		runUninstallMode(dir)
		return
	}

	cfg, err := loadConfig(dir)
	if err != nil {
		messageBox(0, T("cfgErr", err.Error()), "Launcher", mbOK|mbIconError)
		return
	}
	a := &ui{cfg: cfg, startDir: dir, known: map[int]string{}}
	app = a
	for n := 1; n <= cfg.DiscCount; n++ {
		if checkDisc(dir, cfg, n) {
			a.known[n] = dir
		}
	}
	if cfg.Banner != "" {
		if rel, e := safeRel(cfg.Banner); e == nil {
			a.banner = loadBanner(filepath.Join(dir, rel), winW)
		}
	}
	a.top = 90
	if a.banner != nil {
		a.top = a.banner.H
	}
	a.run()
}

// runUninstallMode ist die Deinstallation (uninstall.exe im Spielordner).
func runUninstallMode(dir string) {
	info, err := ReadLog(dir)
	if err != nil {
		messageBox(0, T("unNoLog"), "Uninstall", mbOK|mbIconWarning)
		return
	}
	title := info.Title
	if messageBox(0, T("confirmUn", title), title, mbYesNo|mbIconQuestion) != idYes {
		return
	}
	_, err = RunUninstall(dir, func() bool {
		return messageBox(0, T("extra"), title, mbYesNo|mbIconQuestion) == idYes
	})
	if err != nil {
		messageBox(0, T("errGeneric", err.Error()), title, mbOK|mbIconWarning)
		return
	}
	text := T("unDone", title)
	if info.Steam {
		text += "\n\n" + T("unSteam")
	}
	messageBox(0, text, title, mbOK|mbIconInfo)
}

func (a *ui) run() {
	h, _, _ := pGetModuleHandle.Call(0)
	a.hInst = h
	wndProcC = syscall.NewCallback(wndProc)
	cursor, _, _ := pLoadCursor.Call(0, idcArrow)
	icon, _, _ := pLoadIcon.Call(h, 1) // Symbol aus der EXE-Ressource (ID 1)
	if icon == 0 {
		icon, _, _ = pLoadIcon.Call(0, idiApplication)
	}
	cls := u16("PRSLauncher")
	wc := wndClassEx{
		WndProc: wndProcC, Instance: h, Icon: icon, IconSm: icon, Cursor: cursor,
		Background: colorBtnFace + 1, ClassName: cls,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	pRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	clientH := a.top + 214
	r := rect{0, 0, winW, int32(clientH)}
	pAdjustWindowRect.Call(uintptr(unsafe.Pointer(&r)), uintptr(wsCaption|wsSysMenu|wsMinimizeBox), 0)
	ww, wh := int(r.Right-r.Left), int(r.Bottom-r.Top)
	sx, _, _ := pGetSystemMetrics.Call(smCxScreen)
	sy, _, _ := pGetSystemMetrics.Call(smCyScreen)
	x, y := (int(sx)-ww)/2, (int(sy)-wh)/2
	if y < 0 {
		y = 0
	}
	hwnd, _, _ := pCreateWindowEx.Call(wsExControlPar, uintptr(unsafe.Pointer(cls)),
		uintptr(unsafe.Pointer(u16(a.cfg.Title))),
		uintptr(wsCaption|wsSysMenu|wsMinimizeBox|wsClipChildren),
		uintptr(x), uintptr(y), uintptr(ww), uintptr(wh), 0, 0, h, 0)
	a.hwnd = hwnd
	a.font, _, _ = pGetStockObject.Call(defaultGUIFnt)
	a.fontBold, _, _ = pCreateFont.Call(uintptr(0xFFFFFFF1) /* -15 */, 0, 0, 0, 700, 0, 0, 0, 1, 0, 0, 0, 0,
		uintptr(unsafe.Pointer(u16("MS Shell Dlg"))))
	a.createControls()
	a.refreshState()
	a.showPage(pageHome)
	pShowWindow.Call(hwnd, swShow)
	pUpdateWindow.Call(hwnd)
	pSetForegroundWin.Call(hwnd)

	var m msgT
	for {
		r, _, _ := pGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		if ok, _, _ := pIsDialogMessage.Call(hwnd, uintptr(unsafe.Pointer(&m))); ok != 0 {
			continue
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		pDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
}

// ---------- Steuerelemente ----------

func (a *ui) mk(class, text string, style, ex uint32, x, y, w, h int, id int, font uintptr) uintptr {
	hw, _, _ := pCreateWindowEx.Call(uintptr(ex), uintptr(unsafe.Pointer(u16(class))), uintptr(unsafe.Pointer(u16(text))),
		uintptr(style|wsChild), uintptr(x), uintptr(y), uintptr(w), uintptr(h), a.hwnd, uintptr(id), a.hInst, 0)
	if font == 0 {
		font = a.font
	}
	sendMessage(hw, wmSetFont, font, 1)
	return hw
}

func (a *ui) createControls() {
	t := a.top
	lbl := uint32(ssLeft | ssNoPrefix)
	btn := uint32(bsPushButton | wsTabStop)
	// Home
	a.lblStatus = a.mk("STATIC", "", lbl, 0, 24, t+14, 592, 36, 101, 0)
	a.lblSteam = a.mk("STATIC", T("steamHint"), lbl, 0, 24, t+52, 592, 48, 102, 0)
	a.btnMain = a.mk("BUTTON", "", btn|bsDefPushButton, 0, 24, t+110, 200, 34, idMain, 0)
	a.btnRe = a.mk("BUTTON", T("reinstall"), btn, 0, 232, t+110, 190, 34, idRe, 0)
	a.btnUn = a.mk("BUTTON", T("uninstall"), btn, 0, 430, t+110, 186, 34, idUn, 0)
	a.btnExit = a.mk("BUTTON", T("exit"), btn, 0, 506, t+166, 110, 28, idExit, 0)
	a.btnComp = a.mk("BUTTON", T("comps"), btn, 0, 24, t+166, 330, 28, idComp, 0)
	// Optionen
	a.lblDest = a.mk("STATIC", T("installTo"), lbl, 0, 24, t+10, 592, 18, 111, 0)
	a.edDest = a.mk("EDIT", "", wsTabStop|esAutoHScroll, wsExClientEdge, 24, t+30, 484, 24, idEdit, 0)
	a.btnBrowse = a.mk("BUTTON", T("browse"), btn, 0, 514, t+29, 102, 26, idBrowse, 0)
	a.chkDesk = a.mk("BUTTON", T("scDesktop"), bsAutoCheckbox|wsTabStop, 0, 24, t+66, 592, 20, idChkDesk, 0)
	a.chkStart = a.mk("BUTTON", T("scStart"), bsAutoCheckbox|wsTabStop, 0, 24, t+88, 592, 20, idChkStart, 0)
	a.chkRedist = a.mk("BUTTON", T("redist"), bsAutoCheckbox|wsTabStop, 0, 24, t+110, 592, 20, idChkRedist, 0)
	a.lblSpace = a.mk("STATIC", "", lbl, 0, 24, t+140, 592, 18, 112, 0)
	a.btnDoInstall = a.mk("BUTTON", T("doInstall"), btn|bsDefPushButton, 0, 394, t+166, 110, 28, idDoInstall, 0)
	a.btnBack = a.mk("BUTTON", T("back"), btn, 0, 506, t+166, 110, 28, idBack, 0)
	// Fortschritt
	a.lblCopy = a.mk("STATIC", "", lbl|ssEndEllipsis, 0, 24, t+30, 592, 36, 121, 0)
	a.pbar = a.mk("msctls_progress32", "", 0, 0, 24, t+76, 592, 22, 122, 0)
	a.lblPct = a.mk("STATIC", "", lbl, 0, 24, t+104, 592, 18, 123, 0)
	a.btnCancel = a.mk("BUTTON", T("cancel"), btn, 0, 506, t+166, 110, 28, idBtnCancel, 0)
	sendMessage(a.pbar, pbmSetRange32, 0, 1000)
	sendMessage(a.chkDesk, bmSetCheck, 1, 0)
	sendMessage(a.chkStart, bmSetCheck, 1, 0)
	sendMessage(a.chkRedist, bmSetCheck, 1, 0)
}

func (a *ui) refreshState() {
	a.instDest, a.installed = IsInstalled(a.cfg)
	if a.installed {
		setText(a.lblStatus, T("installed", a.instDest))
		setText(a.btnMain, T("play"))
	} else {
		setText(a.lblStatus, T("notInstalled"))
		setText(a.btnMain, T("install"))
	}
}

func (a *ui) showPage(p int) {
	a.page = p
	vis := func(on bool, hs ...uintptr) {
		for _, h := range hs {
			c := swHide
			if on {
				c = swShow
			}
			pShowWindow.Call(h, uintptr(c))
		}
	}
	steam := a.cfg.IsSteam()
	vis(p == pageHome, a.lblStatus, a.btnMain, a.btnExit)
	vis(p == pageHome && steam, a.lblSteam)
	vis(p == pageHome && a.installed, a.btnRe, a.btnUn)
	vis(p == pageHome && a.installed && len(a.cfg.Redist) > 0, a.btnComp)
	vis(p == pageOpts, a.lblDest, a.edDest, a.btnBrowse, a.lblSpace, a.btnDoInstall, a.btnBack)
	vis(p == pageOpts && !steam, a.chkDesk, a.chkStart)
	vis(p == pageOpts && len(a.cfg.Redist) > 0, a.chkRedist)
	vis(p == pageProg, a.lblCopy, a.pbar, a.lblPct, a.btnCancel)
	if p == pageOpts && steam {
		// Steam-Hinweis anstelle der Verknüpfungen
		setText(a.lblSteam, T("steamHint"))
		pSetWindowPos.Call(a.lblSteam, 0, 24, uintptr(a.top+62), 592, 48, 0x0004)
		pShowWindow.Call(a.lblSteam, swShow)
	} else if p == pageHome && steam {
		pSetWindowPos.Call(a.lblSteam, 0, 24, uintptr(a.top+52), 592, 48, 0x0004)
	}
	if p == pageProg {
		sendMessage(a.pbar, pbmSetPos, 0, 0)
	}
	pInvalidateRect.Call(a.hwnd, 0, 1)
}

func (a *ui) updateSpace() {
	dest := strings.TrimSpace(getText(a.edDest))
	if dest == "" {
		setText(a.lblSpace, "")
		return
	}
	need, free, _ := CheckSpace(a.cfg, dest)
	setText(a.lblSpace, T("space", humanSize(need), humanSize(free)))
}

// ---------- Aktionen ----------

func (a *ui) onCommand(id int, code int) {
	switch id {
	case idMain:
		if a.installed {
			if err := PlayGame(a.cfg, a.instDest); err != nil {
				messageBox(a.hwnd, T("launchFail"), a.cfg.Title, mbOK|mbIconWarning)
				return
			}
			pDestroyWindow.Call(a.hwnd)
			return
		}
		setText(a.edDest, DefaultDir(a.cfg))
		a.updateSpace()
		a.showPage(pageOpts)
	case idRe:
		setText(a.edDest, a.instDest)
		a.updateSpace()
		a.showPage(pageOpts)
	case idUn:
		a.uninstall()
	case idComp:
		if messageBox(a.hwnd, T("compConfirm"), a.cfg.Title, mbYesNo|mbIconQuestion) != idYes {
			return
		}
		pEnableWindow.Call(a.btnComp, 0)
		setText(a.btnComp, T("compRunning"))
		cfg, dest, hwnd := a.cfg, a.instDest, a.hwnd
		go func() {
			var failed uintptr
			if RunRedist(cfg, dest, nil) {
				failed = 1
			}
			postMessage(hwnd, wmAppComp, failed, 0)
		}()
	case idExit:
		pDestroyWindow.Call(a.hwnd)
	case idEdit:
		if code == enChange {
			a.updateSpace()
		}
	case idBrowse:
		if p := a.browse(getText(a.edDest)); p != "" {
			setText(a.edDest, p)
			a.updateSpace()
		}
	case idBack:
		a.showPage(pageHome)
	case idDoInstall:
		a.startInstall()
	case idBtnCancel:
		if a.cancel != nil {
			a.cancel()
		}
	}
}

func (a *ui) browse(cur string) string {
	buf := u16buf(520)
	bi := browseInfo{Owner: a.hwnd, DisplayName: &buf[0], Title: u16(T("pick")), Flags: bifReturnDirs | bifNewDialog}
	pidl, _, _ := pSHBrowseForFolder.Call(uintptr(unsafe.Pointer(&bi)))
	if pidl == 0 {
		return ""
	}
	defer pCoTaskMemFree.Call(pidl)
	out := u16buf(520)
	if r, _, _ := pSHGetPathFromID.Call(pidl, uintptr(unsafe.Pointer(&out[0]))); r == 0 {
		return ""
	}
	return fromU16(out)
}

func (a *ui) uninstall() {
	title := a.cfg.Title
	if messageBox(a.hwnd, T("confirmUn", title), title, mbYesNo|mbIconQuestion) != idYes {
		return
	}
	_, err := RunUninstall(a.instDest, func() bool {
		return messageBox(a.hwnd, T("extra"), title, mbYesNo|mbIconQuestion) == idYes
	})
	if err != nil {
		messageBox(a.hwnd, T("errGeneric", err.Error()), title, mbOK|mbIconWarning)
	} else {
		text := T("unDone", title)
		if a.cfg.IsSteam() {
			text += "\n\n" + T("unSteam")
		}
		messageBox(a.hwnd, text, title, mbOK|mbIconInfo)
	}
	a.refreshState()
	a.showPage(pageHome)
}

func validDest(p string) bool {
	if p == "" || strings.ContainsAny(p, "*?\"<>|") {
		return false
	}
	if strings.HasPrefix(p, `\\`) {
		return len(p) > 4
	}
	return len(p) >= 3 && p[1] == ':' && (p[2] == '\\' || p[2] == '/')
}

func (a *ui) startInstall() {
	dest := filepath.Clean(strings.TrimSpace(getText(a.edDest)))
	if !validDest(dest) || isRootLike(dest) {
		messageBox(a.hwnd, T("badPath"), a.cfg.Title, mbOK|mbIconWarning)
		return
	}
	if _, _, ok := CheckSpace(a.cfg, dest); !ok {
		messageBox(a.hwnd, T("noSpace"), a.cfg.Title, mbOK|mbIconWarning)
		return
	}
	if es, err := os.ReadDir(dest); err == nil && len(es) > 0 {
		if _, e := os.Stat(filepath.Join(dest, logName)); e != nil { // nicht unsere eigene frühere Installation
			if messageBox(a.hwnd, T("folderNotEmpty"), a.cfg.Title, mbYesNo|mbIconQuestion) != idYes {
				return
			}
		}
	}
	opts := InstallOpts{
		Desktop:   sendMessage(a.chkDesk, bmGetCheck, 0, 0) == bstChecked,
		StartMenu: sendMessage(a.chkStart, bmGetCheck, 0, 0) == bstChecked,
		Redist:    len(a.cfg.Redist) > 0 && sendMessage(a.chkRedist, bmGetCheck, 0, 0) == bstChecked,
		SrcRoot:   a.srcRoot(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancel = cancel
	setText(a.lblCopy, T("installing"))
	setText(a.lblPct, "")
	a.showPage(pageProg)
	go a.worker(ctx, dest, opts)
}

// srcRoot liefert den Stamm der Disc 1 (falls bekannt).
func (a *ui) srcRoot() string {
	if r, ok := a.known[1]; ok {
		return r
	}
	return a.startDir
}

func (a *ui) worker(ctx context.Context, dest string, opts InstallOpts) {
	cfg := a.cfg
	h := Hooks{
		NeedDisc: func(n int) (string, error) {
			for {
				if r, ok := FindDisc(cfg, n, a.startDir, a.known); ok {
					a.known[n] = r
					return r, nil
				}
				req := &discReq{n: n, total: cfg.DiscCount, reply: make(chan bool, 1)}
				a.mu.Lock()
				a.pendDisc = req
				a.mu.Unlock()
				postMessage(a.hwnd, wmAppDisc, 0, 0)
				select {
				case ok := <-req.reply:
					if !ok {
						return "", ErrCancelled
					}
				case <-ctx.Done():
					return "", ErrCancelled
				}
			}
		},
		Progress: func(done, total int64, cur string) {
			a.mu.Lock()
			a.progDone, a.progTotal, a.progFile = done, total, cur
			a.mu.Unlock()
			postMessage(a.hwnd, wmAppProg, 0, 0)
		},
		Retry: func(file string, err error) bool {
			req := &retryReq{file: file, err: err, reply: make(chan bool, 1)}
			a.mu.Lock()
			a.pendRetry = req
			a.mu.Unlock()
			postMessage(a.hwnd, wmAppRetry, 0, 0)
			select {
			case ok := <-req.reply:
				return ok
			case <-ctx.Done():
				return false
			}
		},
	}
	err := Install(ctx, cfg, dest, h)
	fin := &finishMsg{err: err}
	if err == nil {
		fin.redistFailed = PostInstall(cfg, dest, opts, func(s string) {
			a.mu.Lock()
			a.statusText = s
			a.mu.Unlock()
			postMessage(a.hwnd, wmAppState, 0, 0)
		})
	}
	a.mu.Lock()
	a.fin = fin
	a.mu.Unlock()
	postMessage(a.hwnd, wmAppDone, 0, 0)
}

// ---------- Fensterprozedur ----------

func wndProc(hwnd, msg, wp, lp uintptr) uintptr {
	a := app
	if a == nil {
		r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
		return r
	}
	switch uint32(msg) {
	case wmCommand:
		a.onCommand(int(wp&0xFFFF), int(wp>>16)&0xFFFF)
		return 0
	case wmPaint:
		a.paint()
		return 0
	case wmAppProg:
		a.mu.Lock()
		done, total, file := a.progDone, a.progTotal, a.progFile
		a.mu.Unlock()
		pct := 0
		if total > 0 {
			pct = int(done * 1000 / total)
		}
		sendMessage(a.pbar, pbmSetPos, uintptr(pct), 0)
		setText(a.lblCopy, T("copying", file))
		setText(a.lblPct, fmt.Sprintf("%d %%    %s / %s", pct/10, humanSize(done), humanSize(total)))
		return 0
	case wmAppState:
		a.mu.Lock()
		s := a.statusText
		a.mu.Unlock()
		setText(a.lblCopy, s)
		return 0
	case wmAppComp:
		pEnableWindow.Call(a.btnComp, 1)
		setText(a.btnComp, T("comps"))
		if wp != 0 {
			messageBox(hwnd, T("redistFail"), a.cfg.Title, mbOK|mbIconWarning)
		} else {
			messageBox(hwnd, T("compDone"), a.cfg.Title, mbOK|mbIconInfo)
		}
		return 0
	case wmAppDisc:
		a.mu.Lock()
		req := a.pendDisc
		a.mu.Unlock()
		if req != nil {
			r := messageBox(hwnd, T("discPrompt", req.n, req.total), a.cfg.Title, mbOKCancel|mbIconInfo)
			req.reply <- (r == idOK)
		}
		return 0
	case wmAppRetry:
		a.mu.Lock()
		req := a.pendRetry
		a.mu.Unlock()
		if req != nil {
			why := req.err.Error()
			if errors.Is(req.err, ErrCorrupt) {
				why = T("corrupt")
			}
			r := messageBox(hwnd, T("retry", req.file, why), a.cfg.Title, mbYesNo|mbIconWarning)
			req.reply <- (r == idYes)
		}
		return 0
	case wmAppDone:
		a.mu.Lock()
		fin := a.fin
		a.mu.Unlock()
		a.cancel = nil
		if fin != nil {
			switch {
			case fin.err == nil:
				text := T("done")
				flags := uint32(mbOK | mbIconInfo)
				if fin.redistFailed {
					text += "\n\n" + T("redistFail")
					flags = mbOK | mbIconWarning
				}
				messageBox(hwnd, text, a.cfg.Title, flags)
			case errors.Is(fin.err, ErrCancelled):
				messageBox(hwnd, T("cancelled"), a.cfg.Title, mbOK|mbIconInfo)
			default:
				messageBox(hwnd, T("errGeneric", fin.err.Error()), a.cfg.Title, mbOK|mbIconError)
			}
		}
		a.refreshState()
		a.showPage(pageHome)
		return 0
	case wmClose:
		if a.page == pageProg {
			if a.cancel != nil {
				a.cancel()
			}
			return 0
		}
		pDestroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProc.Call(hwnd, msg, wp, lp)
	return r
}

// paint zeichnet Banner bzw. Titelband und – falls gewünscht – den Spieltitel.
func (a *ui) paint() {
	var ps paintStruct
	hdc, _, _ := pBeginPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
	defer pEndPaint.Call(a.hwnd, uintptr(unsafe.Pointer(&ps)))
	if a.banner != nil {
		bmi := bitmapInfoHeader{Size: 40, Width: int32(a.banner.W), Height: int32(a.banner.H), Planes: 1, BitCount: 32}
		pSetDIBitsToDevice.Call(hdc, 0, 0, uintptr(a.banner.W), uintptr(a.banner.H), 0, 0, 0, uintptr(a.banner.H),
			uintptr(unsafe.Pointer(&a.banner.BGRA[0])), uintptr(unsafe.Pointer(&bmi)), 0)
		if a.cfg.ShowTitle && a.page == pageHome {
			a.drawTitle(hdc, a.top-34, 0x00FFFFFF, true)
		}
		return
	}
	br, _, _ := pCreateSolidBrush.Call(0x00800000) // Navy (BGR)
	defer pDeleteObject.Call(br)
	band := rect{0, 0, winW, int32(a.top)}
	pFillRect.Call(hdc, uintptr(unsafe.Pointer(&band)), br)
	a.drawTitle(hdc, a.top/2-14, 0x00FFFFFF, false)
}

func (a *ui) drawTitle(hdc uintptr, y int, color uintptr, shadow bool) {
	old, _, _ := pSelectObject.Call(hdc, a.fontBold)
	pSetBkMode.Call(hdc, 1) // TRANSPARENT
	txt := u16(a.cfg.Title)
	if shadow {
		pSetTextColor.Call(hdc, 0)
		rs := rect{25, int32(y + 1), winW - 15, int32(y + 29)}
		pDrawText.Call(hdc, uintptr(unsafe.Pointer(txt)), 0xFFFFFFFF, uintptr(unsafe.Pointer(&rs)), dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	}
	pSetTextColor.Call(hdc, color)
	rc := rect{24, int32(y), winW - 16, int32(y + 28)}
	pDrawText.Call(hdc, uintptr(unsafe.Pointer(txt)), 0xFFFFFFFF, uintptr(unsafe.Pointer(&rc)), dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis)
	pSelectObject.Call(hdc, old)
}
