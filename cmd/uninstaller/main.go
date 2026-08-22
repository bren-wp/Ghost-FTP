package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/userdata"
)

const uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ByFTP`

func sameWindowsPath(a, b string) bool {
	normalize := func(path string) string {
		path = filepath.Clean(path)
		path = strings.TrimPrefix(path, `\\?\`)
		return path
	}
	return strings.EqualFold(normalize(a), normalize(b))
}

func removeOrSchedule(path string) (bool, error) {
	if err := os.Remove(path); err == nil || errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err := platform.ScheduleDeleteOnReboot(path); err != nil {
		return false, err
	}
	return true, nil
}

func userDataDirs() ([]string, error) {
	base, err := platform.LocalAppData()
	if err != nil {
		return nil, err
	}
	return []string{
		filepath.Join(base, brand.ProductName),
		userdata.LegacyDir(base, brand.ProductName),
	}, nil
}

func removeUserData(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("user data folder is unavailable")
	}
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return security.RemoveTreeNoFollow(path)
}

func main() {
	platform.HardenProcessPrivacy()
	defer func() {
		if recover() != nil {
			platform.ErrorDialog(brand.ProductFull, "Uninstall did not finish", "Restart Windows and run the ByFTP uninstaller again.")
		}
	}()
	exe, err := os.Executable()
	if err != nil {
		platform.ErrorDialog(brand.ProductFull, "Uninstall could not start", "Restart Windows and try again.")
		return
	}
	installDir, err := platform.InstallDir()
	if err != nil || !sameWindowsPath(exe, filepath.Join(installDir, "Uninstall.exe")) {
		platform.ErrorDialog(brand.ProductFull, "Uninstall could not start", "Run ByFTP removal from its installed location or from Windows Installed apps.")
		return
	}

	if !platform.ConfirmDialog(
		brand.ProductFull,
		"Uninstall ByFTP?",
		"The application will be removed. You can choose whether saved profiles and settings should also be deleted.",
	) {
		return
	}
	deleteUserData := platform.ConfirmDialog(
		brand.ProductFull,
		"Delete saved profiles and settings too?",
		"Yes = remove local ByFTP profiles and settings from this computer.\nNo = keep them for a possible future reinstall.",
	)

	var errs []error
	deferred := false
	for _, path := range []string{
		filepath.Join(installDir, "ByFTP.exe"),
		filepath.Join(installDir, "ByFTP.exe.new"),
	} {
		wasDeferred, removeErr := removeOrSchedule(path)
		deferred = deferred || wasDeferred
		if removeErr != nil {
			errs = append(errs, removeErr)
		}
	}
	if err := platform.RemoveShortcuts(); err != nil {
		errs = append(errs, err)
	}
	if err := platform.DeleteRegistryKey(uninstallKey); err != nil {
		errs = append(errs, err)
	}
	if err := platform.DeleteRegistryKey(`Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`); err != nil {
		errs = append(errs, err)
	}

	// The running uninstaller cannot remove itself. Schedule only the canonical
	// ByFTP uninstaller and install directory for cleanup.
	if err := platform.ScheduleDeleteOnReboot(exe); err != nil {
		errs = append(errs, err)
	} else {
		deferred = true
	}
	if err := platform.ScheduleDeleteOnReboot(installDir); err != nil {
		errs = append(errs, err)
	}

	if deleteUserData {
		if dataDirs, dataErr := userDataDirs(); dataErr != nil {
			errs = append(errs, dataErr)
		} else {
			for _, dataDir := range dataDirs {
				if dataErr = removeUserData(dataDir); dataErr != nil {
					errs = append(errs, dataErr)
				}
			}
		}
	}

	if len(errs) != 0 {
		platform.ErrorDialog(brand.ProductFull, "Uninstall was not fully completed", "Some files or Windows settings could not be removed. Restart Windows, then run removal again from Installed apps.")
		return
	}
	message := "Saved profiles and settings remain on this computer."
	if deleteUserData {
		message = "Local ByFTP profiles and settings were removed."
	}
	if deferred {
		message += " Final cleanup will finish after Windows restarts."
	}
	platform.InfoDialog(brand.ProductFull, "ByFTP was removed", message)
}
