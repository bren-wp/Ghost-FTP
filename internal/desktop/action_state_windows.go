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

	profileEditable := !a.connected && !a.connectionBusy && !a.profileMutationBusy
	setControlEnabled(a.siteManagerBtn, profileEditable)
	setControlEnabled(a.saveProfile, profileEditable)
	setControlEnabled(a.removeProfile, profileEditable && a.selectedProfileID != "")
	setControlEnabled(a.settingsBtn, !a.connectionBusy && !a.profileMutationBusy)

	comparisonActive := a.directoryComparisonActive()
	localRecursiveActive := a.recursiveSearchPaneActive(false)
	localSelected := validSelectionCount(a.localList, len(a.localItems))
	localReady := !localRecursiveActive && !comparisonActive
	localMutationReady := localReady && !a.localMutationBusy
	setControlEnabled(a.localMkdir, localMutationReady)
	setControlEnabled(a.localRename, localMutationReady && localSelected == 1)
	setControlEnabled(a.localDelete, localMutationReady && localSelected > 0)
	setControlEnabled(a.upload, localReady && a.connected && !a.connectionBusy && localSelected > 0)

	remoteRecursiveActive := a.recursiveSearchPaneActive(true)
	remoteSelected := validSelectionCount(a.remoteList, len(a.remoteItems))
	remoteReady := a.connected && !a.connectionBusy && !remoteRecursiveActive && !comparisonActive
	remoteMutationReady := remoteReady && !a.remoteMutationBusy && !remoteEditSessionBusy(a)
	setControlEnabled(a.remoteMkdir, remoteMutationReady)
	setControlEnabled(a.remoteRename, remoteMutationReady && remoteSelected == 1)
	setControlEnabled(a.remoteDelete, remoteMutationReady && remoteSelected > 0)
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
	setControlEnabled(a.remoteChmod, remoteMutationReady && chmodSelected > 0)

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
