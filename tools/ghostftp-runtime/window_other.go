//go:build !windows

package main

import "os/exec"

func hiddenCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

func stripNativeCaptionSoon()               {}
func handleWindowAction(action string) bool { return false }
