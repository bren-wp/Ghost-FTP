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
		{"QuietUninstallString", quoted},
		{"URLInfoAbout", brand.Website},
	}
	for _, item := range values {
		if err := platform.SetRegistryString(uninstallKey, item.name, item.value); err != nil {
			return err
		}
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoModify", 1); err != nil {
		return err
	}
	if err := platform.SetRegistryDWORD(uninstallKey, "NoRepair", 1); err != nil {
		return err
	}
	return nil
}
