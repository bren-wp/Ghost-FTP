package platform

import (
	"path/filepath"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

// InstallDir returns the canonical per-user GhostFTP installation directory.
func InstallDir() (string, error) {
	base, err := LocalAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Programs", brand.Company, brand.ProductName), nil
}
