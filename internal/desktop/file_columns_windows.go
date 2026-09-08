//go:build windows

package desktop

import "unsafe"

func (a *app) fitFileColumnsToWorkspace() {
	if a == nil {
		return
	}
	for _, spec := range []struct {
		list   uintptr
		remote bool
	}{
		{list: a.localList, remote: false},
		{list: a.remoteList, remote: true},
	} {
		if spec.list == 0 {
			continue
		}
		var client rect
		ok, _, _ := getClientRect.Call(spec.list, uintptr(unsafe.Pointer(&client)))
		if ok == 0 {
			continue
		}
		logicalWidth := a.unscale(int(client.Right - client.Left))
		for index, width := range fileColumnWidths(logicalWidth, spec.remote) {
			sendMessageW.Call(spec.list, lvmSetColumnWidth, uintptr(index), uintptr(a.scale(width)))
		}
	}
}
