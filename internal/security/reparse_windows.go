//go:build windows

package security

import (
	"syscall"
	"unsafe"
)

var (
	kernel32Reparse    = syscall.NewLazyDLL("kernel32.dll")
	getFileAttributesW = kernel32Reparse.NewProc("GetFileAttributesW")
)

func isReparsePoint(path string) bool {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return true
	}
	attrs, _, _ := getFileAttributesW.Call(uintptr(unsafe.Pointer(p)))
	const (
		invalidFileAttributes = 0xffffffff
		fileAttributeReparse  = 0x00000400
	)
	if uint32(attrs) == invalidFileAttributes {
		// On an unexpected attribute failure, fail closed and treat the entry as a
		// non-traversable reparse-like object. os.Remove will return the real error.
		return true
	}
	return uint32(attrs)&fileAttributeReparse != 0
}
