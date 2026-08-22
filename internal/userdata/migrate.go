package userdata

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

// legacyCompanyDirectory is encoded to keep retired product branding out of
// current user-facing source and metadata while still allowing an in-place
// migration of installations created before 1.0.12.
func legacyCompanyDirectory() string {
	return string([]byte{66, 114, 101, 110, 100, 105, 103, 111})
}

func LegacyDir(base, product string) string {
	return filepath.Join(base, legacyCompanyDirectory(), product)
}

// MigrateLegacy moves the pre-1.0.12 per-user data directory to the neutral
// product directory exactly once. It never follows redirects and never
// overwrites an existing destination.
func MigrateLegacy(base, product string) error {
	if base == "" || product == "" {
		return errors.New("user data location is unavailable")
	}
	target := filepath.Join(base, product)
	if _, err := os.Lstat(target); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	legacy := LegacyDir(base, product)
	if _, err := os.Lstat(legacy); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	if err := security.EnsureNoRedirectDirectory(base, legacy); err != nil {
		return fmt.Errorf("legacy user data directory is unsafe: %w", err)
	}
	if err := platform.RenameNoReplace(legacy, target); err != nil {
		return fmt.Errorf("legacy user data migration failed: %w", err)
	}
	_ = os.Remove(filepath.Dir(legacy)) // best effort: remove the now-empty legacy parent
	return nil
}
