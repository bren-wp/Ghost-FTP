//go:build windows

package desktop

import (
	"sync"
	"unsafe"
)

const (
	ssIcon      = 0x00000003
	stmSetImage = 0x0172
	imageIcon   = 1
	lrShared    = 0x00008000
)

var (
	brandLogoWindows sync.Map
	loadImageW       = user32.NewProc("LoadImageW")
)

// ensureBrandLogo reuses the canonical PE icon that is already shipped in the
// Setup and Portable binaries. Keeping the workspace mark tied to resource 1
// prevents a second logo source from drifting away from build/icon.ico.
func (a *app) ensureBrandLogo() uintptr {
	if a == nil || a.hwnd == 0 {
		return 0
	}
	if value, ok := brandLogoWindows.Load(a.hwnd); ok {
		if logo, ok := value.(uintptr); ok && logo != 0 {
			return logo
		}
	}

	hinst, _, _ := getModuleHandleW.Call(0)
	iconSize := a.scale(32)
	icon, _, _ := loadImageW.Call(hinst, 1, imageIcon, uintptr(iconSize), uintptr(iconSize), lrShared)
	if icon == 0 {
		// LoadIcon returns a shared icon resource too, so neither path requires
		// Ghost FTP to destroy a handle that Windows owns.
		icon, _, _ = loadIconW.Call(hinst, 1)
	}
	if icon == 0 {
		return 0
	}

	logo, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("STATIC"))),
		0,
		uintptr(wsChild|wsVisible|ssIcon),
		0, 0, uintptr(iconSize), uintptr(iconSize),
		a.hwnd, 0, hinst, 0,
	)
	if logo == 0 {
		return 0
	}
	sendMessageW.Call(logo, stmSetImage, imageIcon, icon)
	brandLogoWindows.Store(a.hwnd, logo)
	return logo
}

func (a *app) refineBrandHeader() {
	logo := a.ensureBrandLogo()
	if logo == 0 {
		return
	}
	// Geometry for the title/subtitle remains owned by app.layout. This helper
	// only anchors the canonical PE icon so the full “Ghost FTP” wordmark never
	// gets clipped by a second competing layout rule.
	a.move(logo, 14, 11, 32, 32)
}

func styleWorkspaceCombos(combos ...uintptr) {
	theme := uintptr(unsafe.Pointer(wstr("DarkMode_CFD")))
	for _, combo := range combos {
		if combo != 0 {
			setWindowTheme.Call(combo, theme, 0)
		}
	}
}

// stabilizeWorkspaceChrome is intentionally idempotent because state changes
// and resizes both flow through refineWorkspaceLayout. It prevents a disabled
// list-view from reverting to the bright system background while disconnected.
func (a *app) stabilizeWorkspaceChrome() {
	if a == nil {
		return
	}
	styleWorkspaceCombos(a.languageCombo, a.profilesCombo, a.protocol)
	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		styleWorkspaceList(list)
	}
	// Remote operations remain disabled by updateActionControls. The list itself
	// stays enabled so Windows does not replace the dark list surface with the
	// disabled light theme seen on disconnected workspaces.
	setControlEnabled(a.remoteList, true)
	a.refineBrandHeader()
}
