package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRenameNoReplacePreservesExistingDestination(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("destination"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RenameNoReplace(src, dst); err == nil {
		t.Fatal("expected no-replace move to reject existing destination")
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "destination" {
		t.Fatalf("destination was overwritten: %q", got)
	}
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("source should remain after rejected move: %v", err)
	}
}

func TestRenameNoReplaceMovesWhenDestinationMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := RenameNoReplace(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Fatalf("source still exists: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != "source" {
		t.Fatalf("destination mismatch: %q err=%v", got, err)
	}
}
