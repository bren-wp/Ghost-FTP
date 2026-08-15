//go:build windows

package platform

import (
	"os"
	"syscall"
	"unsafe"
)

// ScheduleDeleteOnReboot asks Windows to remove a path at the next reboot.
// It is used only as a fallback when an executable is still locked.
func ScheduleDeleteOnReboot(path string) error {
	moveFile := syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, callErr := moveFile.Call(uintptr(unsafe.Pointer(p)), 0, 0x4) // MOVEFILE_DELAY_UNTIL_REBOOT
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return os.ErrInvalid
	}
	return nil
}
