//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

func StableAskPassExecutable(exePath string) (string, error) {
	if strings.TrimSpace(exePath) == "" {
		return "", errors.New("application executable path is unavailable")
	}
	abs, err := filepath.Abs(exePath)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	info, err := os.Stat(abs)
	if err != nil || !info.Mode().IsRegular() {
		return "", errors.New("application executable identity is unavailable")
	}
	return abs, nil
}

func SameExecutableIdentity(actual, expected string) bool {
	if strings.TrimSpace(actual) == "" || strings.TrimSpace(expected) == "" {
		return false
	}
	actualAbs, err := filepath.Abs(actual)
	if err != nil {
		return false
	}
	expectedAbs, err := filepath.Abs(expected)
	if err != nil {
		return false
	}
	if !strings.EqualFold(filepath.Clean(actualAbs), filepath.Clean(expectedAbs)) {
		return false
	}
	actualInfo, err := os.Stat(actualAbs)
	if err != nil || !actualInfo.Mode().IsRegular() {
		return false
	}
	expectedInfo, err := os.Stat(expectedAbs)
	if err != nil || !expectedInfo.Mode().IsRegular() {
		return false
	}
	return os.SameFile(actualInfo, expectedInfo)
}
