//go:build windows

package desktop

import "unsafe"

// ensureSiteManagerButton creates the one-click Sites control independently of
// the legacy toolbar layout. Keeping it idempotent lets action-state refreshes
// safely restore the current font after DPI changes without creating duplicate
// controls or a second Site Manager implementation.
func (a *app) ensureSiteManagerButton() {
	if a == nil || a.hwnd == 0 {
		return
	}
	if a.siteManagerBtn != 0 {
		if a.font != 0 {
			sendMessageW.Call(a.siteManagerBtn, wmSetFont, a.font, 1)
		}
		return
	}

	hinst, _, _ := getModuleHandleW.Call(0)
	label := nativeMenuWords(a.languageCode())[1]
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|bsOwnerDraw|wsTabStop),
		0, 0, 10, 10,
		a.hwnd, uintptr(idSiteManager), hinst, 0,
	)
	if hwnd == 0 {
		return
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	applyDarkControl(hwnd, "BUTTON")
	a.siteManagerBtn = a.registerButton(hwnd, iconOpenLocal, label, buttonDefault)
}
