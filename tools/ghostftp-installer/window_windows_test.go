//go:build windows

package main

import "testing"

func TestHiddenCommandUsesNoWindow(t *testing.T) {
	cmd := hiddenCommand("cmd.exe", "/C", "exit 0")
	if cmd.SysProcAttr == nil {
		t.Fatal("hiddenCommand must configure SysProcAttr on Windows")
	}
	if !cmd.SysProcAttr.HideWindow {
		t.Fatal("hiddenCommand must set HideWindow")
	}
	if cmd.SysProcAttr.CreationFlags&createNoWindow == 0 {
		t.Fatalf("hiddenCommand must include CREATE_NO_WINDOW, got %#x", cmd.SysProcAttr.CreationFlags)
	}
}
