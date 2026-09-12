//go:build windows

package platform

import "unsafe"

var (
	premiumIsDialogMessageW = user32.NewProc("IsDialogMessageW")
	premiumPostQuitMessage  = user32.NewProc("PostQuitMessage")
)

// premiumDialogMessageAvailable preserves the outer application message-loop
// contract when a nested modal loop observes WM_QUIT. GetMessageW returns zero
// for WM_QUIT and -1 for an API error; only the former must be reposted so the
// application's outer loop can consume the original exit code after the modal
// unwinds.
func premiumDialogMessageAvailable(result uintptr, message *promptMsg) bool {
	if int32(result) == -1 {
		return false
	}
	if result == 0 {
		premiumPostQuitMessage.Call(message.WParam)
		return false
	}
	return true
}

// premiumRunDialogLoop is the one nested message-loop contract for all
// application-owned Ghost FTP modal windows. IsDialogMessageW supplies native
// Tab/Shift+Tab traversal and default/cancel button semantics even though these
// surfaces are registered top-level windows rather than DialogBox resources.
func premiumRunDialogLoop(hwnd uintptr, closed func() bool) {
	var message promptMsg
	for !closed() {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if !premiumDialogMessageAvailable(r, &message) {
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