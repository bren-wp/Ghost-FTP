//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

var (
	loadLibraryW   = kernel32.NewProc("LoadLibraryW")
	freeLibrary    = kernel32.NewProc("FreeLibrary")
	getProcAddress = kernel32.NewProc("GetProcAddress")
)

// enableImmersiveDarkMode enables the native dark-menu/control path on modern
// Windows without adding a runtime dependency. The uxtheme ordinals are used
// only on Windows 10 1809+ where Microsoft ships the corresponding functions;
// older systems simply retain their normal system menu rendering. The main
// window invokes this before child controls are created so combo drop-downs,
// native menus and later Site Manager controls inherit the preferred app mode.
func enableImmersiveDarkMode(hwnd uintptr) {
	build := windowsBuildNumber()
	if hwnd == 0 || build < 17763 {
		return
	}
	module, _, _ := loadLibraryW.Call(uintptr(unsafe.Pointer(wstr("uxtheme.dll"))))
	if module == 0 {
		return
	}
	defer freeLibrary.Call(module)

	preferred, _, _ := getProcAddress.Call(module, 135)
	if preferred != 0 {
		// 1809 exposes AllowDarkModeForApp(BOOL) at this ordinal; 1903+
		// exposes SetPreferredAppMode. Both safely accept these small values.
		mode := uintptr(1)
		if build >= 18362 {
			mode = 2 // AllowDark
		}
		syscall.SyscallN(preferred, mode)
	}
	allowWindow, _, _ := getProcAddress.Call(module, 133)
	if allowWindow != 0 {
		syscall.SyscallN(allowWindow, hwnd, 1)
	}
	flushMenus, _, _ := getProcAddress.Call(module, 136)
	if flushMenus != 0 {
		syscall.SyscallN(flushMenus)
	}
}
