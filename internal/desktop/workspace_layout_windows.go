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
		setWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_ItemsView"))), 0)
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

// refineWorkspaceLayout applies visibility/style rules to the single canonical
// native workspace. Geometry itself is owned by app.layout so resize/DPI state
// has exactly one source of truth.
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

	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		styleWorkspaceList(list)
	}
	applyFileColumnOrder(a.localList, false)
	applyFileColumnOrder(a.remoteList, true)
	a.resizeListColumns()
	invalidateRect.Call(a.hwnd, 0, 1)
}
