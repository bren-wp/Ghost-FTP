//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

type guid struct {
	Data1 uint32
	Data2 uint16
	Data3 uint16
	Data4 [8]byte
}

const (
	desktopShortcutDigestValue   = "DesktopShortcutSHA256"
	startMenuShortcutDigestValue = "StartMenuShortcutSHA256"
)

var ole32Shortcut = syscall.NewLazyDLL("ole32.dll")
var coInitializeEx = ole32Shortcut.NewProc("CoInitializeEx")
var coUninitialize = ole32Shortcut.NewProc("CoUninitialize")
var coCreateInstance = ole32Shortcut.NewProc("CoCreateInstance")
var shell32Shortcut = syscall.NewLazyDLL("shell32.dll")
var shGetFolderPathW = shell32Shortcut.NewProc("SHGetFolderPathW")
var clsidShellLink = guid{0x00021401, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
var iidIShellLinkW = guid{0x000214F9, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}
var iidIPersistFile = guid{0x0000010B, 0, 0, [8]byte{0xC0, 0, 0, 0, 0, 0, 0, 0x46}}

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

func stableShortcutDigest(path string) (string, error) {
	return verifiedRegularFileSHA256(path)
}

func readShellLinkTarget(linkPath string) (string, error) {
	const (
		coinitApartmentThreaded = 0x2
		clsctxInprocServer      = 0x1
		slgpRawPath             = 0x4
	)
	hr, _, _ := coInitializeEx.Call(0, coinitApartmentThreaded)
	if hresultFailed(hr) {
		return "", syscall.Errno(uint32(hr))
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
		return "", errors.New("Windows Shell Link nije dostupan")
	}
	defer releaseCOM(link)

	var persist unsafe.Pointer
	if hr, _, _ = syscall.SyscallN(vtableMethod(link, 0), uintptr(link), uintptr(unsafe.Pointer(&iidIPersistFile)), uintptr(unsafe.Pointer(&persist))); hresultFailed(hr) || persist == nil {
		return "", errors.New("IPersistFile nije dostupan")
	}
	defer releaseCOM(persist)

	pathPtr, err := syscall.UTF16PtrFromString(linkPath)
	if err != nil {
		return "", err
	}
	if hr, _, _ = syscall.SyscallN(vtableMethod(persist, 5), uintptr(persist), uintptr(unsafe.Pointer(pathPtr)), 0); hresultFailed(hr) {
		return "", syscall.Errno(uint32(hr))
	}

	buf := make([]uint16, 32768)
	if hr, _, _ = syscall.SyscallN(
		vtableMethod(link, 3),
		uintptr(link),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)),
		0,
		slgpRawPath,
	); hresultFailed(hr) {
		return "", syscall.Errno(uint32(hr))
	}
	target := syscall.UTF16ToString(buf)
	if strings.TrimSpace(target) == "" {
		return "", errors.New("shortcut nema valjanu ciljnu putanju")
	}
	return target, nil
}

func sameShortcutTarget(actual, expected string) bool {
	actualAbs, err := filepath.Abs(filepath.Clean(actual))
	if err != nil {
		return false
	}
	expectedAbs, err := filepath.Abs(filepath.Clean(expected))
	if err != nil {
		return false
	}
	return strings.EqualFold(actualAbs, expectedAbs)
}

func removeShortcutMatchingDigest(path, expectedDigest string) (bool, error) {
	removed, err := removeVerifiedRegularFileMatchingSHA256(path, expectedDigest)
	if err != nil {
		if strings.Contains(err.Error(), "ownership digest") {
			return false, errors.New("shortcut je promijenjen nakon instalacije i neće biti obrisan")
		}
		return false, err
	}
	return removed, nil
}

func createOwnedShortcut(linkPath, target, workingDir, description, digestValue string) error {
	if _, err := os.Lstat(linkPath); err == nil {
		digest, digestErr := stableShortcutDigest(linkPath)
		if digestErr != nil {
			return digestErr
		}
		recorded, recordedOK, recordedErr := GetRegistryString(ghostFTPUninstallKey, digestValue)
		if recordedErr != nil {
			return recordedErr
		}
		if recordedOK && strings.EqualFold(recorded, digest) {
			return nil
		}

		// Migrate a legacy Ghost FTP shortcut only after independently proving
		// that its stored target is the canonical application path. A foreign
		// same-name shortcut is preserved and never adopted.
		existingTarget, targetErr := readShellLinkTarget(linkPath)
		if targetErr != nil || !sameShortcutTarget(existingTarget, target) {
			_ = DeleteRegistryValue(ghostFTPUninstallKey, digestValue)
			return errors.New("postojeći shortcut istog imena nije Ghost FTP shortcut i nije prepisan")
		}
		return SetRegistryString(ghostFTPUninstallKey, digestValue, digest)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	if err := createShellLink(linkPath, target, workingDir, description); err != nil {
		return err
	}
	digest, err := stableShortcutDigest(linkPath)
	if err != nil {
		return err
	}
	if err := SetRegistryString(ghostFTPUninstallKey, digestValue, digest); err != nil {
		_, cleanupErr := removeShortcutMatchingDigest(linkPath, digest)
		return errors.Join(err, cleanupErr)
	}
	return nil
}

func removeOwnedShortcut(path, digestValue string) error {
	expected, ok, err := GetRegistryString(ghostFTPUninstallKey, digestValue)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	_, err = removeShortcutMatchingDigest(path, expected)
	if err != nil {
		return err
	}
	return DeleteRegistryValue(ghostFTPUninstallKey, digestValue)
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
	description := brand.ProductFull + " — " + brand.Company

	var errs []error
	if err := createOwnedShortcut(desktop, appPath, work, description, desktopShortcutDigestValue); err != nil {
		errs = append(errs, err)
	}
	if err := createOwnedShortcut(start, appPath, work, description, startMenuShortcutDigestValue); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func RemoveShortcuts() error {
	desktop, start, err := ShortcutPaths()
	if err != nil {
		return err
	}
	var errs []error
	if err := removeOwnedShortcut(desktop, desktopShortcutDigestValue); err != nil {
		errs = append(errs, err)
	}
	if err := removeOwnedShortcut(start, startMenuShortcutDigestValue); err != nil {
		errs = append(errs, err)
	}

	startDir := filepath.Dir(start)
	if err := os.Remove(startDir); err != nil && !errors.Is(err, os.ErrNotExist) {
		// A non-empty company Start Menu folder is normal; do not remove
		// unrelated shortcuts that may belong to other GhostFTP products.
		if entries, readErr := os.ReadDir(startDir); readErr != nil || len(entries) == 0 {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
