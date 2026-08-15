//go:build windows

package config

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32Replace = syscall.NewLazyDLL("kernel32.dll")
	moveFileExW     = kernel32Replace.NewProc("MoveFileExW")
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

func replaceFile(src, dst string) error {
	s, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	d, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	r, _, callErr := moveFileExW.Call(
		uintptr(unsafe.Pointer(s)),
		uintptr(unsafe.Pointer(d)),
		moveFileReplaceExisting|moveFileWriteThrough,
	)
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return os.ErrInvalid
	}
	return nil
}
