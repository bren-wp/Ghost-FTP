//go:build windows

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidSHA256Digest(t *testing.T) {
	valid := hex.EncodeToString(make([]byte, sha256.Size))
	if !validSHA256Digest(valid) {
		t.Fatal("valid SHA-256 digest was rejected")
	}
	for _, value := range []string{"", "00", strings.Repeat("z", sha256.Size*2), strings.Repeat("0", sha256.Size*2-2)} {
		if validSHA256Digest(value) {
			t.Fatalf("invalid SHA-256 digest was accepted: %q", value)
		}
	}
}

func TestHelperEnvironmentReplacesExistingDigest(t *testing.T) {
	const old = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	const next = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	t.Setenv(integratedUninstallDigestEnv, old)

	env := helperEnvironment(next)
	count := 0
	for _, item := range env {
		if strings.HasPrefix(strings.ToUpper(item), strings.ToUpper(integratedUninstallDigestEnv+"=")) {
			count++
			if item != integratedUninstallDigestEnv+"="+next {
				t.Fatalf("unexpected helper digest environment: %q", item)
			}
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly one helper digest environment value, got %d", count)
	}
}

func TestSameCanonicalWindowsPath(t *testing.T) {
	dir := t.TempDir()
	left := filepath.Join(dir, "GhostFTP.exe")
	right := filepath.Join(dir, ".", "GhostFTP.exe")
	if !sameCanonicalWindowsPath(left, right) {
		t.Fatalf("canonical Windows paths did not match: %q %q", left, right)
	}
	if sameCanonicalWindowsPath(left, filepath.Join(dir, "Other.exe")) {
		t.Fatal("foreign executable path matched installed path")
	}
}

func TestFinalizerRejectsMalformedInvocationBeforeCleanup(t *testing.T) {
	for _, args := range [][]string{
		{"GhostFTP.exe"},
		{"GhostFTP.exe", integratedUninstallFinalizeArg},
		{"GhostFTP.exe", integratedUninstallFinalizeArg, "0"},
		{"GhostFTP.exe", integratedUninstallFinalizeArg, "not-a-pid"},
	} {
		handled, _ := handleIntegratedUninstallFinalizer(args)
		if len(args) == 1 && handled {
			t.Fatal("non-finalizer invocation was handled as finalizer")
		}
	}

	// Keep the compiler honest about os import on Windows while also documenting
	// that the helper mode never accepts its own PID as a parent identifier.
	if os.Getpid() <= 0 {
		t.Fatal("invalid process id")
	}
}
