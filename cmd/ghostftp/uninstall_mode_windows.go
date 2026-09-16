//go:build windows

package main

import (
	"os"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	ghostFTPUninstallRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`
	ghostFTPAppPathsRegistryKey  = `Software\Microsoft\Windows\CurrentVersion\App Paths\GhostFTP.exe`
)

func integratedUninstallRegistrationRemoved() bool {
	_, appPathExists, appPathErr := platform.GetRegistryString(ghostFTPAppPathsRegistryKey, "")
	_, uninstallExists, uninstallErr := platform.GetRegistryString(ghostFTPUninstallRegistryKey, "UninstallString")
	if appPathErr != nil || uninstallErr != nil {
		return false
	}
	return !appPathExists || !uninstallExists
}

// init handles the Windows Installed Apps maintenance invocation before the
// normal GUI, AskPass helper or transfer engine is initialized.
func init() {
	if handled, exitCode := platform.HandleIntegratedUninstall(os.Args); handled {
		if exitCode == 0 && len(os.Args) == 2 && strings.EqualFold(strings.TrimSpace(os.Args[1]), "--uninstall") && integratedUninstallRegistrationRemoved() {
			if exe, err := os.Executable(); err == nil {
				_ = platform.RemoveOwnedGhostFTPProtocolRegistration(exe)
			}
		}
		os.Exit(exitCode)
	}
}
