//go:build windows

package main

import (
	"os"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

// init handles the Windows Installed Apps maintenance invocation before the
// normal GUI, AskPass helper or transfer engine is initialized.
func init() {
	if handled, exitCode := platform.HandleIntegratedUninstall(os.Args); handled {
		os.Exit(exitCode)
	}
}
