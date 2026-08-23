package remote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/bren-wp/by-ftp/internal/platform"
)

var systemDirectory = platform.SystemDirectory

func findCurl() (string, error) {
	if runtime.GOOS == "windows" {
		if systemDir, err := systemDirectory(); err == nil && systemDir != "" {
			p := filepath.Join(systemDir, "curl.exe")
			if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
				return p, nil
			}
		}
		return "", errors.New("Windows mrežna komponenta za FTP nije dostupna")
	}
	if p, err := exec.LookPath("curl"); err == nil {
		return p, nil
	}
	return "", errors.New("mrežna komponenta za FTP nije pronađena")
}
