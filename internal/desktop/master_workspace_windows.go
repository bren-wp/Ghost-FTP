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
	// The approved master workspace exposes search directly in the action bar.
	// Reuse Ghost FTP's real per-pane filter rather than introducing a decorative
	// search field that cannot affect the file model.
	if a.lastFilePaneRemote && a.connected && !a.connectionBusy {
		a.openFileFilter(a.remoteList)
		return
	}
	a.openFileFilter(a.localList)
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
		a.masterMore = a.ensureMasterWorkspaceButton(hinst, idWorkspaceMore, "Search", iconSearch, buttonSubtle)
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
	label := a.tr("common.connect")
	icon := iconConnect
	variant := buttonAccent
	if a.connectionBusy {
		label = a.tr("common.cancel")
		icon = iconCancel
		variant = buttonDanger
	}
	a.setButtonLabel(a.connect, label)
	a.registerButtonVisual(a.connect, icon, label, variant, false)
	setControlEnabled(a.connect, !a.profileMutationBusy && !a.connected)
	if a.disconnect != 0 {
		disconnectLabel := a.tr("common.disconnect")
		a.setButtonLabel(a.disconnect, disconnectLabel)
		a.registerButtonVisual(a.disconnect, iconDisconnect, disconnectLabel, buttonSubtle, false)
		setControlEnabled(a.disconnect, a.connected || a.connectionBusy)
		invalidateRect.Call(a.disconnect, 0, 0)
	}
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
	if contentWidth < 720 {
		return
	}

	// Credentials are edited in Connections. The main workspace follows the
	// approved reference: one compact site/status strip, a dedicated action bar,
	// two equal file panes and a persistent transfer queue.
	showControls(false,
		a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.chooseKey, a.passphrase,
		a.saveProfile, a.removeProfile,
	)
	showControls(true, a.profilesCombo, a.connectionBadge, a.connect, a.disconnect)

	// Top site/status strip.
	topY, topH := 14, 36
	badgeW := 176
	profileW := contentWidth - badgeW - 12
	if profileW > 720 {
		profileW = 720
	}
	if profileW < 300 {
		profileW = 300
	}
	a.move(a.profilesCombo, contentLeft, topY, profileW, topH)
	a.move(a.connectionBadge, contentRight-badgeW, topY+7, badgeW, 22)

	// Primary action bar mirrors the approved mockup: Connect, Disconnect,
	// New Folder, Upload, Download, Refresh and Search.
	toolbarY, toolbarH, gap := 62, 40, 8
	fixed := []struct {
		control uintptr
		width   int
	}{
		{a.connect, 118},
		{a.disconnect, 118},
		{a.masterNewFolder, 126},
		{a.upload, 108},
		{a.download, 108},
		{a.masterRefresh, 104},
	}
	x := contentLeft
	for _, item := range fixed {
		a.move(item.control, x, toolbarY, item.width, toolbarH)
		x += item.width + gap
	}
	searchW := contentRight - x
	if searchW < 150 {
		searchW = 150
	}
	a.move(a.masterMore, x, toolbarY, searchW, toolbarH)

	// Main file area.
	paneGap := 14
	paneW := (contentWidth - paneGap) / 2
	leftX := contentLeft
	rightX := leftX + paneW + paneGap
	sectionY, pathY, actionY := 120, 148, 190
	a.move(a.sectionLocal, leftX, sectionY, paneW, 22)
	a.move(a.sectionRemote, rightX, sectionY, paneW, 22)

	// Local breadcrumb controls: back / forward / up / path / choose.
	miniW, miniGap := 36, 6
	lx := leftX
	for _, control := range []uintptr{a.masterBack, a.masterForward, a.localUp} {
		a.move(control, lx, pathY, miniW, 34)
		lx += miniW + miniGap
	}
	chooseW := 76
	localPathW := leftX + paneW - lx - chooseW - miniGap
	if localPathW < 120 {
		localPathW = 120
	}
	a.move(a.localPath, lx, pathY, localPathW, 34)
	a.move(a.localChoose, lx+localPathW+miniGap, pathY, chooseW, 34)
	showControls(false, a.localRefresh)

	// Remote breadcrumb controls: up / path / refresh.
	rx := rightX
	a.move(a.remoteUp, rx, pathY, miniW, 34)
	rx += miniW + miniGap
	remoteRefreshW := 78
	remotePathW := rightX + paneW - rx - remoteRefreshW - miniGap
	if remotePathW < 140 {
		remotePathW = 140
	}
	a.move(a.remotePath, rx, pathY, remotePathW, 34)
	a.move(a.remoteRefresh, rx+remotePathW+miniGap, pathY, remoteRefreshW, 34)

	// Keep advanced real file operations visible in a compact utility row.
	actionGap := 6
	localActions := []uintptr{a.localMkdir, a.localRename, a.localDelete}
	localActionW := (paneW - actionGap*(len(localActions)-1)) / len(localActions)
	lx = leftX
	for _, control := range localActions {
		a.move(control, lx, actionY, localActionW, 30)
		lx += localActionW + actionGap
	}
	remoteActions := []uintptr{a.remoteMkdir, a.remoteRename, a.remoteDelete, remoteEditButton(a), a.remoteChmod}
	remoteActionW := (paneW - actionGap*(len(remoteActions)-1)) / len(remoteActions)
	rx = rightX
	for _, control := range remoteActions {
		a.move(control, rx, actionY, remoteActionW, 30)
		rx += remoteActionW + actionGap
	}

	statusY, _ := statusBandGeometry(height)
	queueH := clampInt(height/5, 128, 184)
	queueY := statusY - queueH - 10
	queueButtonsY := queueY - 38
	queueLabelY := queueButtonsY - 25
	listY := actionY + 36
	listBottom := queueLabelY - 12
	listH := listBottom - listY
	if listH < 150 {
		listH = 150
	}
	a.move(a.localList, leftX, listY, paneW, listH)
	a.move(a.remoteList, rightX, listY, paneW, listH)

	// Persistent transfer queue with real lifecycle actions.
	a.move(a.sectionTransfers, contentLeft, queueLabelY, 190, 20)
	a.move(a.transferSummary, contentLeft+190, queueLabelY, clampInt(contentWidth-190, 260, 620), 20)
	queueControls := []struct {
		control uintptr
		width   int
	}{
		{a.pauseQueue, 104},
		{a.resumeQueue, 104},
		{a.cancelJob, 98},
		{a.retryJob, 98},
		{a.clearQueue, 142},
	}
	qx := contentRight
	for i := len(queueControls) - 1; i >= 0; i-- {
		qx -= queueControls[i].width
		a.move(queueControls[i].control, qx, queueButtonsY, queueControls[i].width, 31)
		qx -= 7
	}
	a.move(a.transferList, contentLeft, queueY, contentWidth, queueH)
	a.move(a.status, contentLeft, statusY, contentWidth-265, statusBandHeight)
	a.move(a.statusVersion, contentRight-250, statusY, 250, statusBandHeight)

	a.updateMasterToolbarState()
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
