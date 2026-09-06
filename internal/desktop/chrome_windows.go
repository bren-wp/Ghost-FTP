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
	a.move(logo, 14, 11, 32, 32)

	// Reserve a fixed wordmark gutter large enough for the real Segoe UI bold
	// rendering seen on Windows runners. Keep the subtitle responsive inside
	// the remaining header space rather than allowing it to overlap the title.
	const titleX, titleWidth, subtitleX = 54, 168, 230
	a.move(a.brandTitle, titleX, 10, titleWidth, 35)
	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok != 0 {
		logicalWidth := a.unscale(int(client.Right - client.Left))
		subtitleWidth := logicalWidth - subtitleX - 320
		if subtitleWidth < 120 {
			subtitleWidth = 120
		}
		a.move(a.brandSubtitle, subtitleX, 17, subtitleWidth, 20)
	}
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
		// Native Header controls notify their immediate ListView parent. Install
		// that route before applying the dark list theme so header text and fill
		// stay under Ghost FTP control on both light and dark Windows desktops.
		installWorkspaceHeaderDraw(a, list)
		styleWorkspaceList(list)
	}
	// Remote operations remain disabled by updateActionControls. The list itself
	// stays enabled so Windows does not replace the dark list surface with the
	// disabled light theme seen on disconnected workspaces.
	setControlEnabled(a.remoteList, true)
	a.refineBrandHeader()
}
