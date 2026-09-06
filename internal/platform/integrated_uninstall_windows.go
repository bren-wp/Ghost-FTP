//go:build windows

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/brand"
)

const (
	ghostFTPUninstallKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`
	ghostFTPAppPathsKey  = `Software\Microsoft\Windows\CurrentVersion\App Paths\GhostFTP.exe`
	moveFileDelayUntilReboot = 0x4
)

var moveFileExUninstall = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func scheduleDeleteOnReboot(path string) error {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	r, _, callErr := moveFileExUninstall.Call(uintptr(unsafe.Pointer(p)), 0, moveFileDelayUntilReboot)
	if r == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return errors.New("Windows could not schedule final application-file cleanup")
	}
	return nil
}

func isInstalledGhostFTPExecutable(exe string) bool {
	dir, err := InstallDir()
	if err != nil {
		return false
	}
	expected := filepath.Clean(filepath.Join(dir, "GhostFTP.exe"))
	actual, err := filepath.Abs(exe)
	if err != nil {
		return false
	}
	return strings.EqualFold(filepath.Clean(actual), expected)
}

// HandleIntegratedUninstall implements the Windows Installed Apps uninstall
// entry without shipping a second Uninstall.exe. The installed GhostFTP.exe is
// invoked with --uninstall and removes only application-owned installation
// artifacts. User profiles/settings are deliberately preserved by default.
func HandleIntegratedUninstall(args []string) (handled bool, exitCode int) {
	if len(args) != 2 || !strings.EqualFold(strings.TrimSpace(args[1]), "--uninstall") {
		return false, 0
	}

	exe, err := os.Executable()
	if err != nil || !isInstalledGhostFTPExecutable(exe) {
		ErrorDialog(
			brand.ProductName+" Setup",
			"Uninstall is unavailable",
			"Only the installed Ghost FTP application can run the integrated uninstaller. Portable copies are never removed by this command.",
		)
		return true, 1
	}

	if !ConfirmDialog(
		brand.ProductName+" Setup",
		"Uninstall Ghost FTP?",
		"Ghost FTP application files and shortcuts will be removed. Saved profiles and settings are kept on this computer so an accidental uninstall does not destroy connection configuration.",
	) {
		return true, 0
	}

	var warnings []string
	if err := RemoveShortcuts(); err != nil {
		warnings = append(warnings, "Some shortcuts could not be removed.")
	}
	if err := DeleteRegistryKey(ghostFTPAppPathsKey); err != nil {
		warnings = append(warnings, "The Windows App Paths registration could not be removed.")
	}
	if err := DeleteRegistryKey(ghostFTPUninstallKey); err != nil {
		warnings = append(warnings, "The Windows Installed Apps entry could not be removed.")
	}
	if err := scheduleDeleteOnReboot(exe); err != nil {
		warnings = append(warnings, "The application executable could not be scheduled for final deletion. Delete it manually after closing Ghost FTP.")
	}

	message := "Ghost FTP was uninstalled. Saved profiles and settings were preserved."
	if len(warnings) > 0 {
		message += "\n\n" + strings.Join(warnings, " ")
	}
	InfoDialog(brand.ProductName+" Setup", "Uninstall completed", message)
	return true, 0
}
