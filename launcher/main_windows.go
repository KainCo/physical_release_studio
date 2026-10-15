//go:build windows

// GOGAutoLauncher - Physical Release Studio.
// Build: GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w -H windowsgui" -o GOGAutoLauncher.exe .
// Then:  python patch_pe.py GOGAutoLauncher.exe   (run-from-swap flags, survives disc changes)
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/KainCo/physical_release_studio/launcher/core"
)

const (
	WS_CAPTION     = 0x00C00000
	WS_SYSMENU     = 0x00080000
	WS_MINIMIZEBOX = 0x00020000
	WS_VISIBLE     = 0x10000000
	WM_DESTROY     = 0x0002
	WM_PAINT       = 0x000F
	WM_CLOSE       = 0x0010
	WM_ERASEBKGD   = 0x0014
	WM_KEYDOWN     = 0x0100
	WM_LBUTTON_D   = 0x0201
	WM_LBUTTON_U   = 0x0202
	WM_APP         = 0x8000
	WM_APP_DONE    = WM_APP + 1 // staging worker finished
	WM_APP_ENDED   = WM_APP + 2 // installer process ended

	SW_SHOW                 = 5
	MB_OK                   = 0
	MB_OKCANCEL             = 1
	MB_YESNOCANCEL          = 3
	MB_YESNO                = 4
	MB_ICONERROR            = 0x10
	MB_ICONQUESTION         = 0x20
	MB_ICONINFORMATION      = 0x40
	IDOK                    = 1
	IDCANCEL                = 2
	IDYES                   = 6
	IDNO                    = 7
	BIF_RETURNONLYFSDIRS    = 0x0001
	BIF_NEWDIALOGSTYLE      = 0x0040
	IMAGE_BITMAP            = 0
	LR_LOADFROMFILE         = 0x0010
	LR_CREATEDIBSECTION     = 0x2000
	TRANSPARENT             = 1
	DT_LEFT                 = 0
	DT_CENTER               = 1
	DT_RIGHT                = 2
	DT_VCENTER              = 4
	DT_SINGLELINE           = 0x20
	DT_WORDBREAK            = 0x10
	SRCCOPY                 = 0x00CC0020
	HALFTONE                = 4
	SEE_MASK_NOCLOSEPROCESS = 0x40
	DRIVE_REMOTE            = 4
	VK_RETURN               = 0x0D
	VK_ESCAPE               = 0x1B

	clientW = 820
	clientH = 710

	btnInstall = 1
	btnExtra0  = 10 // 10..14
	btnCancel  = 20
	btnBrowse  = 21

	phaseMenu    = 0
	phaseCopy    = 1
	phaseWait    = 2
	phaseRunning = 3
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	pRegister  = user32.NewProc("RegisterClassExW")
	pCreate    = user32.NewProc("CreateWindowExW")
	pDef       = user32.NewProc("DefWindowProcW")
	pGetMsg    = user32.NewProc("GetMessageW")
	pTranslate = user32.NewProc("TranslateMessage")
	pDispatch  = user32.NewProc("DispatchMessageW")
	pQuit      = user32.NewProc("PostQuitMessage")
	pShow      = user32.NewProc("ShowWindow")
	pUpdate    = user32.NewProc("UpdateWindow")
	pMessage   = user32.NewProc("MessageBoxW")
	pBegin     = user32.NewProc("BeginPaint")
	pEnd       = user32.NewProc("EndPaint")
	pInval     = user32.NewProc("InvalidateRect")
	pValidate  = user32.NewProc("ValidateRect")
	pLoadImage = user32.NewProc("LoadImageW")
	pLoadCur   = user32.NewProc("LoadCursorW")
	pFillRect  = user32.NewProc("FillRect")
	pDrawText  = user32.NewProc("DrawTextW")
	pPost      = user32.NewProc("PostMessageW")
	pSetCap    = user32.NewProc("SetCapture")
	pRelCap    = user32.NewProc("ReleaseCapture")
	pAdjust    = user32.NewProc("AdjustWindowRectEx")
	pMetrics   = user32.NewProc("GetSystemMetrics")
	pDestroy   = user32.NewProc("DestroyWindow")

	pBrush    = gdi32.NewProc("CreateSolidBrush")
	pDelObj   = gdi32.NewProc("DeleteObject")
	pTextCol  = gdi32.NewProc("SetTextColor")
	pBkMode   = gdi32.NewProc("SetBkMode")
	pFont     = gdi32.NewProc("CreateFontW")
	pSelect   = gdi32.NewProc("SelectObject")
	pPen      = gdi32.NewProc("CreatePen")
	pMoveTo   = gdi32.NewProc("MoveToEx")
	pLineTo   = gdi32.NewProc("LineTo")
	pGetObj   = gdi32.NewProc("GetObjectW")
	pCompDC   = gdi32.NewProc("CreateCompatibleDC")
	pCompBmp  = gdi32.NewProc("CreateCompatibleBitmap")
	pBitBlt   = gdi32.NewProc("BitBlt")
	pStretch  = gdi32.NewProc("StretchBlt")
	pDelDC    = gdi32.NewProc("DeleteDC")
	pStretchM = gdi32.NewProc("SetStretchBltMode")

	pModule    = kernel32.NewProc("GetModuleHandleW")
	pDrives    = kernel32.NewProc("GetLogicalDrives")
	pDriveType = kernel32.NewProc("GetDriveTypeW")
	pDiskFree  = kernel32.NewProc("GetDiskFreeSpaceExW")
	pErrMode   = kernel32.NewProc("SetErrorMode")
	pUILang    = kernel32.NewProc("GetUserDefaultUILanguage")
	pWait      = kernel32.NewProc("WaitForSingleObject")
	pCloseH    = kernel32.NewProc("CloseHandle")

	pShell   = shell32.NewProc("ShellExecuteW")
	pShellEx = shell32.NewProc("ShellExecuteExW")
	pBrowse  = shell32.NewProc("SHBrowseForFolderW")
	pGetPath = shell32.NewProc("SHGetPathFromIDListW")

	pCoFree   = ole32.NewProc("CoTaskMemFree")
	pCoInit   = ole32.NewProc("CoInitializeEx")
	pCoUninit = ole32.NewProc("CoUninitialize")
)

