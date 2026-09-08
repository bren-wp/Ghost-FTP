//go:build windows

package platform

import (
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var premiumGetSystemMetrics = user32.NewProc("GetSystemMetrics")
var premiumGetActiveWindow = user32.NewProc("GetActiveWindow")
var premiumGetWindowRect = user32.NewProc("GetWindowRect")
var premiumGetDpiForWindow = user32.NewProc("GetDpiForWindow")
var premiumGetDpiForSystem = user32.NewProc("GetDpiForSystem")
var premiumAdjustWindowRectEx = user32.NewProc("AdjustWindowRectEx")
var premiumAdjustWindowRectExForDpi = user32.NewProc("AdjustWindowRectExForDpi")
var premiumEnableWindow = user32.NewProc("EnableWindow")
var premiumSetActiveWindow = user32.NewProc("SetActiveWindow")
var premiumIsWindow = user32.NewProc("IsWindow")
var premiumDwmapi = syscall.NewLazyDLL("dwmapi.dll")
var premiumDwmSetAttribute = premiumDwmapi.NewProc("DwmSetWindowAttribute")
var premiumGdi32 = syscall.NewLazyDLL("gdi32.dll")
var premiumCreateSolidBrush = premiumGdi32.NewProc("CreateSolidBrush")
var premiumSetTextColor = premiumGdi32.NewProc("SetTextColor")
var premiumSetBkColor = premiumGdi32.NewProc("SetBkColor")
var premiumUxTheme = syscall.NewLazyDLL("uxtheme.dll")
var premiumSetWindowTheme = premiumUxTheme.NewProc("SetWindowTheme")

const (
	premiumWMCtlColorEdit    = 0x0133
	premiumWMCtlColorListBox = 0x0134
	premiumWMCtlColorBtn     = 0x0135
	premiumWMCtlColorStatic  = 0x0138
)

type premiumRect struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

var premiumDialogDark atomic.Bool
var premiumDarkBrushOnce sync.Once
var premiumDarkBrush uintptr
var premiumLightBrushOnce sync.Once
var premiumLightBrush uintptr
var premiumDialogLabelsMu sync.RWMutex
var premiumDialogOKLabel = "OK"
var premiumDialogCancelLabel = "Cancel"

// SetDialogAppearance synchronizes the small native platform dialogs with the
// appearance selected by the desktop frontend. The installer does not call this
// function and therefore keeps the normal Windows light presentation.
func SetDialogAppearance(dark bool) {
	premiumDialogDark.Store(dark)
}

// SetDialogActionLabels keeps compatibility PromptDialog call sites localized
// without teaching the platform package about Ghost FTP's translation catalog.
func SetDialogActionLabels(okLabel, cancelLabel string) {
	if okLabel == "" {
		okLabel = "OK"
	}
	if cancelLabel == "" {
		cancelLabel = "Cancel"
	}
	premiumDialogLabelsMu.Lock()
	premiumDialogOKLabel = okLabel
	premiumDialogCancelLabel = cancelLabel
	premiumDialogLabelsMu.Unlock()
}

func dialogActionLabels() (string, string) {
	premiumDialogLabelsMu.RLock()
	defer premiumDialogLabelsMu.RUnlock()
	return premiumDialogOKLabel, premiumDialogCancelLabel
}

func premiumColor(r, g, b byte) uintptr {
	return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16
}

func premiumDialogSurfaceColor() uintptr {
	if premiumDialogDark.Load() {
		return premiumColor(15, 19, 28)
	}
	// A restrained neutral surface avoids the glaring pure-white flash that the
	// previous application-owned Light dialogs produced next to the main window.
	return premiumColor(246, 248, 251)
}

func premiumDialogTextColor() uintptr {
	if premiumDialogDark.Load() {
		return premiumColor(244, 247, 255)
	}
	return premiumColor(31, 35, 40)
}

func premiumDialogBackgroundBrush() uintptr {
	if premiumDialogDark.Load() {
		premiumDarkBrushOnce.Do(func() {
			premiumDarkBrush, _, _ = premiumCreateSolidBrush.Call(premiumColor(15, 19, 28))
		})
		if premiumDarkBrush != 0 {
			return premiumDarkBrush
		}
		return 6
	}
	premiumLightBrushOnce.Do(func() {
		premiumLightBrush, _, _ = premiumCreateSolidBrush.Call(premiumColor(246, 248, 251))
	})
	if premiumLightBrush != 0 {
		return premiumLightBrush
	}
	return 6
}

// premiumDialogControlColor is shared by the prompt/option/about card window
// procedures. It prevents static labels and edit/list backgrounds from falling
// back to an unrelated stock brush while the active application appearance is
// Dark or the softened application-owned Light palette is active.
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

func premiumDialogOwner() uintptr {
	owner, _, _ := premiumGetActiveWindow.Call()
	if owner == 0 {
		return 0
	}
	valid, _, _ := premiumIsWindow.Call(owner)
	if valid == 0 {
		return 0
	}
	return owner
}

func premiumDialogDPI(owner uintptr) uint32 {
	if owner != 0 {
		if err := premiumGetDpiForWindow.Find(); err == nil {
			dpi, _, _ := premiumGetDpiForWindow.Call(owner)
			if dpi >= 96 && dpi <= 768 {
				return uint32(dpi)
			}
		}
	}
	if err := premiumGetDpiForSystem.Find(); err == nil {
		dpi, _, _ := premiumGetDpiForSystem.Call()
		if dpi >= 96 && dpi <= 768 {
			return uint32(dpi)
		}
	}
	return 96
}

func premiumScale(value int, dpi uint32) int {
	if value < 0 {
		return -premiumScale(-value, dpi)
	}
	if dpi == 0 {
		dpi = 96
	}
	if value == 0 {
		return 0
	}
	return (value*int(dpi) + 48) / 96
}

// premiumDialogOuterSize converts a desired client-area size into the actual
// top-level window size. The old code treated the requested outer size as if it
// were the client size, so the Windows title bar and frame consumed the footer
// and clipped action buttons on current Windows builds and DPI settings.
func premiumDialogOuterSize(clientWidth, clientHeight int, style uint32, exStyle uint32, dpi uint32) (int, int) {
	r := premiumRect{
		Right:  int32(premiumScale(clientWidth, dpi)),
		Bottom: int32(premiumScale(clientHeight, dpi)),
	}
	adjusted := uintptr(0)
	if err := premiumAdjustWindowRectExForDpi.Find(); err == nil {
		adjusted, _, _ = premiumAdjustWindowRectExForDpi.Call(
			uintptr(unsafe.Pointer(&r)),
			uintptr(style),
			0,
			uintptr(exStyle),
			uintptr(dpi),
		)
	}
	if adjusted == 0 {
		r = premiumRect{
			Right:  int32(premiumScale(clientWidth, dpi)),
			Bottom: int32(premiumScale(clientHeight, dpi)),
		}
		adjusted, _, _ = premiumAdjustWindowRectEx.Call(
			uintptr(unsafe.Pointer(&r)),
			uintptr(style),
			0,
			uintptr(exStyle),
		)
	}
	if adjusted == 0 {
		return premiumScale(clientWidth, dpi), premiumScale(clientHeight+48, dpi)
	}
	return int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func premiumDialogPosition(owner uintptr, width, height int) (int, int) {
	if owner != 0 {
		var r premiumRect
		if ok, _, _ := premiumGetWindowRect.Call(owner, uintptr(unsafe.Pointer(&r))); ok != 0 {
			x := int(r.Left) + (int(r.Right-r.Left)-width)/2
			y := int(r.Top) + (int(r.Bottom-r.Top)-height)/2
			if x < 0 {
				x = 0
			}
			if y < 0 {
				y = 0
			}
			return x, y
		}
	}
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

// premiumModalOwner makes the custom top-level window actually modal. Without
// this, the nested message loop still dispatched clicks and keyboard commands
// to the main Ghost FTP window behind the dialog.
func premiumModalOwner(owner uintptr) func() {
	if owner == 0 {
		return func() {}
	}
	premiumEnableWindow.Call(owner, 0)
	var once sync.Once
	return func() {
		once.Do(func() {
			valid, _, _ := premiumIsWindow.Call(owner)
			if valid == 0 {
				return
			}
			premiumEnableWindow.Call(owner, 1)
			premiumSetActiveWindow.Call(owner)
		})
	}
}

func premiumDialogFontForDPI(height int32, dpi uint32, weight uintptr) uintptr {
	scaledHeight := int32(premiumScale(int(height), dpi))
	font, _, _ := promptCreateFontW.Call(
		uintptr(uint32(scaledHeight)),
		0, 0, 0,
		weight,
		0, 0, 0,
		1, 0, 0, 5, 0,
		uintptr(unsafe.Pointer(promptWstr("Segoe UI"))),
	)
	return font
}

func premiumDialogFont(height int32, weight uintptr) uintptr {
	return premiumDialogFontForDPI(height, premiumDialogDPI(0), weight)
}

// applyPremiumDialogWindow uses best-effort DWM hints only. Failure is ignored
// so the same binary remains usable on Windows builds that do not expose a
// particular modern composition attribute. The title bar follows the active
// appearance instead of being forced dark while the dialog body remains light.
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
