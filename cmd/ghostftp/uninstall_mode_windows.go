//go:build windows

package main

import (
	"os"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func shouldRemoveOwnedProtocolHandler(exitCode int, cleanupAuthorized bool) bool {
	return exitCode == 0 && cleanupAuthorized
}

// init handles the Windows Installed Apps maintenance invocation before the
// normal GUI, AskPass helper or transfer engine is initialized.
func init() {
	if handled, exitCode := platform.HandleIntegratedUninstall(os.Args); handled {
		// Exit code 0 is also used when the user cancels the confirmation dialog.
		// Remove the owned protocol handler only after the verified uninstall
		// helper has actually started, never by inferring success from registry
		// values that may already have been absent before this invocation.
		if shouldRemoveOwnedProtocolHandler(exitCode, platform.IntegratedUninstallCleanupAuthorized()) {
			if exe, err := os.Executable(); err == nil {
				_ = platform.RemoveOwnedGhostFTPProtocolRegistration(exe)
			}
		}
		os.Exit(exitCode)
	}
}
