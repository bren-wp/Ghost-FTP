package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallDirectoryGuardRejectsDirectoryReplacement(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "install")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	guard, err := pinInstallDirectory(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer guard.close()

	original := filepath.Join(root, "install-original")
	if err := os.Rename(dir, original); err != nil {
		// Some Windows filesystems deny renaming a directory while the retained
		// handle is open. That behavior itself preserves the identity invariant.
		return
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := guard.verify(); err == nil {
		t.Fatal("replacement install directory was accepted")
	}
}

func TestBoundInstallRefusesReplacementDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "install")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "GhostFTP.exe")

	guard, err := pinInstallDirectory(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	backup, err := backupExistingBound(target, guard)
	if err != nil {
		guard.close()
		t.Fatal(err)
	}
	defer backup.cleanup()

	original := filepath.Join(root, "install-original")
	if err := os.Rename(dir, original); err != nil {
		// An OS-level rename denial while the guard handle is open is also a
		// fail-closed outcome, so no replacement can be installed through.
		return
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := installFile(target, []byte("replacement-payload"), &backup); err == nil {
		t.Fatal("installer accepted a replacement installation directory")
	}
	if _, err := os.Lstat(target); !os.IsNotExist(err) {
		t.Fatalf("replacement directory received activated GhostFTP.exe: %v", err)
	}
}

func TestInstallDirectoryGuardRejectsRedirectedParent(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "Programs")
	dir := filepath.Join(parent, "Ghost FTP")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}

	guard, err := pinInstallDirectory(root, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer guard.close()

	movedParent := filepath.Join(root, "Programs-original")
	if err := os.Rename(parent, movedParent); err != nil {
		return
	}
	outside := filepath.Join(root, "outside")
	if err := os.Mkdir(outside, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, parent); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if err := guard.verify(); err == nil {
		t.Fatal("redirected parent chain was accepted")
	}
}
