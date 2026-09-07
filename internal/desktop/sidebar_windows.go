//go:build windows

package desktop

import (
	"sync"
	"unsafe"
)

const (
	applicationSidebarX       = 14
	applicationSidebarWidth   = 132
	applicationContentLeft    = 160
	applicationSidebarCardH   = 54
	applicationSidebarCardGap = 8
)

var (
	sidebarDiagnostics sync.Map
	sidebarGetWindowRect = user32.NewProc("GetWindowRect")
	sidebarMapWindowPoints = user32.NewProc("MapWindowPoints")
)

func (a *app) ensureSidebarDiagnostics() uintptr {
	if a == nil || a.hwnd == 0 {
		return 0
	}
	if value, ok := sidebarDiagnostics.Load(a.hwnd); ok {
		if hwnd, ok := value.(uintptr); ok && hwnd != 0 {
			return hwnd
		}
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	label := nativeMenuWords(a.languageCode())[8]
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
		0, 0, 1, 1,
		a.hwnd, idDiagnostics, hinst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	applyDarkControl(hwnd, "BUTTON")
	a.registerToolbarButton(hwnd, iconDiagnostics, label, buttonSubtle)
	sidebarDiagnostics.Store(a.hwnd, hwnd)
	return hwnd
}

func (a *app) sidebarDiagnosticsButton() uintptr {
	if a == nil {
		return 0
	}
	value, ok := sidebarDiagnostics.Load(a.hwnd)
	if !ok {
		return 0
	}
	hwnd, _ := value.(uintptr)
	return hwnd
}

func (a *app) setSidebarButtonVisual(hwnd uintptr, icon, label string, variant buttonVariant) {
	if a == nil || hwnd == 0 {
		return
	}
	setText(hwnd, label)
	if a.buttons == nil {
		a.buttons = make(map[uintptr]buttonVisual)
	}
	a.buttons[hwnd] = buttonVisual{Icon: icon, Label: label, Variant: variant, Vertical: true}
	invalidateRect.Call(hwnd, 0, 1)
}

func (a *app) sidebarLogicalRect(hwnd uintptr) (rect, bool) {
	if a == nil || a.hwnd == 0 || hwnd == 0 {
		return rect{}, false
	}
	var value rect
	if ok, _, _ := sidebarGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&value))); ok == 0 {
		return rect{}, false
	}
	// A RECT is two POINT structures, so MapWindowPoints converts both corners
	// from screen coordinates to the main client coordinate system in one call.
	sidebarMapWindowPoints.Call(0, a.hwnd, uintptr(unsafe.Pointer(&value)), 2)
	return rect{
		Left:   int32(a.unscale(int(value.Left))),
		Top:    int32(a.unscale(int(value.Top))),
		Right:  int32(a.unscale(int(value.Right))),
		Bottom: int32(a.unscale(int(value.Bottom))),
	}, true
}

func (a *app) transformSidebarContent(hwnd uintptr, oldLeft, oldRight, newLeft, newRight int) {
	if hwnd == 0 || oldRight <= oldLeft || newRight <= newLeft {
		return
	}
	r, ok := a.sidebarLogicalRect(hwnd)
	if !ok {
		return
	}
	x := int(r.Left)
	w := int(r.Right - r.Left)
	oldSpan := oldRight - oldLeft
	newSpan := newRight - newLeft
	nextX := newLeft + (x-oldLeft)*newSpan/oldSpan
	nextW := w * newSpan / oldSpan
	if nextW < 1 {
		nextW = 1
	}
	a.move(hwnd, nextX, int(r.Top), nextW, int(r.Bottom-r.Top))
}

