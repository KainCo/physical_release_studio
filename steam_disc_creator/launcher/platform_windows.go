//go:build windows

// Windows-Funktionen: Laufwerke, Registry, Verknüpfungen, Programme starten, Selbstlöschung.
// Nur Standardbibliothek; es gibt keinerlei Netzwerkzugriff.
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\PRS_`

func driveRoots() []string {
	m, _, _ := pGetLogicalDrives.Call()
	var out []string
	for i := 0; i < 26; i++ {
		if m&(1<<uint(i)) == 0 {
			continue
		}
		root := string(rune('A'+i)) + `:\`
		t, _, _ := pGetDriveType.Call(uintptr(unsafe.Pointer(u16(root))))
		// 2 = Wechseldatenträger (USB), 3 = Festplatte, 4 = Netzlaufwerk, 5 = CD/DVD/BD
		if t == 2 || t == 3 || t == 4 || t == 5 {
			out = append(out, root)
		}
	}
	return out
}

func defaultGamesRoot() string {
	d := os.Getenv("SystemDrive")
	if d == "" {
		d = "C:"
	}
	return d + `\Games`
}

func freeSpace(path string) (int64, error) {
	var avail, total, free uint64
	r, _, err := pGetDiskFreeSpace.Call(uintptr(unsafe.Pointer(u16(path))),
		uintptr(unsafe.Pointer(&avail)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&free)))
	if r == 0 {
		return 0, err
	}
	return int64(avail), nil
}

func uiLanguage() string {
	r, _, _ := pGetUILang.Call()
	switch r & 0x3ff {
	case 0x07:
		return "de"
	case 0x15:
		return "pl"
	case 0x0c:
		return "fr"
	case 0x0a:
		return "es"
	case 0x10:
		return "it"
	}
	return "en"
}

// ---------- Registry (nur HKEY_CURRENT_USER, keine Administratorrechte nötig) ----------

func regOpen(sub string, sam uint32, create bool) (uintptr, bool) {
	var h uintptr
	var r uintptr
	if create {
		r, _, _ = pRegCreateKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(sub))), 0, 0, 0,
			uintptr(sam), 0, uintptr(unsafe.Pointer(&h)), 0)
	} else {
		r, _, _ = pRegOpenKeyEx.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(sub))), 0,
			uintptr(sam), uintptr(unsafe.Pointer(&h)))
	}
	return h, r == 0
}

func regSetStr(h uintptr, name, val string) {
	b := syscall.StringToUTF16(val)
	pRegSetValueEx.Call(h, uintptr(unsafe.Pointer(u16(name))), 0, regSZ,
		uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)*2))
}

func regSetDword(h uintptr, name string, v uint32) {
	pRegSetValueEx.Call(h, uintptr(unsafe.Pointer(u16(name))), 0, regDWORD,
		uintptr(unsafe.Pointer(&v)), 4)
}

func regGetStr(h uintptr, name string) (string, bool) {
	var typ, size uint32
	r, _, _ := pRegQueryValueEx.Call(h, uintptr(unsafe.Pointer(u16(name))), 0,
		uintptr(unsafe.Pointer(&typ)), 0, uintptr(unsafe.Pointer(&size)))
	if r != 0 || typ != regSZ || size < 2 {
		return "", false
	}
	buf := make([]uint16, size/2+1)
	r, _, _ = pRegQueryValueEx.Call(h, uintptr(unsafe.Pointer(u16(name))), 0,
		uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r != 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

func regLookup(pkgID string) (string, bool) {
	h, ok := regOpen(uninstallKey+pkgID, keyRead, false)
	if !ok {
		return "", false
	}
	defer pRegCloseKey.Call(h)
	return regGetStr(h, "InstallLocation")
}

// regRegister trägt das Spiel unter "Apps & Features" ein (nur für den aktuellen Benutzer).
func regRegister(cfg *Config, dest string, sizeKB int64, uninstall string) error {
	h, ok := regOpen(uninstallKey+cfg.PackageID, keyWrite, true)
	if !ok {
		return os.ErrPermission
	}
	defer pRegCloseKey.Call(h)
	regSetStr(h, "DisplayName", cfg.Title)
	regSetStr(h, "InstallLocation", dest)
	regSetStr(h, "UninstallString", `"`+uninstall+`" /uninstall`)
	regSetStr(h, "DisplayIcon", exeFull(cfg, dest))
	regSetStr(h, "Publisher", "Physical Release Studio")
	regSetDword(h, "EstimatedSize", uint32(sizeKB))
	regSetDword(h, "NoModify", 1)
	regSetDword(h, "NoRepair", 1)
	return nil
}

func regUnregister(pkgID string) {
	pRegDeleteKey.Call(hkeyCurrentUser, uintptr(unsafe.Pointer(u16(uninstallKey+pkgID))))
}

// steamRoot liest den Steam-Ordner aus der Registry (nur lesend).
func steamRoot() string {
	h, ok := regOpen(`Software\Valve\Steam`, keyRead, false)
	if !ok {
		return ""
	}
	defer pRegCloseKey.Call(h)
	if p, ok := regGetStr(h, "SteamPath"); ok && p != "" {
		return filepath.FromSlash(p)
	}
	return ""
}

// ---------- Verknüpfungen (IShellLinkW über COM) ----------

var (
	clsidShellLink = guid{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidShellLinkW  = guid{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidPersistFile = guid{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

// comCall ruft die Methode idx der COM-Schnittstelle obj auf.
func comCall(obj unsafe.Pointer, idx int, args ...uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	fn := *(*uintptr)(unsafe.Add(vtbl, idx*int(unsafe.Sizeof(uintptr(0)))))
	all := append([]uintptr{uintptr(obj)}, args...)
	r, _, _ := syscall.SyscallN(fn, all...)
	return r
}

func makeShortcut(lnk, target, workdir, icon, desc string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	coInit()
	defer pCoUninitialize.Call()

	var link unsafe.Pointer
	r, _, _ := pCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidShellLink)), 0, 1, // CLSCTX_INPROC_SERVER
		uintptr(unsafe.Pointer(&iidShellLinkW)), uintptr(unsafe.Pointer(&link)))
	if r != 0 || link == nil {
		return syscall.EINVAL
	}
	defer comCall(link, 2) // Release

	set := func(idx int, s string) {
		p := u16(s)
		comCall(link, idx, uintptr(unsafe.Pointer(p)))
		runtime.KeepAlive(p)
	}
	set(20, target) // SetPath
	set(9, workdir) // SetWorkingDirectory
	set(7, desc)    // SetDescription
	ip := u16(icon) // SetIconLocation(path, index)
	comCall(link, 17, uintptr(unsafe.Pointer(ip)), 0)
	runtime.KeepAlive(ip)

	var pf unsafe.Pointer
	if comCall(link, 0, uintptr(unsafe.Pointer(&iidPersistFile)), uintptr(unsafe.Pointer(&pf))) != 0 || pf == nil {
		return syscall.EINVAL
	}
	defer comCall(pf, 2)
	lp := u16(lnk)
	hr := comCall(pf, 6, uintptr(unsafe.Pointer(lp)), 1) // IPersistFile::Save
	runtime.KeepAlive(lp)
	if hr != 0 {
		return syscall.EINVAL
	}
	return nil
}

func specialDir(kind string) string {
	csidl := uintptr(0)
	switch kind {
	case "desktop":
		csidl = 0x0010 // CSIDL_DESKTOPDIRECTORY
	case "programs":
		csidl = 0x0002 // CSIDL_PROGRAMS
	default:
		return ""
	}
	buf := u16buf(520)
	r, _, _ := pSHGetFolderPath.Call(0, csidl, 0, 0, uintptr(unsafe.Pointer(&buf[0])))
	if r != 0 {
		return ""
	}
	return fromU16(buf)
}

// ---------- Programme starten ----------

// launchFile öffnet eine Datei oder Adresse (z. B. steam://...) mit dem zugeordneten Programm.
func launchFile(path, dir string) error {
	var d uintptr
	if dir != "" {
		d = uintptr(unsafe.Pointer(u16(dir)))
	}
	r, _, _ := pShellExecute.Call(0, uintptr(unsafe.Pointer(u16("open"))), uintptr(unsafe.Pointer(u16(path))), 0, d, swShow)
	if r <= 32 {
		return syscall.EINVAL
	}
	return nil
}

// runInstaller startet eine Komponente (Visual C++, DirectX, ...) mit Administratorabfrage und wartet.
func runInstaller(path, args, dir string) (int, error) {
	sei := shellExecInfo{
		Mask: seeNoCloseProc | seeNoAsync,
		Verb: u16("runas"),
		File: u16(path),
		Dir:  u16(dir),
		Show: swShow,
	}
	if args != "" {
		sei.Params = u16(args)
	}
	sei.Size = uint32(unsafe.Sizeof(sei))
	r, _, err := pShellExecuteEx.Call(uintptr(unsafe.Pointer(&sei)))
	if r == 0 {
		return -1, err
	}
	if sei.Process == 0 {
		return 0, nil
	}
	defer pCloseHandle.Call(sei.Process)
	pWaitForSingle.Call(sei.Process, infinite)
	var code uint32
	pGetExitCode.Call(sei.Process, uintptr(unsafe.Pointer(&code)))
	return int(int32(code)), nil
}

// deleteSelfLater löscht nach dem Beenden des Programms die noch gesperrten Dateien.
func deleteSelfLater(pending []string, dir string, removeDir bool) {
	var parts []string
	parts = append(parts, "ping -n 3 127.0.0.1 >nul")
	for _, p := range pending {
		parts = append(parts, `del /f /q "`+p+`"`)
	}
	parts = append(parts, `del /f /q "`+filepath.Join(dir, logName)+`"`)
	if removeDir {
		parts = append(parts, `rmdir /s /q "`+dir+`"`)
	} else {
		parts = append(parts, `rmdir "`+dir+`"`)
	}
	cmd := exec.Command("cmd.exe")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
		CmdLine:       `cmd.exe /d /s /c "` + strings.Join(parts, " & ") + `"`,
	}
	cmd.Start()
}
