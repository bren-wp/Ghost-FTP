package appdata

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/bren-wp/by-ftp/internal/brand"
	"github.com/bren-wp/by-ftp/internal/platform"
	"github.com/bren-wp/by-ftp/internal/security"
)

// Dir returns the canonical per-user data directory. Builds before 1.0.12 used
// an unnecessary nested ByFTP/ByFTP directory after the product-only branding
// transition. If that directory already exists and is a real directory, keep
// using it so saved profiles, settings and host-key state remain available.
// New installations use the simpler one-level ByFTP directory.
func Dir() (string, error) {
	base, err := platform.LocalAppData()
	if err != nil {
		return "", err
	}
	root := filepath.Join(base, brand.ProductName)
	legacyNested := filepath.Join(root, brand.ProductName)
	if info, statErr := os.Lstat(legacyNested); statErr == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(legacyNested) {
			return "", errors.New("existing ByFTP data path is not a safe directory")
		}
		return legacyNested, nil
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return "", statErr
	}
	return root, nil
}
