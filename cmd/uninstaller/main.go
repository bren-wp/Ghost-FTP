package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bren-wp/by-ftp/internal/brand"
	"github.com/bren-wp/by-ftp/internal/platform"
	"github.com/bren-wp/by-ftp/internal/security"
)

const (
	uninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\ByFTP`
	appPathsKey  = `Software\Microsoft\Windows\CurrentVersion\App Paths\ByFTP.exe`

	messageBoxTitleActionFailed = "Action did not finish"
)

func normalizeWindowsPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("putanja nije dostupna")
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}

	path = filepath.Clean(abs)

	// Convert Win32 extended-length paths back to their ordinary form before
	// comparing them. \\?\UNC\server\share must become \\server\share.
	if strings.HasPrefix(strings.ToUpper(path), `\\?\UNC\`) {
		path = `\\` + path[len(`\\?\UNC\`):]
	} else if strings.HasPrefix(path, `\\?\`) {
		path = path[len(`\\?\`):]
	}

	return filepath.Clean(path), nil
}

func sameWindowsPath(a, b string) bool {
	normalizedA, err := normalizeWindowsPath(a)
	if err != nil {
		return false
	}

	normalizedB, err := normalizeWindowsPath(b)
	if err != nil {
		return false
	}

	return strings.EqualFold(normalizedA, normalizedB)
}

func validateInstallDir(dir string) error {
	if strings.TrimSpace(dir) == "" {
		return errors.New("instalacijska mapa nije dostupna")
	}

	localAppData, err := platform.LocalAppData()
	if err != nil {
		return err
	}

	if err := security.EnsureNoRedirectDirectory(localAppData, dir); err != nil {
		return errors.New("instalacijska mapa sadrži nesigurno preusmjeravanje")
	}

	info, err := os.Lstat(dir)
	if err != nil {
		return err
	}

	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(dir) {
		return errors.New("instalacijska putanja nije sigurna mapa")
	}

	return nil
}

func validateUninstallFile(path, installDir string) error {
	if !sameWindowsPath(filepath.Dir(path), installDir) {
		return errors.New("datoteka nije u ByFTP instalacijskoj mapi")
	}

	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() ||
		info.Mode()&os.ModeSymlink != 0 ||
		security.IsReparsePoint(path) {
		return errors.New("ciljna datoteka nije sigurna regularna datoteka")
	}

	return nil
}

func removeOrSchedule(path, installDir string) (bool, error) {
	if err := validateUninstallFile(path, installDir); err != nil {
		return false, err
	}

	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if err := os.Remove(path); err == nil || errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	// Revalidate immediately before scheduling a delayed delete. This prevents
	// blindly scheduling an unexpected reparse point or directory after the
	// direct removal attempt failed.
	if err := validateUninstallFile(path, installDir); err != nil {
		return false, err
	}

	if err := platform.ScheduleDeleteOnReboot(path); err != nil {
		return false, err
	}

	return true, nil
}

func userDataDir() (string, error) {
	base, err := platform.LocalAppData()
	if err != nil {
		return "", err
	}

	base, err = filepath.Abs(base)
	if err != nil {
		return "", err
	}

	dir := filepath.Join(base, brand.Company, brand.ProductName)
	dir, err = filepath.Abs(dir)
	if err != nil {
		return "", err
	}

	rel, err := filepath.Rel(base, dir)
	if err != nil {
		return "", err
	}

	if rel == ".." || strings.HasPrefix(rel, `..`+string(filepath.Separator)) {
		return "", errors.New("korisnička mapa nije unutar lokalne korisničke mape")
	}

	return filepath.Clean(dir), nil
}

func removeUserData(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("user data folder is unavailable")
	}

	return security.RemoveTreeNoFollow(path)
}

func dialogTitle() string {
	return brand.ProductFull + " — " + brand.Company
}

func showUninstallError(heading, message string) {
	platform.ErrorDialog(dialogTitle(), heading, message)
}

func runUninstaller() (exitCode int) {
	defer func() {
		if recover() != nil {
			showUninstallError(
				messageBoxTitleActionFailed,
				"Uninstall did not finish. Restart Windows and try again.",
			)
			exitCode = 1
		}
	}()

	exe, err := os.Executable()
	if err != nil {
		showUninstallError(
			"Uninstall was not started",
			"ByFTP cannot be removed right now. Restart Windows and try again.",
		)
		return 1
	}

	exe, err = filepath.Abs(exe)
	if err != nil {
		showUninstallError(
			"Uninstall was not started",
			"Lokaciju programa za deinstalaciju nije moguće potvrditi.",
		)
		return 1
	}

	installDir, err := platform.InstallDir()
	if err != nil {
		showUninstallError(
			"Uninstall was not started",
			"ByFTP instalacijsku mapu nije moguće potvrditi.",
		)
		return 1
	}

	installDir, err = filepath.Abs(installDir)
	if err != nil {
		showUninstallError(
			"Uninstall was not started",
			"ByFTP instalacijsku mapu nije moguće potvrditi.",
		)
		return 1
	}

	expectedUninstaller := filepath.Join(installDir, "Uninstall.exe")
	if !sameWindowsPath(exe, expectedUninstaller) {
		showUninstallError(
			"Uninstall was not started",
			"Start the ByFTP uninstaller from its installed location or from Windows Installed apps.",
		)
		return 1
	}

	if err := validateInstallDir(installDir); err != nil {
		showUninstallError(
			"Uninstall was not started",
			"ByFTP instalacijska mapa nije sigurna ili je preusmjerena. Uklanjanje je zaustavljeno.",
		)
		return 1
	}

	if err := validateUninstallFile(exe, installDir); err != nil {
		showUninstallError(
			"Uninstall was not started",
			"Program za deinstalaciju nije moguće sigurno potvrditi.",
		)
		return 1
	}

	if !platform.ConfirmDialog(
		dialogTitle(),
		"Remove "+brand.ProductFull+"?",
		"The application will be removed. You can then choose whether to keep saved profiles and settings.",
	) {
		return 0
	}

	deleteUserData := platform.ConfirmDialog(
		dialogTitle(),
		"Delete saved profiles and settings too?",
		"Yes = delete local ByFTP profiles and settings from this computer.\nNo = keep them for a possible reinstall.",
	)

	deferred := false
	var cleanupErrs []error

	for _, name := range []string{
		"ByFTP.exe",
		"ByFTP.exe.new",
	} {
		path := filepath.Join(installDir, name)

		wasDeferred, removeErr := removeOrSchedule(path, installDir)
		deferred = deferred || wasDeferred

		if removeErr != nil {
			cleanupErrs = append(
				cleanupErrs,
				fmt.Errorf("uklanjanje %s: %w", name, removeErr),
			)
		}
	}

	// Do not remove the uninstall registration or this uninstaller when the
	// application binaries could not be removed or scheduled. Keeping the
	// uninstall entry allows the user to retry from Windows Settings.
	if len(cleanupErrs) != 0 {
		showUninstallError(
			"Uninstall did not fully finish",
			"Neke ByFTP datoteke nije bilo moguće ukloniti. Ponovno pokrenite računalo pa ponovno pokrenite uklanjanje iz Windows postavki aplikacija.",
		)
		return 1
	}

	var warnings []error

	if err := platform.RemoveShortcuts(); err != nil {
		warnings = append(warnings, fmt.Errorf("uklanjanje prečaca: %w", err))
	}

	if deleteUserData {
		dataDir, dataErr := userDataDir()
		if dataErr != nil {
			warnings = append(warnings, fmt.Errorf("određivanje korisničke mape: %w", dataErr))
		} else if dataErr = removeUserData(dataDir); dataErr != nil {
			warnings = append(warnings, fmt.Errorf("uklanjanje korisničkih podataka: %w", dataErr))
		}
	}

	// Remove App Paths first. Keep the uninstall registration until all core
	// cleanup steps have succeeded so Windows Settings can still retry if
	// something fails.
	if err := platform.DeleteRegistryKey(appPathsKey); err != nil {
		showUninstallError(
			"Uninstall did not fully finish",
			"Windows registraciju ByFTP-a nije bilo moguće potpuno ukloniti. Ponovno pokrenite računalo i pokušajte ponovno.",
		)
		return 1
	}

	if err := platform.DeleteRegistryKey(uninstallKey); err != nil {
		showUninstallError(
			"Uninstall did not fully finish",
			"Windows zapis za deinstalaciju nije bilo moguće ukloniti. Ponovno pokrenite računalo i pokušajte ponovno.",
		)
		return 1
	}

	// The running uninstaller cannot remove itself. Only schedule the already
	// validated canonical uninstaller and canonical installation directory.
	if err := platform.ScheduleDeleteOnReboot(exe); err != nil {
		showUninstallError(
			"Uninstall did not fully finish",
			"Program za deinstalaciju nije moguće označiti za završno uklanjanje nakon ponovnog pokretanja sustava Windows.",
		)
		return 1
	}
	deferred = true

	if err := platform.ScheduleDeleteOnReboot(installDir); err != nil {
		warnings = append(
			warnings,
			fmt.Errorf("odgođeno uklanjanje instalacijske mape: %w", err),
		)
	}

	message := "User profiles and settings were kept on this computer."
	if deleteUserData {
		message = "Local ByFTP profiles and settings were deleted."
	}

	if deferred {
		message += " Final cleanup will finish after Windows restarts."
	}

	if len(warnings) != 0 {
		message += "\n\nByFTP was removed, ali neke pomoćne stavke nije bilo moguće potpuno očistiti."
	}

	platform.InfoDialog(
		dialogTitle(),
		"ByFTP was removed",
		message,
	)

	return 0
}

func main() {
	platform.HardenProcessPrivacy()
	os.Exit(runUninstaller())
}
