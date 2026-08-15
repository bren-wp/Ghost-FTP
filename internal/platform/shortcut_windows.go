//go:build windows

package platform

import (
	"brendigo.com/byftp/internal/brand"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

var (
	ole32Shortcut    = syscall.NewLazyDLL("ole32.dll")
	coInitializeEx   = ole32Shortcut.NewProc("CoInitializeEx")
	coUninitialize   = ole32Shortcut.NewProc("CoUninitialize")
	coCreateInstance = ole32Shortcut.NewProc("CoCreateInstance")
	shell32Shortcut  = syscall.NewLazyDLL("shell32.dll")
	shGetFolderPathW = shell32Shortcut.NewProc("SHGetFolderPathW")
	clsidShellLink   = guid{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIShellLinkW   = guid{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
	iidIPersistFile  = guid{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
)

func hresultFailed(v uintptr) bool { return int32(v) < 0 }

func vtableMethod(obj unsafe.Pointer, index uintptr) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	method := unsafe.Add(vtbl, index*unsafe.Sizeof(uintptr(0)))
	return *(*uintptr)(method)
}

func releaseCOM(obj unsafe.Pointer) {
	if obj != nil {
		syscall.SyscallN(vtableMethod(obj, 2), uintptr(obj))
	}
}

func knownFolder(csidl int32) (string, error) {
	buf := make([]uint16, 32768)
	hr, _, _ := shGetFolderPathW.Call(0, uintptr(uint32(csidl)), 0, 0, uintptr(unsafe.Pointer(&buf[0])))
	if hresultFailed(hr) {
		return "", syscall.Errno(uint32(hr))
	}
	return syscall.UTF16ToString(buf), nil
}

func createShellLink(linkPath, target, workingDir, description string) error {
	const (
		coinitApartmentThreaded = 0x2
		clsctxInprocServer      = 0x1
	)
	hr, _, _ := coInitializeEx.Call(0, coinitApartmentThreaded)
	// S_FALSE (1) is success. RPC_E_CHANGED_MODE is intentionally treated as an error.
	if hresultFailed(hr) {
		return syscall.Errno(uint32(hr))
	}
	defer coUninitialize.Call()

	var link unsafe.Pointer
	hr, _, _ = coCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidShellLink)),
		0,
		clsctxInprocServer,
		uintptr(unsafe.Pointer(&iidIShellLinkW)),
		uintptr(unsafe.Pointer(&link)),
	)
	if hresultFailed(hr) || link == nil {
		return errors.New("Windows Shell Link nije dostupan")
	}
	defer releaseCOM(link)

	pathPtr, _ := syscall.UTF16PtrFromString(target)
	if hr, _, _ = syscall.SyscallN(vtableMethod(link, 20), uintptr(link), uintptr(unsafe.Pointer(pathPtr))); hresultFailed(hr) {
		return syscall.Errno(uint32(hr))
	}
	workPtr, _ := syscall.UTF16PtrFromString(workingDir)
	if hr, _, _ = syscall.SyscallN(vtableMethod(link, 9), uintptr(link), uintptr(unsafe.Pointer(workPtr))); hresultFailed(hr) {
		return syscall.Errno(uint32(hr))
	}
	descPtr, _ := syscall.UTF16PtrFromString(description)
	if hr, _, _ = syscall.SyscallN(vtableMethod(link, 7), uintptr(link), uintptr(unsafe.Pointer(descPtr))); hresultFailed(hr) {
		return syscall.Errno(uint32(hr))
	}
	iconPtr, _ := syscall.UTF16PtrFromString(target)
	if hr, _, _ = syscall.SyscallN(vtableMethod(link, 17), uintptr(link), uintptr(unsafe.Pointer(iconPtr)), 0); hresultFailed(hr) {
		return syscall.Errno(uint32(hr))
	}

	var persist unsafe.Pointer
	if hr, _, _ = syscall.SyscallN(vtableMethod(link, 0), uintptr(link), uintptr(unsafe.Pointer(&iidIPersistFile)), uintptr(unsafe.Pointer(&persist))); hresultFailed(hr) || persist == nil {
		return errors.New("IPersistFile nije dostupan")
	}
	defer releaseCOM(persist)

	if err := os.MkdirAll(filepath.Dir(linkPath), 0755); err != nil {
		return err
	}
	linkPtr, _ := syscall.UTF16PtrFromString(linkPath)
	if hr, _, _ = syscall.SyscallN(vtableMethod(persist, 6), uintptr(persist), uintptr(unsafe.Pointer(linkPtr)), 1); hresultFailed(hr) {
		return syscall.Errno(uint32(hr))
	}
	return nil
}

func ShortcutPaths() (desktop, startMenu string, err error) {
	const (
		csidlPrograms         = 0x0002
		csidlDesktopDirectory = 0x0010
	)
	desktopDir, err := knownFolder(csidlDesktopDirectory)
	if err != nil {
		return "", "", err
	}
	programsDir, err := knownFolder(csidlPrograms)
	if err != nil {
		return "", "", err
	}
	return filepath.Join(desktopDir, brand.ProductFull+".lnk"), filepath.Join(programsDir, brand.Company, brand.ProductFull+".lnk"), nil
}

func CreateShortcuts(appPath string) error {
	desktop, start, err := ShortcutPaths()
	if err != nil {
		return err
	}
	work := filepath.Dir(appPath)
	if err = createShellLink(desktop, appPath, work, brand.ProductFull+" — "+brand.Company); err != nil {
		return err
	}
	if err = createShellLink(start, appPath, work, brand.ProductFull+" — "+brand.Company); err != nil {
		_ = os.Remove(desktop)
		return err
	}
	return nil
}

func RemoveShortcuts() error {
	desktop, start, err := ShortcutPaths()
	if err != nil {
		return err
	}
	var errs []error
	for _, path := range []string{desktop, start} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	startDir := filepath.Dir(start)
	if err := os.Remove(startDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		// A non-empty company Start Menu folder is normal; do not remove
		// unrelated shortcuts that may belong to other Brendigo products.
		if entries, readErr := os.ReadDir(startDir); readErr != nil || len(entries) == 0 {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
