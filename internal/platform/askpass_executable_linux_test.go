//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func firstTrustedExecutableIdentityCandidate() (string, bool) {
	for _, candidate := range []string{
		"/usr/bin/ssh",
		"/bin/ssh",
		"/usr/bin/sftp",
		"/bin/sftp",
		"/usr/bin/sh",
		"/bin/sh",
		"/usr/bin/true",
		"/bin/true",
	} {
		if resolved, ok := trustedAskPassExecutableIdentity(candidate, candidate); ok && resolved != "" {
			return candidate, true
		}
	}
	return "", false
}

func TestTrustedAskPassExecutableIdentityAcceptsTrustedSameImage(t *testing.T) {
	candidate, ok := firstTrustedExecutableIdentityCandidate()
	if !ok {
		t.Skip("no trusted system executable candidate is available")
	}
	resolved, ok := trustedAskPassExecutableIdentity(candidate, candidate)
	if !ok || resolved == "" {
		t.Fatalf("trusted same-inode executable identity rejected: %q", candidate)
	}
}

func TestTrustedAskPassExecutableIdentityRejectsUserControlledPath(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "ghostftp")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if resolved, ok := trustedAskPassExecutableIdentity(fake, fake); ok {
		t.Fatalf("user-controlled executable unexpectedly trusted as %q", resolved)
	}
}

func TestTrustedAskPassExecutableIdentityRejectsDifferentInode(t *testing.T) {
	candidate, ok := firstTrustedExecutableIdentityCandidate()
	if !ok {
		t.Skip("no trusted system executable candidate is available")
	}
	dir := t.TempDir()
	other := filepath.Join(dir, "other")
	if err := os.WriteFile(other, []byte("different"), 0o700); err != nil {
		t.Fatal(err)
	}
	if resolved, ok := trustedAskPassExecutableIdentity(candidate, other); ok {
		t.Fatalf("different running inode unexpectedly trusted as %q", resolved)
	}
}
