//go:build windows

package desktop

import (
	"sync"
	"unsafe"
)

const (
	applicationSidebarX           = 14
	applicationSidebarWidth       = 166
	applicationContentLeft        = 204
	applicationSidebarCardH       = 46
	applicationSidebarCardGap     = 8
	applicationSidebarUtilityH    = 38
	applicationSidebarUtilityGap  = 7
	applicationSidebarLanguageH   = 29
	applicationSidebarPrimaryTop  = 64
	applicationSidebarBrandIcon   = 32
	applicationSidebarBrandGap    = 8
	applicationSidebarBottomInset = 14
)

var (
	sidebarFiles           sync.Map
	sidebarTransfers       sync.Map
	sidebarDiagnostics     sync.Map
	sidebarBookmarks       sync.Map
	sidebarGetWindowRect   = user32.NewProc("GetWindowRect")
	sidebarMapWindowPoints = user32.NewProc("MapWindowPoints")
	sidebarSetFocus        = user32.NewProc("SetFocus")
)

func (a *app) ensureSidebarButton(store *sync.Map, id int, label, icon string) uintptr {
	if a == nil || a.hwnd == 0 || store == nil {
		return 0
	}
	if value, ok := store.Load(a.hwnd); ok {
		if hwnd, ok := value.(uintptr); ok && hwnd != 0 {
			return hwnd
		}
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
		0, 0, 1, 1,
		a.hwnd, uintptr(id), hinst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	applyDarkControl(hwnd, "BUTTON")
	a.registerButton(hwnd, icon, label, buttonSubtle)
	store.Store(a.hwnd, hwnd)
	return hwnd
}

func (a *app) ensureSidebarFiles() uintptr {
	labels := navigationLabelsForLanguage(a.languageCode())
	return a.ensureSidebarButton(&sidebarFiles, idFilesNav, labels.Files, iconOpenLocal)
}

func (a *app) ensureSidebarTransfers() uintptr {
	labels := navigationLabelsForLanguage(a.languageCode())
	return a.ensureSidebarButton(&sidebarTransfers, idTransferQueueNav, labels.TransferQueue, iconSync)
}

func (a *app) ensureSidebarDiagnostics() uintptr {
	labels := navigationLabelsForLanguage(a.languageCode())
	return a.ensureSidebarButton(&sidebarDiagnostics, idDiagnostics, labels.Diagnostics, iconDiagnostics)
}

func (a *app) ensureSidebarBookmarks() uintptr {
	label := bookmarkWordsForLanguage(a.languageCode()).Title
	return a.ensureSidebarButton(&sidebarBookmarks, idBookmarks, label, iconOpenLocal)
}

func sidebarControl(store *sync.Map, owner uintptr) uintptr {
	if store == nil || owner == 0 {
		return 0
	}
	value, ok := store.Load(owner)
	if !ok {
		return 0
	}
	hwnd, _ := value.(uintptr)
	return hwnd
}

func (a *app) sidebarFilesButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarFiles, a.hwnd)
}

func (a *app) sidebarTransferButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarTransfers, a.hwnd)
}

func (a *app) sidebarDiagnosticsButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarDiagnostics, a.hwnd)
}

func (a *app) sidebarBookmarksButton() uintptr {
	if a == nil {
		return 0
	}
	return sidebarControl(&sidebarBookmarks, a.hwnd)
}

func (a *app) setSidebarButtonVisual(hwnd uintptr, icon, label string, variant buttonVariant) {
	if a == nil || hwnd == 0 {
		return
	}
	setText(hwnd, label)
	a.registerButtonVisual(hwnd, icon, label, variant, false)
	invalidateRect.Call(hwnd, 0, 0)
}

func (a *app) updateSidebarTransferBadge() {
	if a == nil {
		return
	}
	a.setButtonBadge(a.sidebarTransferButton(), relevantTransferCount(a.transferJobs))
}

func (a *app) focusFilesWorkspace() {
	if a == nil {
		return
	}
	target := a.localList
	if target == 0 {
		target = a.localPath
	}
	if target != 0 {
		sidebarSetFocus.Call(target)
	}
}

func (a *app) focusTransferQueue() {
	if a == nil || a.transferList == 0 {
		return
	}
	sidebarSetFocus.Call(a.transferList)
}

