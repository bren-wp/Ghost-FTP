//go:build linux

package linuxtrust

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const maxSymlinkDepth = 32

// TrustedExecutable validates that an executable and its complete filesystem
// provenance are controlled by root and cannot be replaced by an unprivileged
// user. Root-owned symlink chains (for example /bin -> /usr/bin) are accepted
// only when every lexical and resolved directory component is itself trusted.
func TrustedExecutable(candidate string) (string, bool) {
	return trustedExecutableDepth(candidate, 0)
}

func trustedExecutableDepth(candidate string, depth int) (string, bool) {
	if depth > maxSymlinkDepth || !filepath.IsAbs(candidate) {
		return "", false
	}
	candidate = filepath.Clean(candidate)
	if !TrustedDirectoryChain(filepath.Dir(candidate), depth) {
		return "", false
	}

	info, err := os.Lstat(candidate)
	if err != nil {
		return "", false
	}
	uid, ok := fileUID(info)
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
		resolvedTarget, ok := trustedExecutableDepth(filepath.Clean(target), depth+1)
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
	if !TrustedMetadata(uid, info.Mode(), false) {
		return "", false
	}

	// Parent-directory symlinks may cause the canonical path to differ even
	// when the executable itself is not a symlink. Validate the canonical path
	// again before returning it.
	evaluated, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		return "", false
	}
	evaluated, err = filepath.Abs(evaluated)
	if err != nil {
		return "", false
	}
	evaluated = filepath.Clean(evaluated)
	if !TrustedDirectoryChain(filepath.Dir(evaluated), depth) {
		return "", false
	}
	finalInfo, err := os.Stat(evaluated)
	if err != nil {
		return "", false
	}
	finalUID, ok := fileUID(finalInfo)
	if !ok || !TrustedMetadata(finalUID, finalInfo.Mode(), false) {
		return "", false
	}
	return evaluated, true
}

// TrustedDirectoryChain validates every lexical directory component. Root-owned
// symlinks are allowed only when their target chain is also root-owned and not
// writable by group or other users.
func TrustedDirectoryChain(dir string, depth int) bool {
	if depth > maxSymlinkDepth || !filepath.IsAbs(dir) {
		return false
	}
	dir = filepath.Clean(dir)
	rootInfo, err := os.Lstat(string(os.PathSeparator))
	if err != nil {
		return false
	}
	rootUID, ok := fileUID(rootInfo)
	if !ok || !TrustedMetadata(rootUID, rootInfo.Mode(), true) {
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
		uid, ok := fileUID(info)
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
			if !TrustedDirectoryChain(filepath.Clean(target), depth+1) {
				return false
			}
			resolvedInfo, err := os.Stat(current)
			if err != nil {
				return false
			}
			resolvedUID, ok := fileUID(resolvedInfo)
			if !ok || !TrustedMetadata(resolvedUID, resolvedInfo.Mode(), true) {
				return false
			}
			continue
		}
		if !TrustedMetadata(uid, info.Mode(), true) {
			return false
		}
	}
	return true
}

func fileUID(info os.FileInfo) (uint32, bool) {
	if info == nil {
		return 0, false
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat == nil {
		return 0, false
	}
	return stat.Uid, true
}

// TrustedMetadata enforces the common root-ownership and non-writable metadata
// contract used by Linux transport discovery and the AskPass parent boundary.
func TrustedMetadata(uid uint32, mode os.FileMode, directory bool) bool {
	if uid != 0 || mode.Perm()&0022 != 0 {
		return false
	}
	if directory {
		return mode.IsDir()
	}
	return mode.IsRegular() && mode.Perm()&0111 != 0
}
