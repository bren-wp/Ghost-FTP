//go:build windows

package desktop

import "unsafe"

const (
	workspaceLVMGetHeader           = 0x101F
	workspaceLVMSetColumnOrderArray = lvmFirst + 58
	workspaceSWHide                 = 0
)

// styleWorkspaceList keeps the native list controls visually consistent with
// the Ghost FTP dark desktop surface without introducing a second UI toolkit.
func styleWorkspaceList(list uintptr) {
	if list == 0 {
		return
	}
	setWindowTheme.Call(list, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
	sendMessageW.Call(list, lvmSetBkColor, 0, listColor())
	sendMessageW.Call(list, lvmSetTextBkColor, 0, listColor())
	sendMessageW.Call(list, lvmSetTextColor, 0, textColor())
	header, _, _ := sendMessageW.Call(list, workspaceLVMGetHeader, 0, 0)
	if header != 0 {
		setWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_ItemsView"))), 0)
	}
}

func applyReferenceFileColumnOrder(list uintptr, remote bool) {
	if list == 0 {
		return
	}
	if remote {
		order := [5]int32{0, 2, 1, 3, 4}
		sendMessageW.Call(list, workspaceLVMSetColumnOrderArray, uintptr(len(order)), uintptr(unsafe.Pointer(&order[0])))
		return
	}
	order := [4]int32{0, 2, 1, 3}
	sendMessageW.Call(list, workspaceLVMSetColumnOrderArray, uintptr(len(order)), uintptr(unsafe.Pointer(&order[0])))
}

// resizeReferenceWorkspaceColumns is retained as the compatibility entry point
// used by the native shell. One canonical column-sizing implementation now
// owns the geometry, avoiding the previous double-layout drift.
func (a *app) resizeReferenceWorkspaceColumns(_, _ int) {
	if a == nil {
		return
	}
	applyReferenceFileColumnOrder(a.localList, false)
	applyReferenceFileColumnOrder(a.remoteList, true)
	a.resizeListColumns()
}

func showControls(show bool, controls ...uintptr) {
	state := workspaceSWHide
	if show {
		state = swShow
	}
	for _, control := range controls {
		if control != 0 {
			showWindow.Call(control, uintptr(state))
		}
	}
}

// refineWorkspaceLayout deliberately does not build a second presentation
// layer. Earlier Beta builds first laid out the real controls and then overlaid
// a separate reference shell with duplicate toolbar aliases. That increased
// visual noise and created resize/language regressions. The 0.2 line keeps one
// canonical native workspace: profiles + Quick Connect, two file panes and one
// transfer queue. Every visible action maps directly to the Engine command that
// owns its validation and security checks.
func (a *app) refineWorkspaceLayout() {
	if a == nil || a.hwnd == 0 {
		return
	}

	// The legacy reference-shell controls can still be created by old action
	// state paths for binary/source compatibility, but they are never presented.
	showControls(false,
		a.shellSidebar, a.shellToolbar, a.shellLogCard, a.shellQuickCard,
		a.shellLocalCard, a.shellRemoteCard, a.shellQueueCard,
		a.sidebarServersLabel, a.sidebarPrivacyTitle, a.sidebarPrivacyBody,
		a.logTitle, a.quickTitle, a.localDeviceLabel, a.remoteStateLabel,
		a.remoteSearch,
		a.toolbarConnect, a.toolbarDisconnect, a.toolbarUpload, a.toolbarDownload,
		a.toolbarRefresh, a.toolbarNewFolder, a.toolbarRename, a.toolbarDelete,
		a.toolbarSites, a.toolbarSettings, a.toolbarDiagnostics,
	)

	var client rect
	if result, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); result != 0 {
		a.layout(int(client.Right-client.Left), int(client.Bottom-client.Top))
	}

	// Keep the real native controls visible. Protocol-specific secrets and the
	// connect/disconnect pair are context-sensitive to reduce clutter.
	showControls(true,
		a.brandTitle, a.brandSubtitle, a.connectionBadge, a.languageCombo,
		a.profilesCombo, a.saveProfile, a.removeProfile, a.settingsBtn, a.aboutBtn,
		a.protocol, a.host, a.port, a.user, a.pass,
		a.sectionLocal, a.sectionRemote, a.sectionTransfers,
		a.localPath, a.localUp, a.localRefresh, a.localChoose, a.localList,
		a.localMkdir, a.localRename, a.localDelete,
		a.remotePath, a.remoteUp, a.remoteRefresh, a.remoteList,
		a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod,
		a.upload, a.download,
		a.transferList, a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue,
		a.status, a.statusVersion, a.transferSummary,
	)

	sftp := a.protocolValue() == "sftp" && !a.connected
	showControls(sftp, a.keyPath, a.chooseKey, a.passphrase)

	if a.connected || a.connectionBusy {
		showControls(false, a.connect)
		showControls(true, a.disconnect)
	} else {
		showControls(true, a.connect)
		showControls(false, a.disconnect)
	}

	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		styleWorkspaceList(list)
	}
	applyReferenceFileColumnOrder(a.localList, false)
	applyReferenceFileColumnOrder(a.remoteList, true)
	a.resizeListColumns()
	invalidateRect.Call(a.hwnd, 0, 1)
}
