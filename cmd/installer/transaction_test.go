package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBackupExistingRollbackRestoresPreviousFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "ByFTP.exe")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	if !b.existed() {
		t.Fatal("expected existing-file backup")
	}
	if err := os.WriteFile(target, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := b.rollback(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "old" {
		t.Fatalf("rollback = %q, want old", got)
	}
}

func TestBackupExistingRollbackRemovesFreshFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "ByFTP.exe")
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	if b.existed() {
		t.Fatal("fresh target unexpectedly marked as existing")
	}
	if err := os.WriteFile(target, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := b.rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("fresh rollback should remove target, stat err=%v", err)
	}
}

func TestBackupExistingRejectsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	realTarget := filepath.Join(dir, "real.exe")
	if err := os.WriteFile(realTarget, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "ByFTP.exe")
	if err := os.Symlink(realTarget, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := backupExisting(link); err == nil {
		t.Fatal("expected symlink target to be rejected")
	}
}
