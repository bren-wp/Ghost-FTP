//go:build windows

package platform

import (
	"syscall"
	"unsafe"
)

var (
	premiumGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	premiumDwmapi           = syscall.NewLazyDLL("dwmapi.dll")
	premiumDwmSetAttribute  = premiumDwmapi.NewProc("DwmSetWindowAttribute")
)

func premiumDialogPosition(width, height int) (int, int) {
	const (
		smCXScreen = 0
		smCYScreen = 1
	)
	w, _, _ := premiumGetSystemMetrics.Call(smCXScreen)
	h, _, _ := premiumGetSystemMetrics.Call(smCYScreen)
	x := (int(w) - width) / 2
	y := (int(h) - height) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return x, y
}

func premiumDialogFont(height int32, weight uintptr) uintptr {
	font, _, _ := promptCreateFontW.Call(
		uintptr(uint32(height)),
		0, 0, 0,
		weight,
		0, 0, 0,
		1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(promptWstr("Segoe UI"))),
	)
	return font
}

// applyPremiumDialogWindow uses best-effort DWM hints only. Failure is ignored
// so the same binary remains usable on Windows builds that do not expose a
// particular modern composition attribute.
func applyPremiumDialogWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	const (
		dwmUseImmersiveDarkMode   = 20
		dwmWindowCornerPreference = 33
		dwmWindowCornerRound      = 2
	)
	dark := int32(1)
	premiumDwmSetAttribute.Call(
		hwnd,
		dwmUseImmersiveDarkMode,
		uintptr(unsafe.Pointer(&dark)),
		unsafe.Sizeof(dark),
	)
	corner := uint32(dwmWindowCornerRound)
	premiumDwmSetAttribute.Call(
		hwnd,
		dwmWindowCornerPreference,
		uintptr(unsafe.Pointer(&corner)),
		unsafe.Sizeof(corner),
	)
}
