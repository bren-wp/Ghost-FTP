//go:build !windows

package main

import (
	"fmt"
	"os/exec"
)

func hiddenCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func stripNativeCaptionSoon()        {}
func handleWindowAction(string) bool { return false }

func chooseInstallFolder() (string, error) {
	return "", fmt.Errorf("folder picker is available on Windows only")
}
