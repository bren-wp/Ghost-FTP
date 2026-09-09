//go:build linux

package platform

import (
	"errors"
	"os"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/linuxtrust"
)

const linuxRunningExecutable = "/proc/self/exe"

// trustedAskPassExecutableIdentity returns a child-executable pathname only
// when that pathname has trusted root-controlled provenance and names the same
// inode as the running Ghost FTP image. /proc/self/exe is used only as an
// identity oracle inside the current process; it is never handed to OpenSSH.
func trustedAskPassExecutableIdentity(exePath, runningPath string) (string, bool) {
	if strings.TrimSpace(exePath) == "" || strings.TrimSpace(runningPath) == "" {
		return "", false
	}
	trustedPath, ok := linuxtrust.TrustedExecutable(exePath)
	if !ok {
		return "", false
	}
	trustedInfo, err := os.Stat(trustedPath)
	if err != nil || !trustedInfo.Mode().IsRegular() {
		return "", false
	}
	runningInfo, err := os.Stat(runningPath)
	if err != nil || !runningInfo.Mode().IsRegular() {
		return "", false
	}
	if !os.SameFile(trustedInfo, runningInfo) {
		return "", false
	}
	return trustedPath, true
}

// StableAskPassExecutable returns an OpenSSH-executable Ghost FTP path only for
// an installed/root-controlled Linux image. A user-writable portable pathname
// cannot be made immutable against a malicious same-UID process, so credential
// AskPass is deliberately unavailable there instead of weakening process-memory
// hardening or passing /proc/<pid>/exe to OpenSSH.
func StableAskPassExecutable(exePath string) (string, error) {
	if trustedPath, ok := trustedAskPassExecutableIdentity(exePath, linuxRunningExecutable); ok {
		return trustedPath, nil
	}
	return "", errors.New("trusted Linux AskPass executable identity is unavailable")
}

// SameExecutableIdentity validates the currently-running helper against the
// same trusted filesystem-provenance boundary used when the main process chose
// SSH_ASKPASS.
func SameExecutableIdentity(_ string, expected string) bool {
	trustedExpected, ok := linuxtrust.TrustedExecutable(strings.TrimSpace(expected))
	if !ok {
		return false
	}
	currentInfo, err := os.Stat(linuxRunningExecutable)
	if err != nil || !currentInfo.Mode().IsRegular() {
		return false
	}
	expectedInfo, err := os.Stat(trustedExpected)
	if err != nil || !expectedInfo.Mode().IsRegular() {
		return false
	}
	return os.SameFile(currentInfo, expectedInfo)
}
