package main

import (
	"os"

	"github.com/bren-wp/Ghost-FTP/internal/desktop"
)

// initializeDesktopLaunchTarget captures only a fully validated Ghost FTP
// protocol argument. Invalid or unrelated command-line input is ignored so it
// cannot interfere with AskPass, uninstall or ordinary application startup
// modes.
func initializeDesktopLaunchTarget(args []string) bool {
	target, found, err := desktopLaunchArgument(args)
	if !found || err != nil {
		return false
	}
	desktop.SetStartupTarget(desktop.StartupTarget{
		Protocol: target.Protocol,
		Host:     target.Host,
		Port:     target.Port,
		Username: target.Username,
		Path:     target.Path,
	})
	return true
}

func init() {
	initializeDesktopLaunchTarget(os.Args[1:])
}
