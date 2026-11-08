//go:build windows

// Dünner Win32-Unterbau ausschließlich mit der Go-Standardbibliothek (kein externes Modul).
// SPDX-License-Identifier: GPL-3.0-or-later
package main

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")

	pRegisterClassEx   = user32.NewProc("RegisterClassExW")
	pCreateWindowEx    = user32.NewProc("CreateWindowExW")
	pDefWindowProc     = user32.NewProc("DefWindowProcW")
	pGetMessage        = user32.NewProc("GetMessageW")
	pTranslateMessage  = user32.NewProc("TranslateMessage")
	pDispatchMessage   = user32.NewProc("DispatchMessageW")
	pIsDialogMessage   = user32.NewProc("IsDialogMessageW")
	pPostQuitMessage   = user32.NewProc("PostQuitMessage")
	pShowWindow        = user32.NewProc("ShowWindow")
	pUpdateWindow      = user32.NewProc("UpdateWindow")
	pSetWindowText     = user32.NewProc("SetWindowTextW")
	pGetWindowText     = user32.NewProc("GetWindowTextW")
	pGetWindowTextLen  = user32.NewProc("GetWindowTextLengthW")
	pSendMessage       = user32.NewProc("SendMessageW")
	pPostMessage       = user32.NewProc("PostMessageW")
	pMessageBox        = user32.NewProc("MessageBoxW")
	pLoadCursor        = user32.NewProc("LoadCursorW")
	pLoadIcon          = user32.NewProc("LoadIconW")
	pAdjustWindowRect  = user32.NewProc("AdjustWindowRect")
	pGetSystemMetrics  = user32.NewProc("GetSystemMetrics")
	pSetWindowPos      = user32.NewProc("SetWindowPos")
	pBeginPaint        = user32.NewProc("BeginPaint")
	pEndPaint          = user32.NewProc("EndPaint")
	pEnableWindow      = user32.NewProc("EnableWindow")
	pDestroyWindow     = user32.NewProc("DestroyWindow")
	pInvalidateRect    = user32.NewProc("InvalidateRect")
	pSetForegroundWin  = user32.NewProc("SetForegroundWindow")
	pGetSysColor       = user32.NewProc("GetSysColor")
	pSetFocus          = user32.NewProc("SetFocus")
	pGetStockObject    = gdi32.NewProc("GetStockObject")
	pCreateFont        = gdi32.NewProc("CreateFontW")
	pSetDIBitsToDevice = gdi32.NewProc("SetDIBitsToDevice")
	pGetModuleHandle   = kernel32.NewProc("GetModuleHandleW")
	pGetLogicalDrives  = kernel32.NewProc("GetLogicalDrives")
	pGetDriveType      = kernel32.NewProc("GetDriveTypeW")
	pGetDiskFreeSpace  = kernel32.NewProc("GetDiskFreeSpaceExW")
	pGetUILang         = kernel32.NewProc("GetUserDefaultUILanguage")
	pCloseHandle       = kernel32.NewProc("CloseHandle")
	pWaitForSingle     = kernel32.NewProc("WaitForSingleObject")
	pGetExitCode       = kernel32.NewProc("GetExitCodeProcess")
	pShellExecute      = shell32.NewProc("ShellExecuteW")
	pShellExecuteEx    = shell32.NewProc("ShellExecuteExW")
	pSHBrowseForFolder = shell32.NewProc("SHBrowseForFolderW")
	pSHGetPathFromID   = shell32.NewProc("SHGetPathFromIDListW")
	pSHGetFolderPath   = shell32.NewProc("SHGetFolderPathW")
	pCoInitializeEx    = ole32.NewProc("CoInitializeEx")
	pCoUninitialize    = ole32.NewProc("CoUninitialize")
	pCoCreateInstance  = ole32.NewProc("CoCreateInstance")
	pCoTaskMemFree     = ole32.NewProc("CoTaskMemFree")
	pRegCreateKeyEx    = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueEx     = advapi32.NewProc("RegSetValueExW")
	pRegCloseKey       = advapi32.NewProc("RegCloseKey")
	pRegDeleteKey      = advapi32.NewProc("RegDeleteKeyW")
	pRegOpenKeyEx      = advapi32.NewProc("RegOpenKeyExW")
	pRegQueryValueEx   = advapi32.NewProc("RegQueryValueExW")
	pInitCommonCtrlsEx = comctl32.NewProc("InitCommonControlsEx")
)

