package remote

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidatePrivateKeyPathAcceptsRegularFile(t *testing.T) {
	key := filepath.Join(t.TempDir(), "id_test")
	if err := os.WriteFile(key, []byte("test-key"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateKeyPath(key); err != nil {
		t.Fatalf("regular private-key file was rejected: %v", err)
	}
}

func TestValidatePrivateKeyPathRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows symlink creation depends on host privileges; reparse protection is covered by the Windows build and security audit")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real-key")
	link := filepath.Join(dir, "linked-key")
	if err := os.WriteFile(target, []byte("test-key"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateKeyPath(link); err == nil {
		t.Fatal("symlink private key must be rejected")
	}
}
