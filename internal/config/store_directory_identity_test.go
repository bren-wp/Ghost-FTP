package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRejectsReplacementDirectoryAfterBinding(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store := New(dir)

	original := filepath.Join(base, "state-original")
	if err := os.Rename(dir, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := store.Write("state.json", testState{Value: 9}); err == nil {
		t.Fatal("write unexpectedly accepted a replacement state directory")
	}
	if _, err := os.Lstat(filepath.Join(dir, "state.json")); !os.IsNotExist(err) {
		t.Fatalf("replacement directory received state data: %v", err)
	}
}

func TestStoreRejectsSymlinkedDirectoryAfterBinding(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state.json"), []byte("{\"value\":1}"), 0600); err != nil {
		t.Fatal(err)
	}
	store := New(dir)

	original := filepath.Join(base, "state-original")
	if err := os.Rename(dir, original); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "state.json"), []byte("{\"value\":99}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, dir); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	var got testState
	if _, err := store.Read("state.json", testState{Value: 7}, &got); err == nil {
		t.Fatal("read unexpectedly accepted a symlinked replacement state directory")
	}
	if got.Value == 99 {
		t.Fatal("state was read through the replacement symlink")
	}
}

func TestStoreCanBindDirectoryAfterRecoverableInitialPathFailure(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.WriteFile(dir, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	store := New(dir)

	var got testState
	if _, err := store.Read("state.json", testState{Value: 3}, &got); err == nil {
		t.Fatal("expected invalid initial state path to fail")
	}
	if err := os.Remove(dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := store.Write("state.json", testState{Value: 5}); err != nil {
		t.Fatalf("repaired state directory did not bind: %v", err)
	}
	if _, err := store.Read("state.json", testState{}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Value != 5 {
		t.Fatalf("value=%d want 5", got.Value)
	}
}
