package platform

import (
	"path/filepath"

	"github.com/bren-wp/by-ftp/internal/brand"
)

// InstallDir returns the canonical per-user ByFTP installation directory.
func InstallDir() (string, error) {
	base, err := LocalAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Programs", brand.Company, brand.ProductName), nil
}