type WNDCLASSEX struct {
	Size                               uint32
	Style                              uint32
	WndProc                            uintptr
	ClsExtra, WndExtra                 int32
	Instance, Icon, Cursor, Background uintptr
	MenuName, ClassName                *uint16
	IconSm                             uintptr
}
type POINT struct{ X, Y int32 }
type MSG struct {
	HWnd           uintptr
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
}
type RECT struct{ Left, Top, Right, Bottom int32 }
type PAINTSTRUCT struct {
	HDC                uintptr
	Erase              int32
	Paint              RECT
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type BITMAP struct {
	Type, Width, Height, WidthBytes uint32
	Planes, BitsPixel               uint16
	Bits                            uintptr
}
type BROWSEINFO struct {
	Owner       uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Fn          uintptr
	LParam      uintptr
	Image       int32
}
type SHELLEXECUTEINFO struct {
	Size     uint32
	Mask     uint32
	Hwnd     uintptr
	Verb     *uint16
	File     *uint16
	Params   *uint16
	Dir      *uint16
	Show     int32
	InstApp  uintptr
	IDList   uintptr
	Class    *uint16
	KeyClass uintptr
	HotKey   uint32
	Icon     uintptr
	Process  uintptr
}

// ---------------------------------------------------------------- state

type state struct {
	mu       sync.Mutex
	phase    int
	status   string
	detail   string
	permille int
	needDisc int
	cancel   bool
	closing  bool
	forced   bool
	manual   map[int]string
	tempDir  string
	result   error
}

type snapshot struct {
	phase, permille, need int
	status, detail        string
}

var (
	st        = state{manual: map[int]string{}}
	cfg       core.Config
	baseDir   string
	iniFound  bool
	hwndMain  uintptr
	coverBmp  uintptr
	S         = &en
	pressedID int
	fonts     = map[[2]int32]uintptr{}
	pens      = map[uint32]uintptr{}
)

func (s *state) snap() snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return snapshot{s.phase, s.permille, s.needDisc, s.status, s.detail}
}
func (s *state) setPhase(p int) {
	s.mu.Lock()
	s.phase = p
	s.mu.Unlock()
	invalidate()
}
func (s *state) isCancelled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cancel
}

