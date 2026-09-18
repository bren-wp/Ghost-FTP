//go:build windows

package desktop

import "unsafe"

// Windows system icon glyphs. The core code points are shared by Segoe Fluent
// Icons and Segoe MDL2 Assets, which lets Ghost FTP use the modern Windows 11
// font while retaining a Windows 10 fallback without shipping fonts, icon packs
// or other network-fetched UI assets.
const (
	iconConnect     = "\uE703"
	iconMore        = "\uE712"
	iconCancel      = "\uE711"
	iconForward     = "\uE72A"
	iconBack        = "\uE72B"
	iconSettings    = "\uE713"
	iconRefresh     = "\uE72C"
	iconUp          = "\uE74A"
	iconDelete      = "\uE74D"
	iconSave        = "\uE74E"
	iconPlay        = "\uE768"
	iconPause       = "\uE769"
	iconCopy        = "\uE8C8"
	iconClear       = "\uE894"
	iconSync        = "\uE895"
	iconDownload    = "\uE896"
	iconUpload      = "\uE898"
	iconRename      = "\uE8AC"
	iconDisconnect  = "\uE8CD"
	iconPermissions = "\uE8D7"
	iconOpenLocal   = "\uE8DA"
	iconNewFolder   = "\uE8F4"
	iconInfo        = "\uE946"
	iconSearch      = "\uE721"
	iconDiagnostics = "\uE9D9"
)

type buttonVariant uint8

const (
	buttonDefault buttonVariant = iota
	buttonSubtle
	buttonAccent
	buttonDanger
	buttonNavActive
)

type buttonVisual struct {
	Icon     string
	Label    string
	Variant  buttonVariant
	Vertical bool
	Badge    int
}

func (a *app) registerButtonVisual(hwnd uintptr, icon, label string, variant buttonVariant, vertical bool) uintptr {
	if hwnd == 0 {
		return hwnd
	}
	if a.buttons == nil {
		a.buttons = make(map[uintptr]buttonVisual)
	}
	badge := 0
	if previous, ok := a.buttons[hwnd]; ok {
		badge = previous.Badge
	}
	a.buttons[hwnd] = buttonVisual{Icon: icon, Label: label, Variant: variant, Vertical: vertical, Badge: badge}
	return hwnd
}

func (a *app) registerButton(hwnd uintptr, icon, label string, variant buttonVariant) uintptr {
	return a.registerButtonVisual(hwnd, icon, label, variant, false)
}

func (a *app) registerToolbarButton(hwnd uintptr, icon, label string, variant buttonVariant) uintptr {
	return a.registerButtonVisual(hwnd, icon, label, variant, true)
}

func (a *app) setButtonBadge(hwnd uintptr, count int) {
	if a == nil || hwnd == 0 {
		return
	}
	if count < 0 {
		count = 0
	}
	visual, ok := a.buttons[hwnd]
	if !ok || visual.Badge == count {
		return
	}
	visual.Badge = count
	a.buttons[hwnd] = visual
	invalidateRect.Call(hwnd, 0, 0)
}

func createIconFont(height int32) uintptr {
	// Segoe Fluent Icons ships with Windows 11. Segoe MDL2 Assets remains the
	// native fallback on Windows 10. We choose by OS build to avoid tofu glyphs.
	name := "Segoe MDL2 Assets"
	if windowsBuildNumber() >= 22000 {
		name = "Segoe Fluent Icons"
	}
	font, _, _ := createFontW.Call(
		uintptr(uint32(height)), 0, 0, 0, 400, 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(wstr(name))),
	)
	return font
}

func windowsBuildNumber() uint32 {
	var v rtlOsVersionInfo
	v.Size = uint32(unsafe.Sizeof(v))
	if r, _, _ := rtlGetVersion.Call(uintptr(unsafe.Pointer(&v))); int32(r) < 0 {
		return 0
	}
	return v.Build
}
