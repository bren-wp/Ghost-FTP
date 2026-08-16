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
	// Deliberately use a different size in addition to a different filesystem
	// object. Windows filesystems may recycle a just-deleted file identifier very
	// quickly, so safe-open also compares stable metadata around the handle.
	if err := os.WriteFile(replacement, []byte(`{"value":222222}`), 0600); err != nil {
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

func TestSameStateSnapshotRejectsMetadataChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	if err := os.WriteFile(path, []byte(`{"value":1}`), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(" "); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if sameStateSnapshot(before, after) {
		t.Fatal("changed state metadata was accepted as the same stable snapshot")
	}
}