func init() {
	// The window, its message loop and the COM apartment must live on ONE OS thread.
	// Without this, Go may move the main goroutine to another thread and the window stops responding.
	runtime.LockOSThread()
}

// ---------------------------------------------------------------- helpers

func u(s string) *uint16  { p, _ := syscall.UTF16PtrFromString(s); return p }
func i32(v int32) uintptr { return uintptr(uint32(v)) }

func invalidate() {
	if hwndMain != 0 {
		pInval.Call(hwndMain, 0, 0)
	}
}
func msgBox(text string, flags uintptr) int {
	r, _, _ := pMessage.Call(hwndMain, uintptr(unsafe.Pointer(u(text))), uintptr(unsafe.Pointer(u("Physical Release Studio"))), flags)
	return int(r)
}
func errBox(text string) { msgBox(text, MB_OK|MB_ICONERROR) }

func fmtBytes(n int64) string {
	const k = 1024.0
	f := float64(n)
	switch {
	case f < k*k:
		return fmt.Sprintf("%.0f KB", f/k)
	case f < k*k*k:
		return fmt.Sprintf("%.0f MB", f/(k*k))
	}
	return fmt.Sprintf("%.1f GB", f/(k*k*k))
}

func freeSpace(dir string) int64 {
	var free, total, totalFree uint64
	r, _, _ := pDiskFree.Call(uintptr(unsafe.Pointer(u(dir))), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&totalFree)))
	if r == 0 {
		return -1
	}
	return int64(free)
}

// ---------------------------------------------------------------- drawing

func rect(dc uintptr, x, y, w, h int32, color uint32) {
	b, _, _ := pBrush.Call(uintptr(color))
	r := RECT{x, y, x + w, y + h}
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), b)
	pDelObj.Call(b)
}
func getFont(size int32, bold bool) uintptr {
	w := int32(400)
	if bold {
		w = 700
	}
	key := [2]int32{size, w}
	if f, ok := fonts[key]; ok {
		return f
	}
	f, _, _ := pFont.Call(i32(-size), 0, 0, 0, uintptr(w), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(u("Tahoma"))))
	fonts[key] = f
	return f
}
func text(dc uintptr, s string, x, y, w, h int32, color uint32, size int32, bold bool, align uint32) {
	old, _, _ := pSelect.Call(dc, getFont(size, bold))
	pTextCol.Call(dc, uintptr(color))
	pBkMode.Call(dc, TRANSPARENT)
	r := RECT{x, y, x + w, y + h}
	pDrawText.Call(dc, uintptr(unsafe.Pointer(u(s))), uintptr(0xFFFFFFFF), uintptr(unsafe.Pointer(&r)), uintptr(align))
	pSelect.Call(dc, old)
}
func line(dc uintptr, x1, y1, x2, y2 int32, c uint32) {
	pen, ok := pens[c]
	if !ok {
		pen, _, _ = pPen.Call(0, 1, uintptr(c))
		pens[c] = pen
	}
	old, _, _ := pSelect.Call(dc, pen)
	pMoveTo.Call(dc, i32(x1), i32(y1), 0)
	pLineTo.Call(dc, i32(x2), i32(y2))
	pSelect.Call(dc, old)
}
func bevel(dc uintptr, x, y, w, h int32, pressed bool) {
	tl1, tl2, br1, br2 := uint32(0x00FFFFFF), uint32(0x00DFDFDF), uint32(0x00404040), uint32(0x00808080)
	if pressed {
		tl1, tl2, br1, br2 = 0x00404040, 0x00808080, 0x00FFFFFF, 0x00DFDFDF
	}
	line(dc, x, y, x+w-1, y, tl1)
	line(dc, x, y, x, y+h-1, tl1)
	line(dc, x+1, y+1, x+w-2, y+1, tl2)
	line(dc, x+1, y+1, x+1, y+h-2, tl2)
	line(dc, x, y+h-1, x+w-1, y+h-1, br1)
	line(dc, x+w-1, y, x+w-1, y+h-1, br1)
	line(dc, x+1, y+h-2, x+w-2, y+h-2, br2)
	line(dc, x+w-2, y+1, x+w-2, y+h-2, br2)
}

