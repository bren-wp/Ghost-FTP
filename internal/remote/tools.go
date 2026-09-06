package remote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

var systemDirectory = platform.SystemDirectory

func existingRegularFile(paths ...string) string {
	for _, candidate := range paths {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if st, err := os.Stat(candidate); err == nil && st.Mode().IsRegular() {
			return candidate
		}
	}
	return ""
}

func windowsCurlCandidates(systemDir, arch string) []string {
	systemDir = filepath.Clean(strings.TrimSpace(systemDir))
	if systemDir == "" || systemDir == "." {
		return nil
	}
	candidates := []string{filepath.Join(systemDir, "curl.exe")}
	// A 32-bit Ghost FTP process on 64-bit Windows can have System32 redirected
	// to SysWOW64. Modern Windows ships curl as a native system component, so use
	// the documented Sysnative escape hatch before declaring FTP unavailable.
	// The path is derived only from GetSystemDirectoryW; PATH and environment
	// variables are intentionally never trusted for the Windows transport.
	if arch == "386" && strings.EqualFold(filepath.Base(systemDir), "SysWOW64") {
		candidates = append(candidates, filepath.Join(filepath.Dir(systemDir), "Sysnative", "curl.exe"))
	}
	return candidates
}

func findCurl() (string, error) {
	if runtime.GOOS == "windows" {
		if systemDir, err := systemDirectory(); err == nil && systemDir != "" {
			if p := existingRegularFile(windowsCurlCandidates(systemDir, runtime.GOARCH)...); p != "" {
				return p, nil
			}
		}
		return "", errors.New("Windows FTP transport component is not available")
	}
	if p, err := exec.LookPath("curl"); err == nil {
		return p, nil
	}
	return "", errors.New("FTP transport component was not found")
}
