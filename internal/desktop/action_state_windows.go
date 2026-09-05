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

	a.ensureSiteManagerButton()
	profileEditable := !a.connected && !a.connectionBusy
	setControlEnabled(a.siteManagerBtn, profileEditable)
	setControlEnabled(a.saveProfile, profileEditable)
	setControlEnabled(a.removeProfile, profileEditable && a.selectedProfileID != "")
	setControlEnabled(a.settingsBtn, !a.connectionBusy)

	localSelected := validSelectionCount(a.localList, len(a.localItems))
	setControlEnabled(a.localRename, localSelected == 1)
	setControlEnabled(a.localDelete, localSelected > 0)
	setControlEnabled(a.upload, a.connected && !a.connectionBusy && localSelected > 0)

	remoteSelected := validSelectionCount(a.remoteList, len(a.remoteItems))
	remoteReady := a.connected && !a.connectionBusy
	setControlEnabled(a.remoteMkdir, remoteReady)
	setControlEnabled(a.remoteRename, remoteReady && remoteSelected == 1)
	setControlEnabled(a.remoteDelete, remoteReady && remoteSelected > 0)
	setControlEnabled(a.download, remoteReady && remoteSelected > 0)

	chmodSelected := 0
	if remoteReady {
		for _, index := range selectedIndices(a.remoteList) {
			if index >= 0 && index < len(a.remoteItems) && !a.remoteItems[index].IsSymlink {
				chmodSelected++
			}
		}
	}
	setControlEnabled(a.remoteChmod, remoteReady && chmodSelected > 0)

	transferState := deriveTransferActionState(a.transferJobs, selectedIndices(a.transferList), a.connected && !a.connectionBusy, a.queuePaused)
	if a.connectionBusy {
		transferState.Pause = false
		transferState.Resume = false
		transferState.Cancel = false
		transferState.Retry = false
	}
	setControlEnabled(a.pauseQueue, transferState.Pause)
	setControlEnabled(a.resumeQueue, transferState.Resume)
	setControlEnabled(a.cancelJob, transferState.Cancel)
	setControlEnabled(a.retryJob, transferState.Retry)
	setControlEnabled(a.clearQueue, transferState.Clear && !a.connectionBusy)

	// The canonical action state still owns enable/disable semantics. Layout is a
	// pure presentation pass applied afterwards so clearer pane proportions never
	// bypass connection, selection or transfer safety checks.
	a.refineWorkspaceLayout()
}
