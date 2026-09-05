//go:build linux

package platform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePrivateRuntimeDirectory(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateRuntimeDirectory(dir, os.Geteuid()); err != nil {
		t.Fatalf("private directory rejected: %v", err)
	}
	if err := os.Chmod(dir, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateRuntimeDirectory(dir, os.Geteuid()); err == nil {
		t.Fatal("group-accessible runtime directory was accepted")
	}
}

func TestAcquireSingleInstanceRejectsConcurrentOwner(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", dir)

	release, ok := AcquireSingleInstance("Ghost FTP test")
	if !ok {
		t.Fatal("first instance lock was not acquired")
	}
	defer release()

	secondRelease, secondOK := AcquireSingleInstance("Ghost FTP test")
	if secondOK {
		secondRelease()
		t.Fatal("second instance unexpectedly acquired the same lock")
	}

	release()
	thirdRelease, thirdOK := AcquireSingleInstance("Ghost FTP test")
	if !thirdOK {
		t.Fatal("instance lock was not reacquired after release")
	}
	thirdRelease()
}

func TestAcquireSingleInstanceRejectsSymlinkLock(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_RUNTIME_DIR", dir)

	victim := filepath.Join(dir, "victim")
	if err := os.WriteFile(victim, []byte("do-not-touch"), 0o600); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(dir, lockFileName("symlink-test"))
	if err := os.Symlink(victim, lockPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	release, ok := AcquireSingleInstance("symlink-test")
	if ok {
		release()
		t.Fatal("symlink lock path was accepted")
	}
	data, err := os.ReadFile(victim)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "do-not-touch" {
		t.Fatal("symlink target was modified")
	}
}

func TestLockFileNameIsPathSafe(t *testing.T) {
	name := lockFileName("Ghost FTP/../../evil")
	if filepath.Base(name) != name {
		t.Fatalf("lock filename escaped its directory: %q", name)
	}
	if name == "" || name[0] == '.' {
		t.Fatalf("unexpected lock filename: %q", name)
	}
}