const (
	wsCaption      = 0x00C00000
	wsSysMenu      = 0x00080000
	wsMinimizeBox  = 0x00020000
	wsVisible      = 0x10000000
	wsChild        = 0x40000000
	wsTabStop      = 0x00010000
	wsGroup        = 0x00020000
	wsClipChildren = 0x02000000
	wsExClientEdge = 0x00000200
	wsExControlPar = 0x00010000

	bsPushButton    = 0x0
	bsDefPushButton = 0x1
	bsAutoCheckbox  = 0x3
	ssLeft          = 0x0
	ssCenter        = 0x1
	ssNoPrefix      = 0x80
	ssEndEllipsis   = 0x4000
	esAutoHScroll   = 0x80

	wmCreate        = 0x0001
	wmDestroy       = 0x0002
	wmPaint         = 0x000F
	wmClose         = 0x0010
	wmSetFont       = 0x0030
	wmCommand       = 0x0111
	wmApp           = 0x8000
	bmGetCheck      = 0x00F0
	bmSetCheck      = 0x00F1
	pbmSetRange32   = 0x0406
	pbmSetPos       = 0x0402
	swHide          = 0
	swShow          = 5
	mbOK            = 0x0
	mbOKCancel      = 0x1
	mbYesNo         = 0x4
	mbIconError     = 0x10
	mbIconQuestion  = 0x20
	mbIconWarning   = 0x30
	mbIconInfo      = 0x40
	idOK            = 1
	idCancel        = 2
	idYes           = 6
	colorBtnFace    = 15
	idcArrow        = 32512
	idiApplication  = 32512
	smCxScreen      = 0
	smCyScreen      = 1
	seeNoCloseProc  = 0x40
	seeNoAsync      = 0x100
	infinite        = 0xFFFFFFFF
	hkeyCurrentUser = 0x80000001
	keyRead         = 0x20019
	keyWrite        = 0x20006
	regSZ           = 1
	regDWORD        = 4
)

type wndClassEx struct {
	Size       uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type point struct{ X, Y int32 }

type msgT struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
}

type rect struct{ Left, Top, Right, Bottom int32 }

type paintStruct struct {
	Hdc         uintptr
	Erase       int32
	Paint       rect
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}

type bitmapInfoHeader struct {
	Size          uint32
	Width, Height int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPels, YPels  int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type browseInfo struct {
	Owner       uintptr
	Root        uintptr
	DisplayName *uint16
	Title       *uint16
	Flags       uint32
	Fn          uintptr
	LParam      uintptr
	Image       int32
}

type shellExecInfo struct {
	Size      uint32
	Mask      uint32
	Hwnd      uintptr
	Verb      *uint16
	File      *uint16
	Params    *uint16
	Dir       *uint16
	Show      int32
	InstApp   uintptr
	IDList    uintptr
	Class     *uint16
	KeyClass  uintptr
	HotKey    uint32
	IconOrMon uintptr
	Process   uintptr
}

type guid struct {
	D1 uint32
	D2 uint16
	D3 uint16
	D4 [8]byte
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func u16buf(n int) []uint16 { return make([]uint16, n) }

func fromU16(b []uint16) string { return syscall.UTF16ToString(b) }

func messageBox(hwnd uintptr, text, caption string, flags uint32) int {
	r, _, _ := pMessageBox.Call(hwnd, uintptr(unsafe.Pointer(u16(text))), uintptr(unsafe.Pointer(u16(caption))), uintptr(flags))
	return int(r)
}

func sendMessage(hwnd uintptr, msg uint32, w, l uintptr) uintptr {
	r, _, _ := pSendMessage.Call(hwnd, uintptr(msg), w, l)
	return r
}

func postMessage(hwnd uintptr, msg uint32, w, l uintptr) {
	pPostMessage.Call(hwnd, uintptr(msg), w, l)
}

func setText(hwnd uintptr, s string) {
	pSetWindowText.Call(hwnd, uintptr(unsafe.Pointer(u16(s))))
}

func getText(hwnd uintptr) string {
	n, _, _ := pGetWindowTextLen.Call(hwnd)
	buf := u16buf(int(n) + 2)
	pGetWindowText.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return fromU16(buf)
}

func coInit() { pCoInitializeEx.Call(0, 2) } // COINIT_APARTMENTTHREADED
