//go:build windows

package main

import (
	"os"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const ghostFTPUninstallRegistryKey = `Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP`

func uninstallEntryExists() (bool, error) {
	_, exists, err := platform.GetRegistryString(ghostFTPUninstallRegistryKey, "UninstallString")
	return exists, err
}

func isInteractiveUninstallInvocation(args []string) bool {
	return len(args) == 2 && strings.EqualFold(strings.TrimSpace(args[1]), "--uninstall")
}

func shouldRemoveProtocolAfterUninstall(exitCode int, interactive, beforeKnown, beforeExists, afterKnown, afterExists bool) bool {
	return exitCode == 0 && interactive && beforeKnown && beforeExists && afterKnown && !afterExists
}

// init handles the Windows Installed Apps maintenance invocation before the
// normal GUI, AskPass helper or transfer engine is initialized.
func init() {
	interactive := isInteractiveUninstallInvocation(os.Args)
	entryExistedBefore := false
	entryStateKnownBefore := false
	if interactive {
		if exists, err := uninstallEntryExists(); err == nil {
			entryExistedBefore = exists
			entryStateKnownBefore = true
		}
	}

	if handled, exitCode := platform.HandleIntegratedUninstall(os.Args); handled {
		// Exit code 0 also represents a user cancelling the confirmation dialog.
		// Remove the custom protocol only when the Installed Apps registration is
		// proven to have existed before this invocation and proven absent after it.
		entryExistsAfter := false
		entryStateKnownAfter := false
		if interactive {
			if exists, err := uninstallEntryExists(); err == nil {
				entryExistsAfter = exists
				entryStateKnownAfter = true
			}
		}
		if shouldRemoveProtocolAfterUninstall(
			exitCode,
			interactive,
			entryStateKnownBefore,
			entryExistedBefore,
			entryStateKnownAfter,
			entryExistsAfter,
		) {
			if exe, err := os.Executable(); err == nil {
				_ = platform.RemoveOwnedGhostFTPProtocolRegistration(exe)
			}
		}
		os.Exit(exitCode)
	}
}
