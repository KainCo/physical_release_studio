//go:build windows

// Weitere Win32-Funktionen für die Oberfläche (nur Standardbibliothek).
// SPDX-License-Identifier: GPL-3.0-or-later
package main

var (
	pSetErrorMode     = kernel32.NewProc("SetErrorMode")
	pFillRect         = user32.NewProc("FillRect")
	pDrawText         = user32.NewProc("DrawTextW")
	pGetClientRect    = user32.NewProc("GetClientRect")
	pCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	pDeleteObject     = gdi32.NewProc("DeleteObject")
	pSelectObject     = gdi32.NewProc("SelectObject")
	pSetBkMode        = gdi32.NewProc("SetBkMode")
	pSetTextColor     = gdi32.NewProc("SetTextColor")
)

const (
	wsBorder      = 0x00800000
	wmEraseBkgnd  = 0x0014
	enChange      = 0x0300
	dtLeft        = 0x0
	dtVCenter     = 0x4
	dtSingleLine  = 0x20
	dtEndEllipsis = 0x8000
	bifReturnDirs = 0x0001
	bifNewDialog  = 0x0040
	semFailCrit   = 0x0001
	bstChecked    = 1
	iccProgress   = 0x20
	defaultGUIFnt = 17
)

type initCommonControlsEx struct{ Size, ICC uint32 }
