//go:build linux

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// StableAskPassExecutable returns an immutable reference to the executable
// image of the running Ghost FTP process. A pathname of a portable/per-user
// binary can be unlinked and replaced while the process is running; /proc PID
// exe remains bound to the already-running inode and therefore cannot be
// redirected to a replacement helper before OpenSSH invokes SSH_ASKPASS.
func StableAskPassExecutable(exePath string) (string, error) {
	if strings.TrimSpace(exePath) == "" {
		return "", errors.New("application executable path is unavailable")
	}
	startedInfo, err := os.Stat(exePath)
	if err != nil || !startedInfo.Mode().IsRegular() {
		return "", errors.New("application executable identity is unavailable")
	}

	procPath := filepath.Join("/proc", strconv.Itoa(os.Getpid()), "exe")
	procInfo, err := os.Stat(procPath)
	if err != nil || !procInfo.Mode().IsRegular() {
		return "", errors.New("stable application executable identity is unavailable")
	}
	if !os.SameFile(startedInfo, procInfo) {
		return "", errors.New("application executable identity changed during startup")
	}
	return procPath, nil
}

// SameExecutableIdentity validates the actual currently-running helper image,
// not the mutable pathname returned by os.Executable. This remains valid even
// if the original portable executable pathname is replaced after startup.
func SameExecutableIdentity(_ string, expected string) bool {
	if strings.TrimSpace(expected) == "" {
		return false
	}
	currentPath := filepath.Join("/proc", strconv.Itoa(os.Getpid()), "exe")
	currentInfo, err := os.Stat(currentPath)
	if err != nil || !currentInfo.Mode().IsRegular() {
		return false
	}
	expectedInfo, err := os.Stat(expected)
	if err != nil || !expectedInfo.Mode().IsRegular() {
		return false
	}
	return os.SameFile(currentInfo, expectedInfo)
}
