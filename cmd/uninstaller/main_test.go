package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSameWindowsPathIsCaseInsensitiveAndNormalizesLongPrefix(t *testing.T) {
	if !sameWindowsPath(`C:\Users\A\AppData\Local\Programs\Brendigo\ByFTP\Uninstall.exe`, `c:\users\a\appdata\local\programs\brendigo\byftp\Uninstall.exe`) {
		t.Fatal("expected case-insensitive equality")
	}
	if !sameWindowsPath(`\\?\C:\Users\A\ByFTP\Uninstall.exe`, `C:\Users\A\ByFTP\Uninstall.exe`) {
		t.Fatal("expected long-path prefix normalization")
	}
}

func TestRemoveUserDataDoesNotFollowNestedSymlink(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "ByFTP")
	outside := filepath.Join(root, "outside")
	if err := os.MkdirAll(data, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(outside, "keep.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(data, "link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := removeUserData(data); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(keep); err != nil || string(b) != "keep" {
		t.Fatalf("nested symlink target was modified: data=%q err=%v", string(b), err)
	}
}
