//go:build windows

package desktop

import (
	"path/filepath"
	"strings"
	"unsafe"
)

type workspaceHistoryEntry struct {
	Remote bool
	Path   string
}

const workspaceHistoryLimit = 64

func normalizeWorkspaceHistoryPath(remote bool, value string) string {
	value = strings.TrimSpace(value)
	if remote {
		if value == "" {
			return "/"
		}
		return cleanRemote(value)
	}
	if value == "" {
		return ""
	}
	return filepath.Clean(value)
}

func sameWorkspaceHistoryEntry(left, right workspaceHistoryEntry) bool {
	return left.Remote == right.Remote &&
		normalizeWorkspaceHistoryPath(left.Remote, left.Path) ==
			normalizeWorkspaceHistoryPath(right.Remote, right.Path)
}

func appendWorkspaceHistory(history []workspaceHistoryEntry, entry workspaceHistoryEntry) []workspaceHistoryEntry {
	entry.Path = normalizeWorkspaceHistoryPath(entry.Remote, entry.Path)
	if entry.Path == "" {
		return history
	}
	if len(history) > 0 && sameWorkspaceHistoryEntry(history[len(history)-1], entry) {
		return history
	}
	history = append(history, entry)
	if len(history) > workspaceHistoryLimit {
		copy(history, history[len(history)-workspaceHistoryLimit:])
		history = history[:workspaceHistoryLimit]
	}
	return history
}

func (a *app) recordWorkspaceNavigation(remote bool, previous, next string) {
	if a == nil {
		return
	}
	oldEntry := workspaceHistoryEntry{Remote: remote, Path: normalizeWorkspaceHistoryPath(remote, previous)}
	newEntry := workspaceHistoryEntry{Remote: remote, Path: normalizeWorkspaceHistoryPath(remote, next)}
	if oldEntry.Path == "" || sameWorkspaceHistoryEntry(oldEntry, newEntry) {
		if a.workspaceReplayActive && sameWorkspaceHistoryEntry(a.workspaceReplayTarget, newEntry) {
			a.workspaceReplayActive = false
		}
		a.updateMasterToolbarState()
		return
	}
	if a.workspaceReplayActive && sameWorkspaceHistoryEntry(a.workspaceReplayTarget, newEntry) {
		a.workspaceReplayActive = false
		a.updateMasterToolbarState()
		return
	}
	a.workspaceBackHistory = appendWorkspaceHistory(a.workspaceBackHistory, oldEntry)
	a.workspaceForwardHistory = nil
	a.updateMasterToolbarState()
}

func (a *app) failWorkspaceReplay(remote bool, target string) {
	if a == nil || !a.workspaceReplayActive {
		return
	}
	entry := workspaceHistoryEntry{Remote: remote, Path: normalizeWorkspaceHistoryPath(remote, target)}
	if !sameWorkspaceHistoryEntry(a.workspaceReplayTarget, entry) {
		return
	}
	a.workspaceBackHistory = appendWorkspaceHistory(a.workspaceBackHistory, entry)
	if len(a.workspaceForwardHistory) > 0 {
		a.workspaceForwardHistory = a.workspaceForwardHistory[:len(a.workspaceForwardHistory)-1]
	}
	a.workspaceReplayActive = false
	a.updateMasterToolbarState()
}

func (a *app) currentWorkspaceHistoryEntry(remote bool) workspaceHistoryEntry {
	if remote {
		return workspaceHistoryEntry{Remote: true, Path: normalizeWorkspaceHistoryPath(true, a.remoteCurrent)}
	}
	return workspaceHistoryEntry{Path: normalizeWorkspaceHistoryPath(false, a.localCurrent)}
}

func (a *app) navigateWorkspaceHistory(back bool) {
	if a == nil || a.workspaceReplayActive {
		return
	}
	var source, destination *[]workspaceHistoryEntry
	if back {
		source = &a.workspaceBackHistory
		destination = &a.workspaceForwardHistory
	} else {
		source = &a.workspaceForwardHistory
		destination = &a.workspaceBackHistory
	}
	if len(*source) == 0 {
		return
	}
	target := (*source)[len(*source)-1]
	*source = (*source)[:len(*source)-1]
	current := a.currentWorkspaceHistoryEntry(target.Remote)
	if current.Path != "" {
		*destination = appendWorkspaceHistory(*destination, current)
	}
	a.workspaceReplayActive = true
	a.workspaceReplayTarget = target
	a.updateMasterToolbarState()
	if target.Remote {
		if !a.connected || a.connectionBusy {
			a.failWorkspaceReplay(true, target.Path)
			return
		}
		a.lastFilePaneRemote = true
		a.refreshRemote(target.Path)
		return
	}
	a.lastFilePaneRemote = false
	a.refreshLocal(target.Path)
}

