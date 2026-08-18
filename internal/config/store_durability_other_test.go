//go:build !windows

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncStateDirectoryAcceptsRealDirectory(t *testing.T) {
	if err := syncStateDirectory(t.TempDir()); err != nil {
		t.Fatalf("syncStateDirectory: %v", err)
	}
}

func TestSyncStateDirectoryRejectsRegularFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := syncStateDirectory(path); err == nil {
		t.Fatal("expected regular file to be rejected")
	}
}

func TestStoreWriteLeavesNoTemporaryGenerationFiles(t *testing.T) {
	dir := t.TempDir()
	s := New(dir)
	if err := s.Write("state.json", testState{Value: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.Write("state.json", testState{Value: 2}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tmp") {
			t.Fatalf("temporary state generation leaked after successful write: %s", entry.Name())
		}
	}
	for _, name := range []string{"state.json", "state.json.previous"} {
		info, err := os.Lstat(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("missing generation %s: %v", name, err)
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			t.Fatalf("generation %s is not a regular file: %v", name, info.Mode())
		}
		if info.Mode().Perm()&0077 != 0 {
			t.Fatalf("generation %s is too permissive: %o", name, info.Mode().Perm())
		}
	}
}
