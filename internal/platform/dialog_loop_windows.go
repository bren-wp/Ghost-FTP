//go:build windows

package platform

import "unsafe"

var premiumIsDialogMessageW = user32.NewProc("IsDialogMessageW")

// premiumRunDialogLoop is the one nested message-loop contract for all
// application-owned Ghost FTP modal windows. IsDialogMessageW supplies native
// Tab/Shift+Tab traversal and default/cancel button semantics even though these
// surfaces are registered top-level windows rather than DialogBox resources.
func premiumRunDialogLoop(hwnd uintptr, closed func() bool) {
	var message promptMsg
	for !closed() {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		handled, _, _ := premiumIsDialogMessageW.Call(hwnd, uintptr(unsafe.Pointer(&message)))
		if handled != 0 {
			continue
		}
		promptTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		promptDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}
