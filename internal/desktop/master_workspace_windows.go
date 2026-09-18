//go:build windows

package desktop

import (
	"path/filepath"
	"strings"
	"unsafe"
)

const (
	masterMoreLocalChoose = 9001 + iota
	masterMoreLocalUp
	masterMoreLocalFilter
	masterMoreLocalSearch
	masterMoreLocalRename
	masterMoreLocalDelete
	masterMoreRemoteUp
	masterMoreRemoteFilter
	masterMoreRemoteSearch
	masterMoreRemoteRename
	masterMoreRemoteDelete
	masterMoreRemotePermissions
	masterMoreRemoteEdit
	masterMoreCompare
	masterMoreConnectionInfo
)

const (
	masterMFString    = 0x0000
	masterMFSeparator = 0x0800
	masterTPMReturn   = 0x0100
	masterTPMRight    = 0x0002
)

var (
	masterCreatePopupMenu = user32.NewProc("CreatePopupMenu")
	masterAppendMenuW      = user32.NewProc("AppendMenuW")
	masterTrackPopupMenu   = user32.NewProc("TrackPopupMenu")
	masterDestroyMenu      = user32.NewProc("DestroyMenu")
	masterGetWindowRect    = user32.NewProc("GetWindowRect")
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
	if a == nil || a.hwnd == 0 || a.masterMore == 0 {
		return
	}
	menu, _, _ := masterCreatePopupMenu.Call()
	if menu == 0 {
		a.showDiagnostics()
		return
	}
	defer masterDestroyMenu.Call(menu)

	appendItem := func(id int, label string) {
		masterAppendMenuW.Call(menu, masterMFString, uintptr(id), uintptr(unsafe.Pointer(wstr(label))))
	}
	appendSeparator := func() {
		masterAppendMenuW.Call(menu, masterMFSeparator, 0, 0)
	}

	appendItem(masterMoreLocalChoose, "Local: Choose folder")
	appendItem(masterMoreLocalUp, "Local: Up")
	appendItem(masterMoreLocalFilter, "Local: Filter")
	appendItem(masterMoreLocalSearch, "Local: Recursive search")
	appendItem(masterMoreLocalRename, "Local: Rename selected")
	appendItem(masterMoreLocalDelete, "Local: Delete selected")
	appendSeparator()
	appendItem(masterMoreRemoteUp, "Remote: Up")
	appendItem(masterMoreRemoteFilter, "Remote: Filter")
	appendItem(masterMoreRemoteSearch, "Remote: Recursive search")
	appendItem(masterMoreRemoteRename, "Remote: Rename selected")
	appendItem(masterMoreRemoteDelete, "Remote: Delete selected")
	appendItem(masterMoreRemotePermissions, "Remote: Permissions")
	appendItem(masterMoreRemoteEdit, "Remote Edit")
	appendSeparator()
	appendItem(masterMoreCompare, "Compare local and remote folders")
	appendItem(masterMoreConnectionInfo, "Connection info")

	var bounds rect
	if ok, _, _ := masterGetWindowRect.Call(a.masterMore, uintptr(unsafe.Pointer(&bounds))); ok == 0 {
		a.showDiagnostics()
		return
	}
	command, _, _ := masterTrackPopupMenu.Call(
		menu,
		masterTPMReturn|masterTPMRight,
		uintptr(bounds.Left),
		uintptr(bounds.Bottom),
		0,
		a.hwnd,
		0,
	)
	switch int(command) {
	case masterMoreLocalChoose:
		a.chooseLocalDirectory()
	case masterMoreLocalUp:
		a.refreshLocal(filepath.Dir(getText(a.localPath)))
	case masterMoreLocalFilter:
		a.localFilterAction()
	case masterMoreLocalSearch:
		a.recursiveSearchCommand(false)
	case masterMoreLocalRename:
		a.localRenameAction()
	case masterMoreLocalDelete:
		a.localDeleteAction()
	case masterMoreRemoteUp:
		a.remoteUpOne()
	case masterMoreRemoteFilter:
		a.remoteFilterAction()
	case masterMoreRemoteSearch:
		a.recursiveSearchCommand(true)
	case masterMoreRemoteRename:
		a.remoteRenameAction()
	case masterMoreRemoteDelete:
		a.remoteDeleteAction()
	case masterMoreRemotePermissions:
		a.remoteChmodAction()
	case masterMoreRemoteEdit:
		a.remoteEditAction()
	case masterMoreCompare:
		a.directoryComparisonCommand()
	case masterMoreConnectionInfo:
		a.showDiagnostics()
	}
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

func (a *app) applyMasterWorkspaceLabels() {
	if a == nil {
		return
	}
	// The supplied master screenshots use these canonical English workspace
	// nouns. Keep non-English catalog strings intact; English gets the exact
	// product terminology used across Windows, Linux, macOS and Android.
	if a.languageCode() != "en" {
		return
	}
	setText(a.sectionLocal, "Local Files")
	setText(a.sectionRemote, "Remote Files")
	setText(a.sectionTransfers, "Transfer Queue")
}

func (a *app) layoutMasterWorkspaceChrome() {
	if a == nil || a.hwnd == 0 {
		return
	}
	a.ensureMasterWorkspaceControls()
	a.applyMasterWorkspaceLabels()
	if a.font != 0 {
		sendMessageW.Call(a.sectionLocal, wmSetFont, a.font, 1)
		sendMessageW.Call(a.sectionRemote, wmSetFont, a.font, 1)
		sendMessageW.Call(a.sectionTransfers, wmSetFont, a.font, 1)
	}

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
	// The master Files reference keeps per-pane mutation/navigation operations
	// out of permanent rows. They remain fully available through the real More
	// menu and master toolbar, so hiding these duplicate controls removes visual
	// clutter without removing functionality.
	showControls(false,
		a.localUp, a.localChoose, a.localRefresh,
		a.localMkdir, a.localRename, a.localDelete,
		a.remoteUp, a.remoteRefresh,
		a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod,
		remoteEditButton(a),
	)

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
	sectionY, pathY, actionY := 108, 136, 174
	a.move(a.sectionLocal, leftX, sectionY, paneW, 24)
	a.move(a.sectionRemote, rightX, sectionY, paneW, 24)

	a.move(a.localPath, leftX, pathY, paneW, 29)
	a.move(a.remotePath, rightX, pathY, paneW, 29)

	// Preserve deterministic hidden-control bounds for command ownership and
	// accessibility bookkeeping; these controls are not part of the visible
	// master Files composition.
	for _, control := range []uintptr{
		a.localUp, a.localChoose, a.localRefresh,
		a.localMkdir, a.localRename, a.localDelete,
		a.remoteUp, a.remoteRefresh,
		a.remoteMkdir, a.remoteRename, a.remoteDelete, remoteEditButton(a), a.remoteChmod,
	} {
		a.move(control, contentRight-1, actionY, 1, 1)
	}

	statusY, _ := statusBandGeometry(height)
	queueH := clampInt(height/6, 105, 165)
	queueY := statusY - queueH - 9
	queueButtonsY := queueY - 38
	queueLabelY := queueButtonsY - 23
	listY := pathY + 29 + 10
	listBottom := queueLabelY - 10
	listH := listBottom - listY
	if listH < 120 {
		listH = 120
	}
	a.move(a.localList, leftX, listY, paneW, listH)
	a.move(a.remoteList, rightX, listY, paneW, listH)

	a.move(a.sectionTransfers, contentLeft, queueLabelY-2, 190, 22)
	a.move(a.transferSummary, contentLeft+190, queueLabelY, clampInt(contentWidth-190, 260, 620), 18)
	qx := contentLeft
	queueWidths := []int{104, 104, 96, 96}
	for i, control := range []uintptr{a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob} {
		a.move(control, qx, queueButtonsY, queueWidths[i], 31)
		qx += queueWidths[i] + 7
	}
	clearW := 150
	a.move(a.clearQueue, contentRight-clearW, queueButtonsY, clearW, 31)
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
