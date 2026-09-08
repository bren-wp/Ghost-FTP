//go:build !windows

package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenStableRootDirectoryRejectsPathSwapToSymlink(t *testing.T) {
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

	parent, err := os.OpenRoot(base)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	before, err := parent.Lstat("root")
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
	opened, _, err := openStableRootDirectory(parent, "root", before)
	if opened != nil {
		opened.Close()
	}
	if err == nil {
		t.Fatal("expected path-swap protection to reject symlink replacement")
	}
	got, err := os.ReadFile(filepath.Join(outside, "must-survive.txt"))
	if err != nil || string(got) != "safe" {
		t.Fatalf("outside file changed: %q err=%v", got, err)
	}
}

func TestRemoveTreeNoFollowDoesNotTraverseSwappedRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	original := root + ".original"
	outside := filepath.Join(base, "outside")

	if err := os.MkdirAll(filepath.Join(root, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "child", "delete-me.txt"), []byte("delete"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(outside, "child"), 0700); err != nil {
		t.Fatal(err)
	}
	outsideFile := filepath.Join(outside, "child", "must-survive.txt")
	if err := os.WriteFile(outsideFile, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}

	swapped := false
	hooks := &removeTreeHooks{beforeDescend: func() {
		if swapped {
			return
		}
		if err := os.Rename(root, original); err != nil {
			t.Fatalf("rename root: %v", err)
		}
		if err := os.Symlink(outside, root); err != nil {
			t.Fatalf("replace root with symlink: %v", err)
		}
		swapped = true
	}}

	if err := removeTreeNoFollowWithHooks(root, hooks); err == nil {
		t.Fatal("expected root identity change to abort deletion")
	}
	if !swapped {
		t.Fatal("test did not perform the deterministic root swap")
	}
	got, err := os.ReadFile(outsideFile)
	if err != nil || string(got) != "safe" {
		t.Fatalf("outside file changed through swapped root: %q err=%v", got, err)
	}
}
