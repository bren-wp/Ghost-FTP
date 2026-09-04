package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestBackupExistingRollbackRestoresPreviousFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
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
	if err := installFile(target, []byte("new"), &b); err != nil {
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

func TestBackupExistingRollbackRemovesFreshInstalledFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	if b.existed() {
		t.Fatal("fresh target unexpectedly marked as existing")
	}
	if err := installFile(target, []byte("new"), &b); err != nil {
		t.Fatal(err)
	}
	if err := b.rollback(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("fresh rollback should remove installer-owned target, stat err=%v", err)
	}
}

func TestBackupExistingRejectsSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	realTarget := filepath.Join(dir, "real.exe")
	if err := os.WriteFile(realTarget, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "GhostFTP.exe")
	if err := os.Symlink(realTarget, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := backupExisting(link); err == nil {
		t.Fatal("expected symlink target to be rejected")
	}
}

func TestBackupExistingRejectsLstatOpenReplacement(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
	originalMoved := filepath.Join(dir, "original.exe")
	replacement := filepath.Join(dir, "replacement.exe")
	if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(replacement, []byte("replacement"), 0644); err != nil {
		t.Fatal(err)
	}

	oldOpen := openInstallerBackupSource
	openInstallerBackupSource = func(name string) (*os.File, error) {
		if err := os.Rename(name, originalMoved); err != nil {
			return nil, err
		}
		if err := os.Rename(replacement, name); err != nil {
			return nil, err
		}
		return os.Open(name)
	}
	t.Cleanup(func() { openInstallerBackupSource = oldOpen })

	if _, err := backupExisting(target); err == nil {
		t.Fatal("Lstat->Open target replacement was accepted")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "replacement" {
		t.Fatalf("replacement path was unexpectedly modified: %q", got)
	}
}

func TestFreshInstallDoesNotOverwriteTargetThatAppearedAfterSnapshot(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("external"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installFile(target, []byte("GhostFTP"), &b); err == nil {
		t.Fatal("fresh install overwrote a target that appeared after snapshot")
	}
	if err := b.rollback(); err != nil {
		t.Fatalf("rollback before activation should be a no-op: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "external" {
		t.Fatalf("external target was modified or deleted: %q", got)
	}
}

func TestUpgradeRejectsOriginalChangedAfterBackup(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
	if err := os.WriteFile(target, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	defer b.cleanup()
	if err := os.WriteFile(target, []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := installFile(target, []byte("new"), &b); err == nil {
		t.Fatal("upgrade accepted an original changed after backup")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "changed" {
		t.Fatalf("changed original was overwritten: %q", got)
	}
}

func TestRollbackRefusesInstalledFileChangedByAnotherActor(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := installFile(target, []byte("GhostFTP"), &b); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("external-change"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := b.rollback(); err == nil {
		t.Fatal("rollback removed a target modified after installer activation")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "external-change" {
		t.Fatalf("rollback modified externally changed target: %q", got)
	}
}

func TestRollbackBeforeActivationNeverDeletesFreshExternalTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "GhostFTP.exe")
	b, err := backupExisting(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("external"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := b.rollback(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if string(got) != "external" {
		t.Fatalf("pre-activation rollback deleted external target: %q", got)
	}
}
