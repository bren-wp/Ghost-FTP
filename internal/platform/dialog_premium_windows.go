//go:build windows

package platform

import (
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var (
	premiumGetSystemMetrics = user32.NewProc("GetSystemMetrics")
	premiumDwmapi           = syscall.NewLazyDLL("dwmapi.dll")
	premiumDwmSetAttribute  = premiumDwmapi.NewProc("DwmSetWindowAttribute")
	premiumGdi32            = syscall.NewLazyDLL("gdi32.dll")
	premiumCreateSolidBrush = premiumGdi32.NewProc("CreateSolidBrush")
	premiumSetTextColor     = premiumGdi32.NewProc("SetTextColor")
	premiumSetBkColor       = premiumGdi32.NewProc("SetBkColor")
	premiumUxTheme          = syscall.NewLazyDLL("uxtheme.dll")
	premiumSetWindowTheme   = premiumUxTheme.NewProc("SetWindowTheme")
)

const (
	premiumWMCtlColorEdit    = 0x0133
	premiumWMCtlColorListBox = 0x0134
	premiumWMCtlColorBtn     = 0x0135
	premiumWMCtlColorStatic  = 0x0138
)

var (
	premiumDialogDark    atomic.Bool
	premiumDarkBrushOnce sync.Once
	premiumDarkBrush     uintptr
)

// SetDialogAppearance synchronizes the small native platform dialogs with the
// appearance selected by the desktop frontend. The installer does not call this
// function and therefore keeps the normal Windows light presentation.
func SetDialogAppearance(dark bool) {
	premiumDialogDark.Store(dark)
}

func premiumColor(r, g, b byte) uintptr {
	return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16
}

func premiumDialogSurfaceColor() uintptr {
	if premiumDialogDark.Load() {
		return premiumColor(15, 19, 28)
	}
	return premiumColor(255, 255, 255)
}

func premiumDialogTextColor() uintptr {
	if premiumDialogDark.Load() {
		return premiumColor(244, 247, 255)
	}
	return premiumColor(31, 35, 40)
}

func premiumDialogBackgroundBrush() uintptr {
	if !premiumDialogDark.Load() {
		// COLOR_WINDOW + 1 is the documented class-background convention for a
		// stock Windows white surface and does not allocate a GDI object.
		return 6
	}
	premiumDarkBrushOnce.Do(func() {
		premiumDarkBrush, _, _ = premiumCreateSolidBrush.Call(premiumDialogSurfaceColor())
	})
	if premiumDarkBrush == 0 {
		return 6
	}
	return premiumDarkBrush
}

// premiumDialogControlColor is shared by the prompt/option/about card window
// procedures. It prevents static labels and edit/list backgrounds from falling
// back to a white stock brush while the active application appearance is Dark.
func premiumDialogControlColor(hdc uintptr) uintptr {
	if hdc != 0 {
		premiumSetTextColor.Call(hdc, premiumDialogTextColor())
		premiumSetBkColor.Call(hdc, premiumDialogSurfaceColor())
	}
	return premiumDialogBackgroundBrush()
}

func applyPremiumDialogControl(hwnd uintptr, class string) {
	if hwnd == 0 {
		return
	}
	if !premiumDialogDark.Load() {
		premiumSetWindowTheme.Call(hwnd, 0, 0)
		return
	}
	theme := ""
	switch class {
	case "EDIT", "COMBOBOX":
		theme = "DarkMode_CFD"
	case "BUTTON", "LISTBOX":
		theme = "DarkMode_Explorer"
	}
	if theme != "" {
		premiumSetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(promptWstr(theme))), 0)
	}
}

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
// particular modern composition attribute. Unlike the previous implementation,
// the title bar follows the active appearance instead of being forced dark while
// the dialog body remained light.
func applyPremiumDialogWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	const (
		dwmUseImmersiveDarkMode   = 20
		dwmWindowCornerPreference = 33
		dwmWindowCornerRound      = 2
	)
	dark := int32(0)
	if premiumDialogDark.Load() {
		dark = 1
	}
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
