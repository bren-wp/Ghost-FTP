package platform

import (
	"path/filepath"

	"brendigo.com/byftp/internal/brand"
)

// InstallDir returns the canonical per-user ByFTP installation directory.
func InstallDir() (string, error) {
	base, err := LocalAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "Programs", brand.Company, brand.ProductName), nil
}
