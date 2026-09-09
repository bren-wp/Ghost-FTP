//go:build linux

package remote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

const maxLinuxTransportSymlinkDepth = 32

var linuxTransportSystemDirs = []string{
	"/usr/bin",
	"/bin",
	"/usr/sbin",
	"/sbin",
	"/usr/local/bin",
	"/usr/local/sbin",
}

// findTrustedTransportExecutable treats PATH only as a discovery hint. Every
// candidate must independently prove that its complete filesystem provenance
// is controlled by root and cannot be replaced by an unprivileged user.
func findTrustedTransportExecutable(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, '\x00') || filepath.Base(name) != name {
		return "", errors.New("invalid Linux transport component name")
	}

	candidates := make([]string, 0, len(linuxTransportSystemDirs)+1)
	if hinted, err := exec.LookPath(name); err == nil {
		candidates = append(candidates, hinted)
	}
	for _, dir := range linuxTransportSystemDirs {
		candidates = append(candidates, filepath.Join(dir, name))
	}

	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if !filepath.IsAbs(candidate) {
			absolute, err := filepath.Abs(candidate)
			if err != nil {
				continue
			}
			candidate = absolute
		}
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		if resolved, ok := trustedLinuxTransportExecutable(candidate); ok {
			return resolved, nil
		}
	}
	return "", errors.New("trusted Linux transport component was not found")
}

func trustedLinuxTransportExecutable(candidate string) (string, bool) {
	return trustedLinuxTransportExecutableDepth(candidate, 0)
}

func trustedLinuxTransportExecutableDepth(candidate string, depth int) (string, bool) {
	if depth > maxLinuxTransportSymlinkDepth || !filepath.IsAbs(candidate) {
		return "", false
	}
	candidate = filepath.Clean(candidate)
	if !trustedLinuxDirectoryChain(filepath.Dir(candidate), depth) {
		return "", false
	}

	info, err := os.Lstat(candidate)
	if err != nil {
		return "", false
	}
	uid, ok := linuxFileUID(info)
	if !ok || uid != 0 {
		return "", false
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(candidate)
		if err != nil {
			return "", false
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(candidate), target)
		}
		resolvedTarget, ok := trustedLinuxTransportExecutableDepth(filepath.Clean(target), depth+1)
		if !ok {
			return "", false
		}
		evaluated, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			return "", false
		}
		evaluated, err = filepath.Abs(evaluated)
		if err != nil || filepath.Clean(evaluated) != resolvedTarget {
			return "", false
		}
		return resolvedTarget, true
	}
	if !trustedLinuxMetadata(uid, info.Mode(), false) {
		return "", false
	}

	// Parent-directory symlinks may cause the canonical path to differ even
	// when the executable itself is not a symlink. Validate the canonical path
	// again before returning it so cmd.Dir also lands in a trusted directory.
	evaluated, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", false
	}
	evaluated, err = filepath.Abs(evaluated)
	if err != nil {
		return "", false
	}
	evaluated = filepath.Clean(evaluated)
	if !trustedLinuxDirectoryChain(filepath.Dir(evaluated), depth) {
		return "", false
	}
	finalInfo, err := os.Stat(evaluated)
	if err != nil {
		return "", false
	}
	finalUID, ok := linuxFileUID(finalInfo)
	if !ok || !trustedLinuxMetadata(finalUID, finalInfo.Mode(), false) {
		return "", false
	}
	return evaluated, true
}

// trustedLinuxDirectoryChain validates every lexical directory component.
// Root-owned symlinks such as /bin -> /usr/bin are allowed only when their
// target chain is itself root-owned and non-writable by group/other users.
func trustedLinuxDirectoryChain(dir string, depth int) bool {
	if depth > maxLinuxTransportSymlinkDepth || !filepath.IsAbs(dir) {
		return false
	}
	dir = filepath.Clean(dir)
	rootInfo, err := os.Lstat(string(os.PathSeparator))
	if err != nil {
		return false
	}
	rootUID, ok := linuxFileUID(rootInfo)
	if !ok || !trustedLinuxMetadata(rootUID, rootInfo.Mode(), true) {
		return false
	}
	if dir == string(os.PathSeparator) {
		return true
	}

	current := string(os.PathSeparator)
	for _, part := range strings.Split(strings.TrimPrefix(dir, string(os.PathSeparator)), string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			return false
		}
		uid, ok := linuxFileUID(info)
		if !ok || uid != 0 {
			return false
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(current)
			if err != nil {
				return false
			}
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(current), target)
			}
			if !trustedLinuxDirectoryChain(filepath.Clean(target), depth+1) {
				return false
			}
			resolvedInfo, err := os.Stat(current)
			if err != nil {
				return false
			}
			resolvedUID, ok := linuxFileUID(resolvedInfo)
			if !ok || !trustedLinuxMetadata(resolvedUID, resolvedInfo.Mode(), true) {
				return false
			}
			continue
		}
		if !trustedLinuxMetadata(uid, info.Mode(), true) {
			return false
		}
	}
	return true
}

func linuxFileUID(info os.FileInfo) (uint32, bool) {
	if info == nil {
		return 0, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return 0, false
	}
	return stat.Uid, true
}

func trustedLinuxMetadata(uid uint32, mode os.FileMode, directory bool) bool {
	if uid != 0 || mode.Perm()&0022 != 0 {
		return false
	}
	if directory {
		return mode.IsDir()
	}
	return mode.IsRegular() && mode.Perm()&0111 != 0
}
