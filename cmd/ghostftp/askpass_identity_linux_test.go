//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func TestValidAskpassInvocationAcceptsStableRunningImage(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	stable, err := platform.StableAskPassExecutable(exe)
	if err != nil {
		t.Fatal(err)
	}
	if stable == filepath.Clean(exe) {
		t.Fatalf("Linux AskPass helper retained mutable executable pathname %q", stable)
	}
	if !validAskpassInvocation(exe, stable, "force", validToken) {
		t.Fatal("stable running executable identity was rejected")
	}
}

func TestValidAskpassInvocationRejectsDifferentExecutableIdentity(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	other := filepath.Join(t.TempDir(), "GhostFTP")
	if err := os.WriteFile(other, []byte("not-the-running-image"), 0755); err != nil {
		t.Fatal(err)
	}
	if validAskpassInvocation(exe, other, "force", validToken) {
		t.Fatal("different executable identity unexpectedly accepted as AskPass helper")
	}
}
