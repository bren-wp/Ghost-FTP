package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadLimitedRejectsLstatOpenPathSwap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	outside := filepath.Join(t.TempDir(), "outside.json")
	if err := os.WriteFile(path, []byte(`{"value":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte(`{"value":99}`), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := readLimitedWithOpen(path, func(name string) (*os.File, error) {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := os.Symlink(outside, name); err != nil {
			t.Skipf("symlink not available: %v", err)
		}
		return os.Open(name)
	})
	if err == nil {
		t.Fatal("state path replacement between Lstat and Open was accepted")
	}
}

func TestReadLimitedRejectsDifferentRegularFileAfterLstat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	replacement := filepath.Join(dir, "replacement.json")
	if err := os.WriteFile(path, []byte(`{"value":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte(`{"value":2}`), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := readLimitedWithOpen(path, func(name string) (*os.File, error) {
		if err := os.Remove(name); err != nil {
			return nil, err
		}
		if err := os.Rename(replacement, name); err != nil {
			return nil, err
		}
		return os.Open(name)
	})
	if err == nil {
		t.Fatal("different regular file swapped in between Lstat and Open was accepted")
	}
}
