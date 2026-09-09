//go:build linux

package remote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/linuxtrust"
)

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

// Keep package-local wrappers so the transport regression suite continues to
// exercise the shared provenance primitive through the same public behavior.
func trustedLinuxTransportExecutable(candidate string) (string, bool) {
	return linuxtrust.TrustedExecutable(candidate)
}

func trustedLinuxDirectoryChain(dir string, depth int) bool {
	return linuxtrust.TrustedDirectoryChain(dir, depth)
}

func trustedLinuxMetadata(uid uint32, mode os.FileMode, directory bool) bool {
	return linuxtrust.TrustedMetadata(uid, mode, directory)
}
