//go:build !windows

package main

import "fmt"

func stripNativeCaptionSoon()        {}
func handleWindowAction(string) bool { return false }

func chooseInstallFolder() (string, error) {
	return "", fmt.Errorf("folder picker is available on Windows only")
}
