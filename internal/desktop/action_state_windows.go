//go:build windows

package desktop

func setControlEnabled(hwnd uintptr, enabled bool) {
	if hwnd == 0 {
		return
	}
	value := uintptr(0)
	if enabled {
		value = 1
	}
	enableWindow.Call(hwnd, value)
}

func validSelectionCount(list uintptr, itemCount int) int {
	count := 0
	for _, index := range selectedIndices(list) {
		if index >= 0 && index < itemCount {
			count++
		}
	}
	return count
}

func (a *app) updateActionControls() {
	if a == nil || a.closing {
		return
	}

	profileEditable := !a.connected && !a.connectionBusy
	setControlEnabled(a.siteManagerBtn, profileEditable)
	setControlEnabled(a.saveProfile, profileEditable)
	setControlEnabled(a.removeProfile, profileEditable && a.selectedProfileID != "")
	setControlEnabled(a.settingsBtn, !a.connectionBusy)

	localSearch := a.recursiveSearchPaneActive(false)
	localSelected := validSelectionCount(a.localList, len(a.localItems))
	setControlEnabled(a.localMkdir, !localSearch)
	setControlEnabled(a.localRename, !localSearch && localSelected == 1)
	setControlEnabled(a.localDelete, !localSearch && localSelected > 0)
	setControlEnabled(a.upload, !localSearch && a.connected && !a.connectionBusy && localSelected > 0)

	remoteSearch := a.recursiveSearchPaneActive(true)
	remoteSelected := validSelectionCount(a.remoteList, len(a.remoteItems))
	remoteReady := a.connected && !a.connectionBusy && !remoteSearch
	setControlEnabled(a.remoteMkdir, remoteReady)
	setControlEnabled(a.remoteRename, remoteReady && remoteSelected == 1)
	setControlEnabled(a.remoteDelete, remoteReady && remoteSelected > 0)
	setControlEnabled(a.download, remoteReady && remoteSelected > 0)
	setControlEnabled(remoteEditButton(a), remoteReady && a.remoteEditSelectionReady())

	chmodSelected := 0
	if remoteReady {
		for _, index := range selectedIndices(a.remoteList) {
			if index >= 0 && index < len(a.remoteItems) && !a.remoteItems[index].IsSymlink {
				chmodSelected++
			}
		}
	}
	setControlEnabled(a.remoteChmod, remoteReady && chmodSelected > 0)

	selectedTransfers := selectedIndices(a.transferList)
	transferState := deriveTransferActionState(a.transferJobs, selectedTransfers, a.connected && !a.connectionBusy, a.queuePaused)
	priorityState := deriveQueuePriorityState(a.transferJobs, selectedTransfers)
	if a.connectionBusy {
		transferState.Pause = false
		transferState.Resume = false
		transferState.Cancel = false
		transferState.Retry = false
		priorityState.MoveUp = false
		priorityState.MoveDown = false
	}
	setControlEnabled(a.pauseQueue, transferState.Pause)
	setControlEnabled(a.resumeQueue, transferState.Resume)
	setControlEnabled(a.cancelJob, transferState.Cancel)
	setControlEnabled(a.retryJob, transferState.Retry)
	setControlEnabled(a.clearQueue, transferState.Clear && !a.connectionBusy)
	a.updateQueuePriorityControls(priorityState)

	a.refineWorkspaceLayout()
}
