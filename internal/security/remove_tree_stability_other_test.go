//go:build !windows

package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadStableDirectoryRejectsPathSwapToSymlink(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	outside := filepath.Join(base, "outside")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "must-survive.txt"), []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Lstat(root)
	if err != nil {
		t.Fatal(err)
	}
	original := root + ".original"
	if err := os.Rename(root, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, root); err != nil {
		t.Fatal(err)
	}
	if _, err := readStableDirectory(root, before); err == nil {
		t.Fatal("expected path-swap protection to reject symlink replacement")
	}
	got, err := os.ReadFile(filepath.Join(outside, "must-survive.txt"))
	if err != nil || string(got) != "safe" {
		t.Fatalf("outside file changed: %q err=%v", got, err)
	}
}
