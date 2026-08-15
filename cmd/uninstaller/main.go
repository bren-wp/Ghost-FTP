package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"brendigo.com/byftp/internal/brand"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
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

func userDataDir() (string, error) {
	base, err := platform.LocalAppData()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, brand.Company, brand.ProductName), nil
}

func removeUserData(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("korisnička mapa nije dostupna")
	}
	return security.RemoveTreeNoFollow(path)
}

func main() {
	platform.HardenProcessPrivacy()
	defer func() {
		if recover() != nil {
			platform.ErrorDialog(brand.ProductFull+" — "+brand.Company, "Radnja nije dovršena", "Uklanjanje nije dovršeno. Ponovno pokrenite računalo i pokušajte ponovno.")
		}
	}()
	exe, err := os.Executable()
	if err != nil {
		platform.ErrorDialog(brand.ProductFull+" — "+brand.Company, "Deinstalacija nije pokrenuta", "ByFTP trenutačno nije moguće ukloniti. Ponovno pokrenite računalo i pokušajte ponovno.")
		return
	}
	installDir, err := platform.InstallDir()
	if err != nil || !sameWindowsPath(exe, filepath.Join(installDir, "Uninstall.exe")) {
		platform.ErrorDialog(brand.ProductFull+" — "+brand.Company, "Deinstalacija nije pokrenuta", "Pokrenite uklanjanje ByFTP-a iz njegove instalirane lokacije ili iz Windows postavki aplikacija.")
		return
	}

	if !platform.ConfirmDialog(
		brand.ProductFull+" — "+brand.Company,
		"Ukloniti "+brand.ProductFull+"?",
		"Aplikacija će biti uklonjena. Nakon toga možete odlučiti želite li zadržati spremljene profile i postavke.",
	) {
		return
	}
	deleteUserData := platform.ConfirmDialog(
		brand.ProductFull+" — "+brand.Company,
		"Obrisati i spremljene profile i postavke?",
		"Da = brišu se lokalni ByFTP profili i postavke s ovog računala.\nNe = ostaju sačuvani za moguću ponovnu instalaciju.",
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
	// ByFTP uninstaller and its canonical install directory for cleanup.
	if err := platform.ScheduleDeleteOnReboot(exe); err != nil {
		errs = append(errs, err)
	} else {
		deferred = true
	}
	if err := platform.ScheduleDeleteOnReboot(installDir); err != nil {
		errs = append(errs, err)
	}

	if deleteUserData {
		if dataDir, dataErr := userDataDir(); dataErr != nil {
			errs = append(errs, dataErr)
		} else if dataErr = removeUserData(dataDir); dataErr != nil {
			errs = append(errs, dataErr)
		}
	}

	if len(errs) != 0 {
		platform.ErrorDialog(brand.ProductFull+" — "+brand.Company, "Uklanjanje nije potpuno dovršeno", "Neke datoteke ili Windows postavke nije bilo moguće ukloniti. Ponovno pokrenite računalo pa ponovno pokrenite uklanjanje iz Windows postavki aplikacija.")
		return
	}
	message := "Korisnički profili i postavke ostali su sačuvani na ovom računalu."
	if deleteUserData {
		message = "Lokalni ByFTP profili i postavke su obrisani."
	}
	if deferred {
		message += " Završno čišćenje dovršit će se nakon ponovnog pokretanja sustava Windows."
	}
	platform.InfoDialog(brand.ProductFull+" — "+brand.Company, "ByFTP je uklonjen", message)
}
