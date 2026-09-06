//go:build windows

package desktop

import "unsafe"

const (
	workspaceLVMGetHeader           = 0x101F
	workspaceLVMSetColumnOrderArray = lvmFirst + 58
	workspaceSWHide                 = 0
)

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
		// Explorer's dark header theme keeps the column text readable on Windows
		// versions where DarkMode_ItemsView falls back to a dark face with dark
		// text. The screenshot pipeline guards this native-control regression.
		setWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
	}
}

func applyFileColumnOrder(list uintptr, remote bool) {
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

// refineWorkspaceLayout applies visibility and native-theme rules to the
// canonical workspace. Geometry remains owned by app.layout. Importantly this
// function does not invalidate the entire parent window: MoveWindow and the
// individual owner-drawn controls already repaint their own changed bounds.
// Avoiding a full background erase removes the startup/resize flash that was
// visible on real Windows systems.
func (a *app) refineWorkspaceLayout() {
	if a == nil || a.hwnd == 0 {
		return
	}

	sftp := a.protocolValue() == "sftp" && !a.connected && !a.connectionBusy
	showControls(sftp, a.keyPath, a.chooseKey, a.passphrase)
	if a.connected || a.connectionBusy {
		showControls(false, a.connect)
		showControls(true, a.disconnect)
	} else {
		showControls(true, a.connect)
		showControls(false, a.disconnect)
	}

	a.stabilizeWorkspaceChrome()
	applyFileColumnOrder(a.localList, false)
	applyFileColumnOrder(a.remoteList, true)
	a.resizeListColumns()
}
