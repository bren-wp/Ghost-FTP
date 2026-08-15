//go:build windows

package platform

import (
	"os"
	"syscall"
	"unsafe"
)

var moveFileNoReplaceW = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

// RenameNoReplace moves src to dst without ever replacing an existing dst.
// This closes the check-then-rename race around user files on Windows.
func RenameNoReplace(src, dst string) error {
	s, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	d, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	const moveFileWriteThrough = 0x8
	r, _, callErr := moveFileNoReplaceW.Call(
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(d)),
		moveFileWriteThrough,
	)
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return os.ErrExist
	}
	return nil
}
