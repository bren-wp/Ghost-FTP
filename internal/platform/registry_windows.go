//go:build windows

package platform

import (
	"errors"
	"syscall"
	"unsafe"
)

const (
	hkcu        uintptr = 0x80000001
	keyRead     uintptr = 0x20019
	keyWrite    uintptr = 0x20006
	regSZ       uintptr = 1
	regDWORD    uintptr = 4
	maxRegBytes         = 1 << 20
)

var advapi32 = syscall.NewLazyDLL("advapi32.dll")
var regCreateKeyExW = advapi32.NewProc("RegCreateKeyExW")
var regOpenKeyExW = advapi32.NewProc("RegOpenKeyExW")
var regQueryValueExW = advapi32.NewProc("RegQueryValueExW")
var regSetValueExW = advapi32.NewProc("RegSetValueExW")
var regDeleteValueW = advapi32.NewProc("RegDeleteValueW")
var regDeleteKeyW = advapi32.NewProc("RegDeleteKeyW")
var regCloseKey = advapi32.NewProc("RegCloseKey")

func utf16z(s string) (*uint16, error) { return syscall.UTF16PtrFromString(s) }

func createRegistryKey(subkey string) (uintptr, error) {
	key, err := utf16z(subkey)
	if err != nil {
		return 0, err
	}
	var h uintptr
	var disp uint32
	r, _, _ := regCreateKeyExW.Call(hkcu, uintptr(unsafe.Pointer(key)), 0, 0, 0, keyWrite, 0, uintptr(unsafe.Pointer(&h)), uintptr(unsafe.Pointer(&disp)))
	if r != 0 {
		return 0, syscall.Errno(r)
	}
	return h, nil
}

func openRegistryKey(subkey string, access uintptr) (uintptr, bool, error) {
	key, err := utf16z(subkey)
	if err != nil {
		return 0, false, err
	}
	var h uintptr
	r, _, _ := regOpenKeyExW.Call(hkcu, uintptr(unsafe.Pointer(key)), 0, access, uintptr(unsafe.Pointer(&h)))
	if r == 2 { // ERROR_FILE_NOT_FOUND
		return 0, false, nil
	}
	if r != 0 {
		return 0, false, syscall.Errno(r)
	}
	return h, true, nil
}

func SetRegistryString(subkey, name, value string) error {
	h, err := createRegistryKey(subkey)
	if err != nil {
		return err
	}
	defer regCloseKey.Call(h)
	namePtr, err := utf16z(name)
	if err != nil {
		return err
	}
	v, err := syscall.UTF16FromString(value)
	if err != nil {
		return err
	}
	r, _, _ := regSetValueExW.Call(h, uintptr(unsafe.Pointer(namePtr)), 0, regSZ, uintptr(unsafe.Pointer(&v[0])), uintptr(uint32(len(v)*2)))
	if r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

func SetRegistryDWORD(subkey, name string, value uint32) error {
	h, err := createRegistryKey(subkey)
	if err != nil {
		return err
	}
	defer regCloseKey.Call(h)
	namePtr, err := utf16z(name)
	if err != nil {
		return err
	}
	r, _, _ := regSetValueExW.Call(h, uintptr(unsafe.Pointer(namePtr)), 0, regDWORD, uintptr(unsafe.Pointer(&value)), 4)
	if r != 0 {
		return syscall.Errno(r)
	}
	return nil
}

func GetRegistryString(subkey, name string) (string, bool, error) {
	h, exists, err := openRegistryKey(subkey, keyRead)
	if err != nil || !exists {
		return "", false, err
	}
	defer regCloseKey.Call(h)
	namePtr, err := utf16z(name)
	if err != nil {
		return "", false, err
	}
	var typ uint32
	var size uint32
	r, _, _ := regQueryValueExW.Call(h, uintptr(unsafe.Pointer(namePtr)), 0, uintptr(unsafe.Pointer(&typ)), 0, uintptr(unsafe.Pointer(&size)))
	if r == 2 {
		return "", false, nil
	}
	if r != 0 {
		return "", false, syscall.Errno(r)
	}
	if typ != uint32(regSZ) || size == 0 || size > maxRegBytes {
		return "", false, errors.New("Windows registry vrijednost nije očekivanog tipa")
	}
	buf := make([]uint16, (size+1)/2)
	r, _, _ = regQueryValueExW.Call(h, uintptr(unsafe.Pointer(namePtr)), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r != 0 {
		return "", false, syscall.Errno(r)
	}
	return syscall.UTF16ToString(buf), true, nil
}

func GetRegistryDWORD(subkey, name string) (uint32, bool, error) {
	h, exists, err := openRegistryKey(subkey, keyRead)
	if err != nil || !exists {
		return 0, false, err
	}
	defer regCloseKey.Call(h)
	namePtr, err := utf16z(name)
	if err != nil {
		return 0, false, err
	}
	var typ uint32
	var size uint32 = 4
	var value uint32
	r, _, _ := regQueryValueExW.Call(h, uintptr(unsafe.Pointer(namePtr)), 0, uintptr(unsafe.Pointer(&typ)), uintptr(unsafe.Pointer(&value)), uintptr(unsafe.Pointer(&size)))
	if r == 2 {
		return 0, false, nil
	}
	if r != 0 {
		return 0, false, syscall.Errno(r)
	}
	if typ != uint32(regDWORD) || size != 4 {
		return 0, false, errors.New("Windows registry vrijednost nije očekivanog tipa")
	}
	return value, true, nil
}

func DeleteRegistryValue(subkey, name string) error {
	h, exists, err := openRegistryKey(subkey, keyWrite)
	if err != nil || !exists {
		return err
	}
	defer regCloseKey.Call(h)
	namePtr, err := utf16z(name)
	if err != nil {
		return err
	}
	r, _, _ := regDeleteValueW.Call(h, uintptr(unsafe.Pointer(namePtr)))
	if r != 0 && r != 2 {
		return syscall.Errno(r)
	}
	return nil
}

func DeleteRegistryKey(subkey string) error {
	key, err := utf16z(subkey)
	if err != nil {
		return err
	}
	r, _, _ := regDeleteKeyW.Call(hkcu, uintptr(unsafe.Pointer(key)))
	if r != 0 && r != 2 {
		return syscall.Errno(r)
	}
	return nil
}

var kernel32Files = syscall.NewLazyDLL("kernel32.dll")
var moveFileExW = kernel32Files.NewProc("MoveFileExW")

func ReplaceFile(src, dst string) error {
	s, err := syscall.UTF16PtrFromString(src)
	if err != nil {
		return err
	}
	d, err := syscall.UTF16PtrFromString(dst)
	if err != nil {
		return err
	}
	const moveFileReplaceExisting = 0x1
	const moveFileWriteThrough = 0x8
	r, _, callErr := moveFileExW.Call(uintptr(unsafe.Pointer(s)), uintptr(unsafe.Pointer(d)), moveFileReplaceExisting|moveFileWriteThrough)
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}
	return nil
}
