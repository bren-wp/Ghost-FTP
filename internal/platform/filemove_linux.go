//go:build linux

package platform

import (
	"syscall"
	"unsafe"
)

// RenameNoReplace uses renameat2(RENAME_NOREPLACE), which asks the kernel to
// reject an existing destination atomically instead of checking the path first.
func RenameNoReplace(src, dst string) error {
	s, err := syscall.BytePtrFromString(src)
	if err != nil {
		return err
	}
	d, err := syscall.BytePtrFromString(dst)
	if err != nil {
		return err
	}
	const renameNoReplace = 1
	dirFD := -100 // AT_FDCWD
	_, _, errno := syscall.Syscall6(
		syscall.SYS_RENAMEAT2,
		uintptr(dirFD), uintptr(unsafe.Pointer(s)),
		uintptr(dirFD), uintptr(unsafe.Pointer(d)),
		uintptr(renameNoReplace), 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