func (a *app) masterNewFolderAction() {
	if a == nil {
		return
	}
	if a.lastFilePaneRemote && a.connected && !a.connectionBusy && !a.remoteMutationBusy {
		a.remoteMkdirAction()
		return
	}
	a.localMkdirAction()
}

func (a *app) masterMoreAction() {
	if a == nil {
		return
	}
	// Connection info is the canonical secondary utility surface. More never
	// fabricates operations or hidden state; advanced per-pane controls remain
	// available below the master toolbar.
	a.showDiagnostics()
}

func (a *app) masterConnectAction() {
	if a == nil {
		return
	}
	if a.connectionBusy || a.connected {
		a.disconnectNow()
		return
	}
	a.openSiteManager()
}

func (a *app) ensureMasterWorkspaceButton(hinst uintptr, id int, label, icon string, variant buttonVariant) uintptr {
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
	return a.registerButton(hwnd, icon, label, variant)
}

func (a *app) ensureMasterWorkspaceControls() {
	if a == nil || a.hwnd == 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	if a.masterBack == 0 {
		a.masterBack = a.ensureMasterWorkspaceButton(hinst, idWorkspaceBack, "Back", iconBack, buttonSubtle)
		a.masterForward = a.ensureMasterWorkspaceButton(hinst, idWorkspaceForward, "Forward", iconForward, buttonSubtle)
		a.masterRefresh = a.ensureMasterWorkspaceButton(hinst, idRefreshAll, a.tr("common.refresh"), iconRefresh, buttonSubtle)
		a.masterNewFolder = a.ensureMasterWorkspaceButton(hinst, idWorkspaceNewFolder, a.tr("common.new_folder"), iconNewFolder, buttonDefault)
		a.masterBookmarks = a.ensureMasterWorkspaceButton(hinst, idBookmarks, bookmarkWordsForLanguage(a.languageCode()).Title, iconOpenLocal, buttonDefault)
		a.masterMore = a.ensureMasterWorkspaceButton(hinst, idWorkspaceMore, "More", iconMore, buttonSubtle)
	}
	a.setButtonLabel(a.masterRefresh, a.tr("common.refresh"))
	a.setButtonLabel(a.masterNewFolder, a.tr("common.new_folder"))
	a.setButtonLabel(a.masterBookmarks, bookmarkWordsForLanguage(a.languageCode()).Title)
	a.updateMasterToolbarState()
}

func (a *app) updateMasterConnectVisual() {
	if a == nil || a.connect == 0 {
		return
	}
	label := "Quick Connect"
	icon := iconConnect
	variant := buttonAccent
	if a.connectionBusy {
		label = a.tr("common.cancel")
		icon = iconCancel
		variant = buttonDanger
	} else if a.connected {
		label = a.tr("common.disconnect")
		icon = iconDisconnect
		variant = buttonDanger
	}
	a.setButtonLabel(a.connect, label)
	a.registerButtonVisual(a.connect, icon, label, variant, false)
	setControlEnabled(a.connect, !a.profileMutationBusy)
	invalidateRect.Call(a.connect, 0, 0)
}

func (a *app) updateMasterToolbarState() {
	if a == nil {
		return
	}
	setControlEnabled(a.masterBack, len(a.workspaceBackHistory) > 0 && !a.workspaceReplayActive)
	setControlEnabled(a.masterForward, len(a.workspaceForwardHistory) > 0 && !a.workspaceReplayActive)
	setControlEnabled(a.masterRefresh, !a.closing)
	setControlEnabled(a.masterNewFolder, !a.closing && !a.localMutationBusy &&
		(!a.lastFilePaneRemote || (a.connected && !a.connectionBusy && !a.remoteMutationBusy)))
	setControlEnabled(a.masterBookmarks, !a.closing)
	setControlEnabled(a.masterMore, !a.closing)
	a.updateMasterConnectVisual()
}

