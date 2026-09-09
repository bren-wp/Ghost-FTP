//go:build windows

package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func digestText(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestVerifiedRegularFileSHA256(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned.lnk")
	if err := os.WriteFile(path, []byte("owned shortcut"), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := verifiedRegularFileSHA256(path)
	if err != nil {
		t.Fatal(err)
	}
	want := digestText("owned shortcut")
	if got != want {
		t.Fatalf("digest mismatch: got %q want %q", got, want)
	}
}

func TestRemoveVerifiedRegularFileMatchingSHA256DeletesExactObject(t *testing.T) {
	path := filepath.Join(t.TempDir(), "owned.lnk")
	if err := os.WriteFile(path, []byte("owned shortcut"), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := removeVerifiedRegularFileMatchingSHA256(path, digestText("owned shortcut"))
	if err != nil {
		t.Fatal(err)
	}
	if !removed {
		t.Fatal("expected owned shortcut to be removed")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("owned shortcut still exists: %v", err)
	}
}

func TestRemoveVerifiedRegularFileMatchingSHA256PreservesMismatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "foreign.lnk")
	if err := os.WriteFile(path, []byte("foreign shortcut"), 0600); err != nil {
		t.Fatal(err)
	}

	removed, err := removeVerifiedRegularFileMatchingSHA256(path, digestText("owned shortcut"))
	if !errors.Is(err, errVerifiedOwnershipDigestMismatch) {
		t.Fatalf("expected ownership mismatch, got %v", err)
	}
	if removed {
		t.Fatal("foreign shortcut was reported removed")
	}
	if got, readErr := os.ReadFile(path); readErr != nil || string(got) != "foreign shortcut" {
		t.Fatalf("foreign shortcut was not preserved: %q %v", got, readErr)
	}
}

func TestVerifiedRegularHandleBlocksPathReplacement(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "owned.lnk")
	replacement := filepath.Join(dir, "replacement.lnk")
	if err := os.WriteFile(path, []byte("owned shortcut"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("foreign shortcut"), 0600); err != nil {
		t.Fatal(err)
	}

	f, err := openVerifiedRegularFile(path, true)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if err := os.Rename(replacement, path); err == nil {
		t.Fatal("replacement unexpectedly succeeded while verified handle was pinned")
	}
	got, err := verifiedOpenFileSHA256(f)
	if err != nil {
		t.Fatal(err)
	}
	if got != digestText("owned shortcut") {
		t.Fatalf("pinned handle changed identity/content: %q", got)
	}
}

func TestVerifiedRegularFileRejectsDirectory(t *testing.T) {
	if _, err := verifiedRegularFileSHA256(t.TempDir()); err == nil {
		t.Fatal("directory was accepted as a verified regular file")
	}
}
