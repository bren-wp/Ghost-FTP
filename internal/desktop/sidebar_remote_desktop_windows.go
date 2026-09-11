//go:build windows

package desktop

import (
	"sync"
	"unsafe"
)

const idRemoteDesktop = 96

var sidebarRemoteDesktop sync.Map

func (a *app) ensureSidebarRemoteDesktop() uintptr {
	if a == nil || a.hwnd == 0 {
		return 0
	}
	if value, ok := sidebarRemoteDesktop.Load(a.hwnd); ok {
		if hwnd, ok := value.(uintptr); ok && hwnd != 0 {
			return hwnd
		}
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	label := "Remote Desktop"
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
		0, 0, 1, 1,
		a.hwnd, idRemoteDesktop, hinst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	applyDarkControl(hwnd, "BUTTON")
	a.registerToolbarButton(hwnd, iconDiagnostics, label, buttonSubtle)
	sidebarRemoteDesktop.Store(a.hwnd, hwnd)
	return hwnd
}

func (a *app) sidebarRemoteDesktopButton() uintptr {
	if a == nil {
		return 0
	}
	value, ok := sidebarRemoteDesktop.Load(a.hwnd)
	if !ok {
		return 0
	}
	hwnd, _ := value.(uintptr)
	return hwnd
}
