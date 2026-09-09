//go:build linux

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

func VerifiedRegularFileSHA256(path string) (string, error) {
	before, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("path is not a safe regular file")
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	opened, err := f.Stat()
	if err != nil {
		return "", err
	}
	if !opened.Mode().IsRegular() || !os.SameFile(before, opened) {
		return "", errors.New("file changed during verified open")
	}

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	after, err := f.Stat()
	if err != nil {
		return "", err
	}
	current, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !os.SameFile(opened, after) || !os.SameFile(after, current) || current.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("file changed during verification")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func RemoveVerifiedRegularFileMatchingSHA256(string, string) (bool, error) {
	// This exact-object deletion primitive is intentionally Windows-only. The
	// Linux package lifecycle does not use the Windows legacy Uninstall.exe
	// migration; fail closed rather than emulate it with pathname deletion.
	return false, errors.New("verified legacy Windows file deletion is unavailable on Linux")
}

func ScheduleDeleteOnReboot(string) error { return nil }
