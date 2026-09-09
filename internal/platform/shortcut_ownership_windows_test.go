//go:build windows

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRemoveShortcutMatchingDigestPreservesModifiedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Ghost FTP.lnk")
	if err := os.WriteFile(path, []byte("owned-shortcut"), 0600); err != nil {
		t.Fatal(err)
	}
	digest, err := stableShortcutDigest(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("user-replacement"), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := removeShortcutMatchingDigest(path, digest)
	if err == nil {
		t.Fatal("modified shortcut was accepted as installer-owned")
	}
	if removed {
		t.Fatal("modified shortcut was reported removed")
	}
	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "user-replacement" {
		t.Fatalf("modified shortcut changed: %q", got)
	}
}

func TestRemoveShortcutMatchingDigestDeletesExactOwnedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Ghost FTP.lnk")
	if err := os.WriteFile(path, []byte("owned-shortcut"), 0600); err != nil {
		t.Fatal(err)
	}
	digest, err := stableShortcutDigest(path)
	if err != nil {
		t.Fatal(err)
	}

	removed, err := removeShortcutMatchingDigest(path, digest)
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("exact owned shortcut was not removed")
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("owned shortcut still exists: %v", err)
	}
}

func TestStableShortcutDigestRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.lnk")
	if err := os.WriteFile(target, []byte("target"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "Ghost FTP.lnk")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := stableShortcutDigest(link); err == nil {
		t.Fatal("symlink shortcut was accepted as owned regular file")
	}
}

func TestSameShortcutTargetCanonicalizesCaseAndDots(t *testing.T) {
	base := t.TempDir()
	expected := filepath.Join(base, "GhostFTP.exe")
	actual := filepath.Join(base, ".", "GhostFTP.exe")
	if !sameShortcutTarget(actual, expected) {
		t.Fatal("equivalent shortcut targets were not recognized")
	}
}
