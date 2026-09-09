//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidAskpassInvocationRejectsDifferentExecutableIdentity(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "GhostFTP")
	if err := os.WriteFile(other, []byte("not-the-running-image"), 0o755); err != nil {
		t.Fatal(err)
	}
	if validAskpassInvocation(exe, other, "force", validToken) {
		t.Fatal("different executable identity unexpectedly accepted as AskPass helper")
	}
}