func (a *app) resizeSidebarColumns() {
	resizeFile := func(list uintptr, remote bool) {
		if list == 0 {
			return
		}
		var client rect
		if ok, _, _ := getClientRect.Call(list, uintptr(unsafe.Pointer(&client))); ok == 0 {
			return
		}
		width := int(client.Right - client.Left)
		if width < a.scale(260) {
			return
		}
		if remote {
			parts := []int{35, 16, 14, 22, 13}
			for index, percent := range parts {
				sendMessageW.Call(list, lvmSetColumnWidth, uintptr(index), uintptr(width*percent/100))
			}
			return
		}
		parts := []int{42, 20, 16, 22}
		for index, percent := range parts {
			sendMessageW.Call(list, lvmSetColumnWidth, uintptr(index), uintptr(width*percent/100))
		}
	}
	resizeFile(a.localList, false)
	resizeFile(a.remoteList, true)

	if a.transferList != 0 {
		var client rect
		if ok, _, _ := getClientRect.Call(a.transferList, uintptr(unsafe.Pointer(&client))); ok != 0 {
			width := int(client.Right - client.Left)
			if width >= a.scale(360) {
				parts := []int{12, 25, 25, 25, 13}
				for index, percent := range parts {
					sendMessageW.Call(a.transferList, lvmSetColumnWidth, uintptr(index), uintptr(width*percent/100))
				}
			}
		}
	}
}

// applyApplicationSidebar turns the application-level navigation into one
// canonical left rail without rewriting the proven two-pane layout engine.
// The normal layout first computes a complete baseline. If the language combo
// is still in that baseline header position, this function applies one affine
// horizontal transform to the operational workspace and then anchors the rail.
// State-only refreshes see the already-left language selector and therefore do
// not compound the transform.
func (a *app) applyApplicationSidebar() {
	if a == nil || a.hwnd == 0 {
		return
	}
	diagnostics := a.ensureSidebarDiagnostics()
	words := nativeMenuWords(a.languageCode())
	a.setSidebarButtonVisual(a.siteManagerBtn, iconOpenLocal, words[5], buttonDefault)
	a.setSidebarButtonVisual(a.settingsBtn, iconSettings, a.tr("common.settings"), buttonSubtle)
	a.setSidebarButtonVisual(diagnostics, iconDiagnostics, words[8], buttonSubtle)
	a.setSidebarButtonVisual(a.aboutBtn, iconInfo, a.tr("common.about"), buttonSubtle)

	// Rail controls are always anchored explicitly, including state-only passes.
	a.move(a.languageCombo, applicationSidebarX, 54, applicationSidebarWidth, 29)
	y := 96
	for _, control := range []uintptr{a.siteManagerBtn, a.settingsBtn, diagnostics, a.aboutBtn} {
		a.move(control, applicationSidebarX, y, applicationSidebarWidth, applicationSidebarCardH)
		y += applicationSidebarCardH + applicationSidebarCardGap
	}

	// If the profile combo is already to the right of the rail then this layout
	// pass was transformed previously. Do not scale a transformed workspace twice.
	profileRect, ok := a.sidebarLogicalRect(a.profilesCombo)
	if ok && int(profileRect.Left) >= applicationContentLeft-2 {
		a.resizeSidebarColumns()
		return
	}

	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	oldLeft := premiumOuterGap
	oldRight := width - premiumOuterGap
	newLeft := applicationContentLeft
	newRight := oldRight
	if newRight-newLeft < 520 {
		return
	}

	for _, control := range []uintptr{
		a.protocol, a.host, a.port, a.user, a.pass, a.connect, a.disconnect,
		a.keyPath, a.chooseKey, a.passphrase,
		a.sectionLocal, a.sectionRemote,
		a.localPath, a.localUp, a.localChoose, a.localRefresh, a.localMkdir, a.localRename, a.localDelete, a.localList,
		a.remotePath, a.remoteUp, a.remoteRefresh, a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod, a.remoteList,
		a.upload, a.download,
		a.sectionTransfers, a.transferSummary, a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue, a.transferList,
		a.status, a.statusVersion,
	} {
		a.transformSidebarContent(control, oldLeft, oldRight, newLeft, newRight)
	}

	availableProfile := newRight - newLeft
	buttonW, gap := 128, 8
	profileW := availableProfile - 2*buttonW - 2*gap
	if profileW < 220 {
		profileW = 220
	}
	a.move(a.profilesCombo, newLeft, 51, profileW, 29)
	a.move(a.saveProfile, newLeft+profileW+gap, 51, buttonW, 29)
	a.move(a.removeProfile, newLeft+profileW+gap+buttonW+gap, 51, buttonW, 29)
	a.resizeSidebarColumns()
}
