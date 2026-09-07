//go:build windows

package main

import (
	"fmt"
	"path/filepath"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func registerIntegratedUninstall(appPath, currentVersion string) error {
	quoted := fmt.Sprintf("\"%s\" --uninstall", appPath)
	values := []struct {
		name  string
		value string
	}{
		{"DisplayName", brand.ProductFull},
		{"DisplayVersion", currentVersion},
		{"Publisher", brand.Company},
		{"InstallLocation", filepath.Dir(appPath)},
		{"DisplayIcon", appPath + ",0"},
		{"UninstallString", quoted},
		{"URLInfoAbout", brand.Website},
	}
	for _, item := range values {
		if err := platform.SetRegistryString(uninstallKey, item.name, item.value); err != nil {
			return err
		}
	}

	// The integrated uninstaller is intentionally interactive: it requires
	// confirmation and reports completion to the user. Do not advertise that
	// same command as QuietUninstallString. Remove the stale value left by
	// earlier installers; the surrounding registry snapshot restores it if the
	// installation transaction later rolls back.
	if err := platform.DeleteRegistryValue(uninstallKey, "QuietUninstallString"); err != nil {
		return err
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoModify", 1); err != nil {
		return err
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoRepair", 1); err != nil {
		return err
	}
	return nil
}