type button struct {
	id      int
	x, y    int32
	w, h    int32
	label   string
	primary bool
}

func layout(phase int) []button {
	var out []button
	switch phase {
	case phaseMenu:
		out = append(out, button{btnInstall, 210, 426, 400, 42, S.Install, true})
		for i, e := range cfg.Extras {
			if i >= 5 {
				break
			}
			l := e.Label
			if l == "" {
				l = filepath.Base(e.Path)
			}
			out = append(out, button{btnExtra0 + i, 28, int32(510 + i*31), 764, 27, l, false})
		}
	case phaseCopy:
		out = append(out, button{btnCancel, 28, 545, 180, 30, S.Cancel, false})
	case phaseWait:
		out = append(out, button{btnCancel, 28, 545, 180, 30, S.Cancel, false})
		out = append(out, button{btnBrowse, 220, 545, 220, 30, S.Browse, false})
	}
	return out
}
func drawButton(dc uintptr, b button, pressed bool) {
	rect(dc, b.x, b.y, b.w, b.h, 0x00C0C0C0)
	bevel(dc, b.x, b.y, b.w, b.h, pressed)
	o := int32(0)
	if pressed {
		o = 1
	}
	if b.primary {
		text(dc, b.label, b.x+8+o, b.y+2+o, b.w-16, b.h-4, 0, 12, true, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
	} else {
		text(dc, b.label, b.x+10+o, b.y+2+o, b.w-20, b.h-4, 0, 10, false, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	}
}
func drawCover(dc uintptr) {
	x, y, w, h := int32(28), int32(44), int32(764), int32(310)
	rect(dc, x-2, y-2, w+4, h+4, 0x00808080)
	rect(dc, x, y, w, h, 0)
	if coverBmp == 0 {
		text(dc, "NO COVER ART", x+8, y+h/2-12, w-16, 24, 0x00C0C0C0, 11, true, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		return
	}
	var bm BITMAP
	pGetObj.Call(coverBmp, unsafe.Sizeof(bm), uintptr(unsafe.Pointer(&bm)))
	if bm.Width == 0 || bm.Height == 0 {
		return
	}
	mem, _, _ := pCompDC.Call(dc)
	old, _, _ := pSelect.Call(mem, coverBmp)
	iw, ih := int32(bm.Width), int32(bm.Height)
	sc := float64(w) / float64(iw)
	if s2 := float64(h) / float64(ih); s2 < sc {
		sc = s2
	}
	dw, dh := int32(float64(iw)*sc), int32(float64(ih)*sc)
	pStretchM.Call(dc, HALFTONE)
	pStretch.Call(dc, i32(x+(w-dw)/2), i32(y+(h-dh)/2), i32(dw), i32(dh), mem, 0, 0, i32(iw), i32(ih), SRCCOPY)
	pSelect.Call(mem, old)
	pDelDC.Call(mem)
}
func draw(dc uintptr) {
	sn := st.snap()
	rect(dc, 0, 0, clientW, clientH, 0x00C0C0C0)
	rect(dc, 0, 0, clientW, 30, 0x00800000)
	text(dc, "Physical Release Studio", 8, 4, 620, 22, 0x00FFFFFF, 13, true, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	drawCover(dc)
	text(dc, cfg.Title, 28, 365, 764, 30, 0, 17, true, DT_CENTER|DT_VCENTER|DT_WORDBREAK)
	sub := S.SubTitle
	if cfg.MediaType != "" {
		sub += "  |  " + cfg.MediaType
	}
	if cfg.MediaCount > 1 {
		sub += fmt.Sprintf("  |  %d Discs", cfg.MediaCount)
	}
	text(dc, sub, 28, 397, 764, 18, 0x00404040, 9, false, DT_CENTER|DT_VCENTER|DT_SINGLELINE)

	if sn.phase == phaseMenu {
		text(dc, S.Extras, 28, 482, 764, 18, 0, 10, true, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		line(dc, 28, 502, 792, 502, 0x00808080)
		if len(cfg.Extras) == 0 {
			text(dc, S.NoExtras, 30, 510, 760, 22, 0x00404040, 10, false, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		}
	} else {
		status, hint := sn.status, ""
		switch sn.phase {
		case phaseCopy:
			status = S.Copying
		case phaseWait:
			status = fmt.Sprintf(S.InsertDisc, sn.need, cfg.MediaCount)
			hint = S.WaitHint
		case phaseRunning:
			status = S.Running
			hint = S.RunningHint
		}
		text(dc, status, 28, 430, 764, 26, 0, 13, true, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
		if sn.phase != phaseRunning {
			text(dc, sn.detail, 28, 460, 764, 36, 0x00404040, 10, false, DT_LEFT|DT_WORDBREAK)
			rect(dc, 28, 502, 764, 24, 0x00FFFFFF)
			bevel(dc, 28, 502, 764, 24, true)
			if sn.permille > 0 {
				rect(dc, 31, 505, int32(758*sn.permille/1000), 18, 0x00800000)
			}
			text(dc, fmt.Sprintf("%d%%", sn.permille/10), 28, 502, 764, 24, 0x00808080, 10, true, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
		if hint != "" {
			text(dc, hint, 28, 590, 764, 60, 0x00404040, 10, false, DT_LEFT|DT_WORDBREAK)
		}
	}
	for _, b := range layout(sn.phase) {
		drawButton(dc, b, pressedID == b.id)
	}
	line(dc, 8, 674, 812, 674, 0x00808080)
	text(dc, "PHYSICAL RELEASE STUDIO", 12, 681, 430, 18, 0x00404040, 8, true, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
	text(dc, "KAINCO  •  DEDICATED TO GAMES", 480, 681, 320, 18, 0x00404040, 8, false, DT_RIGHT|DT_VCENTER|DT_SINGLELINE)
}
func paint(hwnd uintptr) {
	var ps PAINTSTRUCT
	dc, _, _ := pBegin.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
	if dc == 0 {
		pValidate.Call(hwnd, 0) // never loop on WM_PAINT
		return
	}
	mem, _, _ := pCompDC.Call(dc)
	bmp, _, _ := pCompBmp.Call(dc, clientW, clientH)
	old, _, _ := pSelect.Call(mem, bmp)
	draw(mem)
	pBitBlt.Call(dc, 0, 0, clientW, clientH, mem, 0, 0, SRCCOPY)
	pSelect.Call(mem, old)
	pDelObj.Call(bmp)
	pDelDC.Call(mem)
	pEnd.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
}

// ---------------------------------------------------------------- actions (UI thread)

func openPath(path string) bool {
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	path = filepath.Clean(path)
	if _, e := os.Stat(path); e != nil {
		errBox(fmt.Sprintf(S.ErrNotFound, path))
		return false
	}
	r, _, _ := pShell.Call(hwndMain, uintptr(unsafe.Pointer(u("open"))), uintptr(unsafe.Pointer(u(path))), 0, uintptr(unsafe.Pointer(u(baseDir))), SW_SHOW)
	if r <= 32 {
		errBox(fmt.Sprintf(S.ErrOpen, path))
		return false
	}
	return true
}

func chooseFolder(prompt string) string {
	name := make([]uint16, 260)
	bi := BROWSEINFO{Owner: hwndMain, DisplayName: &name[0], Title: u(prompt), Flags: BIF_RETURNONLYFSDIRS | BIF_NEWDIALOGSTYLE}
	pid, _, _ := pBrowse.Call(uintptr(unsafe.Pointer(&bi)))
	runtime.KeepAlive(&bi)
	runtime.KeepAlive(name)
	if pid == 0 {
		return ""
	}
	defer pCoFree.Call(pid)
	buf := make([]uint16, 32768)
	r, _, _ := pGetPath.Call(pid, uintptr(unsafe.Pointer(&buf[0])))
	if r == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func onInstall() {
	if !iniFound {
		errBox(S.ErrNoINI)
		return
	}
	if cfg.Installer == "" {
		errBox(S.ErrNoInstaller)
		return
	}
	forced := cfg.Forced()
	tempParent := os.TempDir()
	if forced {
		need := cfg.StagingBytes + 256<<20
		for {
			free := freeSpace(tempParent)
			if free < 0 || free >= need {
				break
			}
			if msgBox(fmt.Sprintf(S.ErrNoSpace, fmtBytes(need), fmtBytes(free)), MB_OKCANCEL|MB_ICONERROR) != IDOK {
				return
			}
			f := chooseFolder(S.Browse)
			if f == "" {
				return
			}
			tempParent = f
		}
		startStaging(true, tempParent)
		return
	}
	if cfg.MediaCount > 1 {
		quick := false
		if len(cfg.Bins) > 0 && cfg.StagingBytes > 0 {
			need := cfg.StagingBytes * 3 // copy + installed game (estimate)
			free := freeSpace(tempParent)
			if free >= need {
				switch msgBox(fmt.Sprintf(S.ChooseMode, fmtBytes(need), fmtBytes(free)), MB_YESNOCANCEL|MB_ICONQUESTION) {
				case IDYES:
				case IDNO:
					quick = true
				default:
					return
				}
			} else if msgBox(fmt.Sprintf(S.NativeInfo, cfg.MediaCount), MB_OKCANCEL|MB_ICONINFORMATION) != IDOK {
				return
			}
		} else if msgBox(fmt.Sprintf(S.NativeInfo, cfg.MediaCount), MB_OKCANCEL|MB_ICONINFORMATION) != IDOK {
			return
		}
		if quick {
			startStaging(false, tempParent)
			return
		}
	}
	openPath(cfg.Installer) // launcher stays open (it is not needed any more, but harmless)
}

func onBrowse() {
	sn := st.snap()
	f := chooseFolder(fmt.Sprintf(S.PickFolder, sn.need))
	if f == "" {
		return
	}
	if !core.IsDiscRoot(&cfg, f, sn.need) {
		errBox(fmt.Sprintf(S.ErrNotDisc, sn.need))
		return
	}
	st.mu.Lock()
	st.manual[sn.need] = f
	st.mu.Unlock()
}

func onButton(id int) {
	switch {
	case id == btnInstall:
		onInstall()
	case id >= btnExtra0 && id < btnExtra0+5:
		if i := id - btnExtra0; i < len(cfg.Extras) {
			openPath(cfg.Extras[i].Path)
		}
	case id == btnCancel:
		st.mu.Lock()
		st.cancel = true
		st.mu.Unlock()
	case id == btnBrowse:
		onBrowse()
	}
}

func buttonAt(x, y int32) int {
	for _, b := range layout(st.snap().phase) {
		if x >= b.x && x < b.x+b.w && y >= b.y && y < b.y+b.h {
			return b.id
		}
	}
	return 0
}

// ---------------------------------------------------------------- staging

type winUI struct{}

func (winUI) Cancelled() bool { return st.isCancelled() }
func (winUI) Progress(done, total int64, file string) {
	pm := 0
	if total > 0 {
		pm = int(done * 1000 / total)
		if pm > 1000 {
			pm = 1000
		}
	}
	st.mu.Lock()
	changed := pm != st.permille || file != st.detail
	st.permille, st.detail = pm, file
	st.mu.Unlock()
	if changed {
		invalidate()
	}
}
func (winUI) AskDisc(n int) (string, error) {
	if r := findDisc(n); r != "" {
		return r, nil
	}
	st.mu.Lock()
	st.phase, st.needDisc = phaseWait, n
	st.mu.Unlock()
	invalidate()
	for {
		for i := 0; i < 5; i++ {
			if st.isCancelled() {
				return "", core.ErrCancelled
			}
			time.Sleep(100 * time.Millisecond)
		}
		if r := findDisc(n); r != "" {
			st.setPhase(phaseCopy)
			return r, nil
		}
	}
}

func findDisc(n int) string {
	st.mu.Lock()
	m := st.manual[n]
	st.mu.Unlock()
	if m != "" && core.IsDiscRoot(&cfg, m, n) {
		return m
	}
	// extracted package on the hard drive: sibling folders "Disc N"
	sib := filepath.Join(filepath.Dir(baseDir), fmt.Sprintf("Disc %d", n))
	if core.IsDiscRoot(&cfg, sib, n) {
		return sib
	}
	mask, _, _ := pDrives.Call()
	for i := 2; i < 26; i++ { // skip A: and B:
		if mask&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		if t, _, _ := pDriveType.Call(uintptr(unsafe.Pointer(u(root)))); t < 2 || t == DRIVE_REMOTE {
			continue
		}
		if core.IsDiscRoot(&cfg, root, n) {
			return root
		}
	}
	return ""
}

func startStaging(forced bool, tempParent string) {
	st.mu.Lock()
	st.cancel, st.permille, st.detail, st.forced, st.result, st.tempDir = false, 0, "", forced, nil, ""
	st.phase = phaseCopy
	st.mu.Unlock()
	invalidate()
	go func() {
		err := func() error {
			dir, e := os.MkdirTemp(tempParent, "PhysicalReleaseStudio-Install-*")
			if e != nil {
				return e
			}
			st.mu.Lock()
			st.tempDir = dir
			st.mu.Unlock()
			return core.Stage(&cfg, cfg.Plan(forced), baseDir, dir, winUI{})
		}()
		st.mu.Lock()
		st.result = err
		st.mu.Unlock()
		pPost.Call(hwndMain, WM_APP_DONE, 0, 0)
	}()
}

func removeTemp(dir string) {
	if dir != "" {
		go os.RemoveAll(dir)
	}
}

func onStageDone() {
	st.mu.Lock()
	err, dir, closing := st.result, st.tempDir, st.closing
	st.mu.Unlock()
	if err != nil || closing {
		if closing && dir != "" {
			os.RemoveAll(dir) // synchronous: the process is about to exit
		} else {
			removeTemp(dir)
		}
		st.setPhase(phaseMenu)
		if closing {
			pDestroy.Call(hwndMain)
			return
		}
		if !errors.Is(err, core.ErrCancelled) {
			errBox(fmt.Sprintf(S.ErrStage, err.Error()))
		}
		return
	}
	p := filepath.Join(dir, filepath.Base(cfg.Installer))
	sei := SHELLEXECUTEINFO{Mask: SEE_MASK_NOCLOSEPROCESS, Hwnd: hwndMain, Verb: u("open"), File: u(p), Dir: u(dir), Show: SW_SHOW}
	sei.Size = uint32(unsafe.Sizeof(sei))
	r, _, _ := pShellEx.Call(uintptr(unsafe.Pointer(&sei)))
	runtime.KeepAlive(&sei)
	if r == 0 {
		errBox(fmt.Sprintf(S.ErrStart, p))
		removeTemp(dir)
		st.setPhase(phaseMenu)
		return
	}
	if sei.Process == 0 {
		errBox(fmt.Sprintf(S.KeepTemp, dir))
		st.setPhase(phaseMenu)
		return
	}
	st.setPhase(phaseRunning)
	h := sei.Process
	go func() {
		pWait.Call(h, 0xFFFFFFFF)
		pCloseH.Call(h)
		pPost.Call(hwndMain, WM_APP_ENDED, 0, 0)
	}()
}

func onSetupEnded() {
	st.mu.Lock()
	dir := st.tempDir
	st.mu.Unlock()
	size := int64(cfg.StagingBytes)
	if msgBox(fmt.Sprintf(S.AskDelete, fmtBytes(size)), MB_YESNO|MB_ICONQUESTION) == IDYES {
		removeTemp(dir)
	} else {
		msgBox(fmt.Sprintf(S.KeepTemp, dir), MB_OK|MB_ICONINFORMATION)
	}
	st.setPhase(phaseMenu)
}

// ---------------------------------------------------------------- window

func wndProc(hwnd, m, w, l uintptr) uintptr {
	switch m {
	case WM_PAINT:
		paint(hwnd)
		return 0
	case WM_ERASEBKGD:
		return 1
	case WM_LBUTTON_D:
		if id := buttonAt(int32(int16(l)), int32(int16(l>>16))); id != 0 {
			pressedID = id
			pSetCap.Call(hwnd)
			invalidate()
		}
		return 0
	case WM_LBUTTON_U:
		pRelCap.Call()
		id := buttonAt(int32(int16(l)), int32(int16(l>>16)))
		p := pressedID
		pressedID = 0
		invalidate()
		if id != 0 && id == p {
			onButton(id)
		}
		return 0
	case WM_KEYDOWN:
		ph := st.snap().phase
		if w == VK_RETURN && ph == phaseMenu && l&(1<<30) == 0 { // ignore key auto-repeat
			onInstall()
		} else if w == VK_ESCAPE && (ph == phaseCopy || ph == phaseWait) {
			onButton(btnCancel)
		}
		return 0
	case WM_APP_DONE:
		onStageDone()
		return 0
	case WM_APP_ENDED:
		onSetupEnded()
		return 0
	case WM_CLOSE:
		ph := st.snap().phase
		if ph == phaseCopy || ph == phaseWait {
			st.mu.Lock()
			st.cancel, st.closing = true, true // window closes when the worker has stopped
			st.mu.Unlock()
			return 0
		}
		if ph == phaseRunning {
			st.mu.Lock()
			dir := st.tempDir
			st.mu.Unlock()
			if msgBox(fmt.Sprintf(S.CloseRunning, dir), MB_OKCANCEL|MB_ICONQUESTION) != IDOK {
				return 0
			}
		}
	case WM_DESTROY:
		pQuit.Call(0)
		return 0
	}
	r, _, _ := pDef.Call(hwnd, m, w, l)
	return r
}

func loadConfig() {
	f, err := os.Open(filepath.Join(baseDir, "GOGAutoLauncher.ini"))
	if err != nil {
		cfg = core.ParseINI(strings.NewReader(""))
		return
	}
	defer f.Close()
	iniFound = true
	cfg = core.ParseINI(f)
}

func main() {
	pErrMode.Call(1) // SEM_FAILCRITICALERRORS: no "no disk in drive" popups while scanning
	pCoInit.Call(0, 2)
	defer pCoUninit.Call()
	exe, e := os.Executable()
	if e != nil {
		return
	}
	baseDir = filepath.Dir(exe)
	loadConfig()
	lang, _, _ := pUILang.Call()
	if cfg.Language == "de" || (cfg.Language == "" && lang&0x3FF == 0x07) {
		S = &de
	}
	if cfg.Cover != "" {
		p := cfg.Cover
		if !filepath.IsAbs(p) {
			p = filepath.Join(baseDir, p)
		}
		coverBmp, _, _ = pLoadImage.Call(0, uintptr(unsafe.Pointer(u(p))), IMAGE_BITMAP, 0, 0, LR_LOADFROMFILE|LR_CREATEDIBSECTION)
	}
	inst, _, _ := pModule.Call(0)
	cur, _, _ := pLoadCur.Call(0, 32512) // IDC_ARROW
	class := u("PhysicalReleaseStudioLauncher")
	wc := WNDCLASSEX{Size: uint32(unsafe.Sizeof(WNDCLASSEX{})), WndProc: syscall.NewCallback(wndProc), Instance: inst, Cursor: cur, Background: 16, ClassName: class}
	if r, _, _ := pRegister.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		return
	}
	style := uintptr(WS_CAPTION | WS_SYSMENU | WS_MINIMIZEBOX)
	rc := RECT{0, 0, clientW, clientH}
	pAdjust.Call(uintptr(unsafe.Pointer(&rc)), style, 0, 0)
	ww, wh := rc.Right-rc.Left, rc.Bottom-rc.Top
	sx, _, _ := pMetrics.Call(0)
	sy, _, _ := pMetrics.Call(1)
	x, y := (int32(sx)-ww)/2, (int32(sy)-wh)/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	hwndMain, _, _ = pCreate.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(u("Physical Release Studio - "+cfg.Title))), style|WS_VISIBLE, i32(x), i32(y), i32(ww), i32(wh), 0, 0, inst, 0)
	if hwndMain == 0 {
		return
	}
	pShow.Call(hwndMain, SW_SHOW)
	pUpdate.Call(hwndMain)
	var m MSG
	for {
		r, _, _ := pGetMsg.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslate.Call(uintptr(unsafe.Pointer(&m)))
		pDispatch.Call(uintptr(unsafe.Pointer(&m)))
	}
}
