//go:build windows

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	genericReadAccess         = 0x80000000
	deleteFileAccess          = 0x00010000
	fileShareRead             = 0x00000001
	openExisting              = 3
	fileFlagOpenReparsePoint  = 0x00200000
	fileAttributeDirectory    = 0x00000010
	fileAttributeReparsePoint = 0x00000400
	fileDispositionInfo       = 4
)

var errVerifiedOwnershipDigestMismatch = errors.New("verified file content does not match its ownership digest")

var kernel32Delete = syscall.NewLazyDLL("kernel32.dll")
var moveFileExDelete = kernel32Delete.NewProc("MoveFileExW")
var setFileInformationByHandleDelete = kernel32Delete.NewProc("SetFileInformationByHandle")

func openVerifiedRegularFile(path string, allowDelete bool) (*os.File, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	access := uint32(genericReadAccess)
	if allowDelete {
		access |= deleteFileAccess
	}
	h, err := syscall.CreateFile(
		p,
		access,
		fileShareRead,
		nil,
		openExisting,
		fileFlagOpenReparsePoint,
		0,
	)
	if err != nil {
		return nil, err
	}

	var info syscall.ByHandleFileInformation
	if err := syscall.GetFileInformationByHandle(h, &info); err != nil {
		_ = syscall.CloseHandle(h)
		return nil, err
	}
	if info.FileAttributes&(fileAttributeDirectory|fileAttributeReparsePoint) != 0 {
		_ = syscall.CloseHandle(h)
		return nil, errors.New("path is not a safe regular file")
	}

	f := os.NewFile(uintptr(h), path)
	if f == nil {
		_ = syscall.CloseHandle(h)
		return nil, errors.New("Windows file handle could not be attached")
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if !stat.Mode().IsRegular() {
		_ = f.Close()
		return nil, errors.New("path is not a safe regular file")
	}
	return f, nil
}

func verifiedOpenFileSHA256(f *os.File) (string, error) {
	if f == nil {
		return "", errors.New("verified file handle is unavailable")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func verifiedRegularFileSHA256(path string) (string, error) {
	f, err := openVerifiedRegularFile(path, false)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return verifiedOpenFileSHA256(f)
}

func deleteVerifiedOpenFile(f *os.File) error {
	if f == nil {
		return errors.New("verified file handle is unavailable")
	}
	deleteFile := uint32(1)
	r, _, callErr := setFileInformationByHandleDelete.Call(
		f.Fd(),
		fileDispositionInfo,
		uintptr(unsafe.Pointer(&deleteFile)),
		unsafe.Sizeof(deleteFile),
	)
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("Windows could not delete the verified file object")
	}
	return nil
}

func removeVerifiedRegularFileMatchingSHA256(path, expectedDigest string) (bool, error) {
	if strings.TrimSpace(expectedDigest) == "" {
		return false, nil
	}

	f, err := openVerifiedRegularFile(path, true)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer f.Close()

	actualDigest, err := verifiedOpenFileSHA256(f)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(actualDigest, expectedDigest) {
		return false, errVerifiedOwnershipDigestMismatch
	}
	if err := deleteVerifiedOpenFile(f); err != nil {
		return false, err
	}
	return true, nil
}

// ScheduleDeleteOnReboot asks Windows to remove a path at the next reboot.
// It is used only as a fallback when an executable is still locked.
func ScheduleDeleteOnReboot(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, callErr := moveFileExDelete.Call(uintptr(unsafe.Pointer(p)), 0, 0x4) // MOVEFILE_DELAY_UNTIL_REBOOT
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return os.ErrInvalid
	}
	return nil
}