func (a *app) cleanupSidebarControls() {
	if a == nil || a.hwnd == 0 {
		return
	}
	for _, store := range []*sync.Map{&sidebarFiles, &sidebarTransfers, &sidebarBookmarks, &sidebarDiagnostics} {
		if hwnd := sidebarControl(store, a.hwnd); hwnd != 0 {
			delete(a.buttons, hwnd)
		}
		store.Delete(a.hwnd)
	}
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
	// File panes have one canonical width policy. The old sidebar-specific
	// percentages overrode resizeListColumns and made Permissions too narrow.
	// Fit from each ListView's actual client width so sidebar and non-sidebar
	// passes cannot disagree about the same columns.
	a.fitFileColumnsToWorkspace()

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

func (a *app) layoutSidebarRail(height int) {
	files := a.ensureSidebarFiles()
	transfers := a.ensureSidebarTransfers()
	diagnostics := a.ensureSidebarDiagnostics()
	bookmarks := a.ensureSidebarBookmarks()
	labels := navigationLabelsForLanguage(a.languageCode())

	a.setSidebarButtonVisual(files, iconOpenLocal, labels.Files, buttonNavActive)
	a.setSidebarButtonVisual(a.siteManagerBtn, iconConnect, labels.Connections, buttonDefault)
	a.setSidebarButtonVisual(transfers, iconSync, labels.TransferQueue, buttonDefault)
	a.setSidebarButtonVisual(a.settingsBtn, iconSettings, a.tr("common.settings"), buttonDefault)
	a.setSidebarButtonVisual(bookmarks, iconOpenLocal, bookmarkWordsForLanguage(a.languageCode()).Title, buttonSubtle)
	a.setSidebarButtonVisual(diagnostics, iconDiagnostics, labels.Diagnostics, buttonSubtle)
	a.setSidebarButtonVisual(a.aboutBtn, iconInfo, a.tr("common.about"), buttonSubtle)
	a.updateSidebarTransferBadge()

	logo := a.ensureBrandLogo()
	if logo != 0 {
		a.move(logo, applicationSidebarX+2, 14, applicationSidebarBrandIcon, applicationSidebarBrandIcon)
	}
	titleX := applicationSidebarX + applicationSidebarBrandIcon + applicationSidebarBrandGap + 2
	a.move(a.brandTitle, titleX, 13, applicationSidebarWidth-(titleX-applicationSidebarX), 34)
	// The master desktop references keep the rail brand intentionally compact.
	// The descriptive subtitle remains in About rather than competing with
	// operational navigation and file content.
	showControls(false, a.brandSubtitle)

	y := applicationSidebarPrimaryTop
	for _, control := range []uintptr{files, a.siteManagerBtn, transfers, a.settingsBtn} {
		a.move(control, applicationSidebarX, y, applicationSidebarWidth, applicationSidebarCardH)
		y += applicationSidebarCardH + applicationSidebarCardGap
	}

	utilityBlockH := 3*applicationSidebarUtilityH + 2*applicationSidebarUtilityGap + applicationSidebarLanguageH + applicationSidebarUtilityGap
	utilityY := height - applicationSidebarBottomInset - utilityBlockH
	if utilityY < y+18 {
		utilityY = y + 18
	}
	for _, control := range []uintptr{bookmarks, diagnostics, a.aboutBtn} {
		a.move(control, applicationSidebarX, utilityY, applicationSidebarWidth, applicationSidebarUtilityH)
		utilityY += applicationSidebarUtilityH + applicationSidebarUtilityGap
	}
	a.move(a.languageCombo, applicationSidebarX, utilityY, applicationSidebarWidth, applicationSidebarLanguageH)
}

// applyApplicationSidebar turns application-level navigation into one canonical
// left rail without rewriting the proven two-pane layout engine. The baseline
// layout computes the operational workspace first, then this function applies
// one affine horizontal transform and anchors the rail. State-only refreshes
// detect an already-transformed profile control and never compound the transform.
func (a *app) applyApplicationSidebar() {
	if a == nil || a.hwnd == 0 {
		return
	}

	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	a.layoutSidebarRail(height)

	// If the profile combo is already to the right of the rail then this layout
	// pass was transformed previously. Do not scale a transformed workspace twice.
	profileRect, ok := a.sidebarLogicalRect(a.profilesCombo)
	if ok && int(profileRect.Left) >= applicationContentLeft-2 {
		a.resizeSidebarColumns()
		return
	}

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

	// The saved-connection picker becomes the first content-row control, matching
	// the master reference's connection bar while save/delete remain real profile
	// operations rather than decorative buttons.
	availableProfile := newRight - newLeft
	buttonW, gap := 126, 8
	profileW := availableProfile - 2*buttonW - 2*gap
	if profileW < 220 {
		profileW = 220
	}
	a.move(a.profilesCombo, newLeft, 13, profileW, 31)
	a.move(a.saveProfile, newLeft+profileW+gap, 13, buttonW, 31)
	a.move(a.removeProfile, newLeft+profileW+gap+buttonW+gap, 13, buttonW, 31)
	a.resizeSidebarColumns()
}
