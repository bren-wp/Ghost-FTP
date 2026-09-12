//go:build darwin

package remote

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func findTrustedTransportExecutable(name string) (string, error) {
	base := strings.ToLower(strings.TrimSpace(filepath.Base(name)))
	base = strings.TrimSuffix(base, ".exe")
	if base != "ssh" && base != "sftp" && base != "ssh-keyscan" {
		return "", errors.New("unsupported macOS transport executable")
	}
	path := filepath.Join("/usr/bin", base)
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("trusted macOS transport executable is unavailable")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return "", errors.New("macOS transport executable permissions are unsafe")
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != 0 {
		return "", errors.New("macOS transport executable owner is unsafe")
	}
	return path, nil
}
