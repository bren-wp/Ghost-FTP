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

// enableImmersiveDarkMode keeps the legacy call-site name for compatibility,
// but now applies the active Ghost FTP appearance. On light appearance it
// explicitly returns the process/window to the normal Windows app mode instead
// of leaving DarkMode_* state behind on menus and native controls.
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

	dark := activeThemeIsDark()
	preferred, _, _ := getProcAddress.Call(module, 135)
	if preferred != 0 {
		if build >= 18362 {
			mode := uintptr(0) // Default
			if dark {
				mode = 2 // AllowDark
			}
			syscall.SyscallN(preferred, mode)
		} else {
			value := uintptr(0)
			if dark {
				value = 1
			}
			syscall.SyscallN(preferred, value)
		}
	}
	allowWindow, _, _ := getProcAddress.Call(module, 133)
	if allowWindow != 0 {
		value := uintptr(0)
		if dark {
			value = 1
		}
		syscall.SyscallN(allowWindow, hwnd, value)
	}
	flushMenus, _, _ := getProcAddress.Call(module, 136)
	if flushMenus != 0 {
		syscall.SyscallN(flushMenus)
	}
}
