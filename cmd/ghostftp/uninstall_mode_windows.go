//go:build windows

package main

import (
	"os"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	ghostFTPUninstallRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`
)

func uninstallEntryExists() (bool, error) {
	_, exists, err := platform.GetRegistryString(ghostFTPUninstallRegistryKey, "UninstallString")
	return exists, err
}

func isInteractiveUninstallInvocation(args []string) bool {
	return len(args) == 2 && strings.EqualFold(strings.TrimSpace(args[1]), "--uninstall")
}

// init handles the Windows Installed Apps maintenance invocation before the
// normal GUI, AskPass helper or transfer engine is initialized.
func init() {
	interactive := isInteractiveUninstallInvocation(os.Args)
	entryExistedBefore := false
	entryStateKnown := false
	if interactive {
		if exists, err := uninstallEntryExists(); err == nil {
			entryExistedBefore = exists
			entryStateKnown = true
		}
	}

	if handled, exitCode := platform.HandleIntegratedUninstall(os.Args); handled {
		// Exit code 0 also represents a user cancelling the confirmation dialog.
		// Remove the custom protocol only when the Installed Apps registration is
		// proven to have existed before this invocation and proven absent after it.
		// A pre-existing missing App Paths value or another partial installation
		// therefore cannot turn cancellation into destructive protocol cleanup.
		if exitCode == 0 && interactive && entryStateKnown && entryExistedBefore {
			if existsAfter, err := uninstallEntryExists(); err == nil && !existsAfter {
				if exe, err := os.Executable(); err == nil {
					_ = platform.RemoveOwnedGhostFTPProtocolRegistration(exe)
				}
			}
		}
		os.Exit(exitCode)
	}
}