func (a *app) layoutMasterWorkspaceChrome() {
	if a == nil || a.hwnd == 0 {
		return
	}
	a.ensureMasterWorkspaceControls()

	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	contentLeft := applicationContentLeft
	contentRight := width - premiumOuterGap
	contentWidth := contentRight - contentLeft
	if contentWidth < 520 {
		return
	}

	// Connection credentials are owned by the Connections window. Keeping their
	// hidden native controls preserves the existing engine/profile binding while
	// the Files workspace follows the supplied master composition.
	showControls(false,
		a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.chooseKey, a.passphrase,
		a.saveProfile, a.removeProfile, a.disconnect,
	)
	showControls(true, a.profilesCombo, a.connectionBadge, a.connect)

	topY, rowH, gap := 13, 34, 8
	badgeW, connectW := 122, 134
	profileW := contentWidth - badgeW - connectW - 2*gap
	if profileW < 240 {
		profileW = 240
	}
	a.move(a.profilesCombo, contentLeft, topY, profileW, rowH)
	badgeX := contentLeft + profileW + gap
	a.move(a.connectionBadge, badgeX, topY+6, badgeW, 22)
	a.move(a.connect, badgeX+badgeW+gap, topY, connectW, rowH)

	toolbarY, toolbarH := 56, 38
	controls := []uintptr{
		a.masterBack, a.masterForward, a.masterRefresh, a.masterNewFolder,
		a.upload, a.download, a.masterBookmarks, a.masterMore,
	}
	toolbarGap := 7
	buttonW := (contentWidth - toolbarGap*(len(controls)-1)) / len(controls)
	if buttonW < 76 {
		buttonW = 76
	}
	x := contentLeft
	for _, control := range controls {
		a.move(control, x, toolbarY, buttonW, toolbarH)
		x += buttonW + toolbarGap
	}

	// Remove the legacy center transfer column and use two equal master panes.
	paneGap := 12
	paneW := (contentWidth - paneGap) / 2
	leftX := contentLeft
	rightX := contentLeft + paneW + paneGap
	sectionY, pathY, actionY := 108, 132, 169
	pathButtonGap := 6
	pathButtonW := 78
	localPathW := paneW - 3*pathButtonW - 3*pathButtonGap
	if localPathW < 120 {
		localPathW = 120
	}
	remotePathW := paneW - 2*pathButtonW - 2*pathButtonGap
	if remotePathW < 120 {
		remotePathW = 120
	}

	a.move(a.sectionLocal, leftX, sectionY, paneW, 20)
	a.move(a.sectionRemote, rightX, sectionY, paneW, 20)

	a.move(a.localPath, leftX, pathY, localPathW, 29)
	lx := leftX + localPathW + pathButtonGap
	for _, control := range []uintptr{a.localUp, a.localChoose, a.localRefresh} {
		a.move(control, lx, pathY, pathButtonW, 29)
		lx += pathButtonW + pathButtonGap
	}

	a.move(a.remotePath, rightX, pathY, remotePathW, 29)
	rx := rightX + remotePathW + pathButtonGap
	for _, control := range []uintptr{a.remoteUp, a.remoteRefresh} {
		a.move(control, rx, pathY, pathButtonW, 29)
		rx += pathButtonW + pathButtonGap
	}

	actionGap := 6
	localActionW := (paneW - 2*actionGap) / 3
	lx = leftX
	for _, control := range []uintptr{a.localMkdir, a.localRename, a.localDelete} {
		a.move(control, lx, actionY, localActionW, 29)
		lx += localActionW + actionGap
	}
	remoteControls := []uintptr{a.remoteMkdir, a.remoteRename, a.remoteDelete, remoteEditButton(a), a.remoteChmod}
	remoteActionW := (paneW - actionGap*(len(remoteControls)-1)) / len(remoteControls)
	rx = rightX
	for _, control := range remoteControls {
		a.move(control, rx, actionY, remoteActionW, 29)
		rx += remoteActionW + actionGap
	}

	statusY, _ := statusBandGeometry(height)
	queueH := clampInt(height/6, 105, 165)
	queueY := statusY - queueH - 9
	queueButtonsY := queueY - 38
	queueLabelY := queueButtonsY - 23
	listY := actionY + 29 + 44
	listBottom := queueLabelY - 10
	listH := listBottom - listY
	if listH < 120 {
		listH = 120
	}
	a.move(a.localList, leftX, listY, paneW, listH)
	a.move(a.remoteList, rightX, listY, paneW, listH)

	a.move(a.sectionTransfers, contentLeft, queueLabelY, 180, 18)
	a.move(a.transferSummary, contentLeft+180, queueLabelY, clampInt(contentWidth-180, 260, 640), 18)
	qx := contentLeft
	queueWidths := []int{112, 112, 104, 104, 150}
	for i, control := range []uintptr{a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue} {
		a.move(control, qx, queueButtonsY, queueWidths[i], 31)
		qx += queueWidths[i] + 7
	}
	a.move(a.transferList, contentLeft, queueY, contentWidth, queueH)
	a.move(a.status, contentLeft, statusY, contentWidth-250, statusBandHeight)
	a.move(a.statusVersion, contentRight-238, statusY, 238, statusBandHeight)
}

func (a *app) cleanupMasterWorkspaceControls() {
	if a == nil {
		return
	}
	for _, control := range []uintptr{
		a.masterBack, a.masterForward, a.masterRefresh,
		a.masterNewFolder, a.masterBookmarks, a.masterMore,
	} {
		delete(a.buttons, control)
	}
}
