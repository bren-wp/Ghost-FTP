//go:build darwin

package platform

/*
#cgo LDFLAGS: -lproc
#include <libproc.h>
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func LocalAppData() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", errors.New("user home directory is unavailable")
	}
	return filepath.Join(home, "Library", "Application Support"), nil
}

func SystemDirectory() (string, error) {
	return "", errors.New("Windows system directory is unavailable on macOS")
}

// HardenProcessPrivacy applies an owner-only creation mask before the shared
// engine creates any state or credential-bearing temporary files.
func HardenProcessPrivacy() {
	syscall.Umask(0o077)
}

func darwinProcessPath(pid int) string {
	buf := make([]byte, C.PROC_PIDPATHINFO_MAXSIZE)
	if len(buf) == 0 {
		return ""
	}
	n := C.proc_pidpath(C.int(pid), unsafe.Pointer(&buf[0]), C.uint32_t(len(buf)))
	if n <= 0 || int(n) > len(buf) {
		return ""
	}
	return strings.TrimRight(string(buf[:int(n)]), "\x00")
}

func trustedDarwinSystemTool(path string) bool {
	path = filepath.Clean(strings.TrimSpace(path))
	if path != "/usr/bin/ssh" && path != "/usr/bin/sftp" {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o022 != 0 {
		return false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == 0
}

// TrustedAskPassParent permits only Apple's root-owned ssh/sftp executable as
// the immediate parent of the bundled credential helper.
func TrustedAskPassParent() bool {
	return trustedDarwinSystemTool(darwinProcessPath(os.Getppid()))
}

func ChoosePrivateKey() (string, error) {
	return "", errors.New("private-key selection is provided by the macOS AppKit UI")
}

func ChooseDirectory() (string, error) {
	return "", errors.New("directory selection is provided by the macOS AppKit UI")
}

func MessageBox(title, text string, flags uintptr) int {
	_, _ = os.Stderr.WriteString(title + ": " + text + "\n")
	return 0
}

func ConfirmDialog(string, string, string) bool { return false }

func InfoDialog(title, instruction, content string) {
	_, _ = os.Stdout.WriteString(title + ": " + instruction + "\n" + content + "\n")
}

func ErrorDialog(title, instruction, content string) {
	_, _ = os.Stderr.WriteString(title + ": " + instruction + "\n" + content + "\n")
}

func AcquireSingleInstance(string) (func(), bool) { return func() {}, true }
