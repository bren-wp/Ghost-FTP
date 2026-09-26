//go:build !windows

package main

func stripNativeCaptionSoon()               {}
func handleWindowAction(action string) bool { return false }
