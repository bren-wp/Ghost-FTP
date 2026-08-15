package localfs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteMissingItemReturnsError(t *testing.T) {
	s := New()
	if err := s.Delete(t.TempDir(), "missing.txt"); err == nil {
		t.Fatal("expected missing local item to return an error")
	}
}

func TestRenameDoesNotOverwriteExistingItem(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0600); err != nil {
		t.Fatal(err)
	}
	s := New()
	if err := s.Rename(dir, "a.txt", "b.txt"); err == nil {
		t.Fatal("expected rename over existing item to fail")
	}
	got, err := os.ReadFile(filepath.Join(dir, "b.txt"))
	if err != nil || string(got) != "b" {
		t.Fatalf("destination changed unexpectedly: %q err=%v", got, err)
	}
}

func TestDeleteRemovesSymlinkNotTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "link.txt")
	if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := New().Delete(dir, "link.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(link); !os.IsNotExist(err) {
		t.Fatalf("link still exists: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "keep" {
		t.Fatalf("target was modified: %q err=%v", got, err)
	}
}

func TestListMarksLinkLikeEntryAsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(dir, "linked")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	_, items, err := New().List(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Name == "linked" {
			if !item.IsSymlink || item.IsDirectory {
				t.Fatalf("link-like entry exposed as normal directory: %#v", item)
			}
			return
		}
	}
	t.Fatal("link entry missing")
}
