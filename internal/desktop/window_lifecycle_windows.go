//go:build windows

package desktop

import "syscall"

const (
	wmSysCommand      = 0x0112
	systemCommandMask = 0xFFF0
	scMinimize        = 0xF020
	scClose           = 0xF060
)

// Windows places invocation metadata in the low four bits of WM_SYSCOMMAND.
// Masking those bits keeps mouse, keyboard and accessibility activation of the
// titlebar controls on the same lifecycle path.
func normalizedSystemCommand(wParam uintptr) uintptr {
	return wParam & systemCommandMask
}

func isCloseSystemCommand(wParam uintptr) bool {
	return normalizedSystemCommand(wParam) == scClose
}

// The native main window already has a fail-safe WM_CLOSE path that confirms
// active connections/transfers and then destroys the window. Intercept only
// SC_CLOSE here so the titlebar X is guaranteed to enter that path directly.
// Other system commands, especially SC_MINIMIZE, continue through the normal
// Win32 default processing and therefore cannot be mistaken for close.
func systemCommandAwareWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if message == wmSysCommand && isCloseSystemCommand(wParam) {
		return wndProc(hwnd, wmClose, 0, 0)
	}
	return wndProc(hwnd, message, wParam, lParam)
}

func init() {
	wndProcPtr = syscall.NewCallback(systemCommandAwareWndProc)
}
