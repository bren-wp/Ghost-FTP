package remote

import (
	"bytes"
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

func TestSnapshotPrivateKeyCreatesIndependentPrivateCopy(t *testing.T) {
	dir := t.TempDir()
	key := filepath.Join(dir, "id_test")
	original := []byte("-----BEGIN TEST KEY-----\nsecret\n-----END TEST KEY-----\n")
	if err := os.WriteFile(key, original, 0600); err != nil {
		t.Fatal(err)
	}
	copyPath, err := snapshotPrivateKey(dir, key)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(copyPath)
	if copyPath == key {
		t.Fatal("snapshot must not reuse the original private-key path")
	}
	if err := os.WriteFile(key, []byte("replacement"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("snapshot changed with original path: %q", got)
	}
	st, err := os.Lstat(copyPath)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		t.Fatalf("snapshot is not a regular file: %v", st.Mode())
	}
	// Unix permission bits are meaningful on Unix. Windows file privacy is
	// enforced by the GhostFTP-owned no-redirect session directory and Windows ACL
	// semantics; Go's synthetic FileMode commonly reports 0666 on NTFS even
	// after Chmod(0600), so asserting POSIX bits there would be a false failure.
	if runtime.GOOS != "windows" && st.Mode().Perm()&0077 != 0 {
		t.Fatalf("snapshot permissions are not private: %o", st.Mode().Perm())
	}
}

func TestValidatePrivateKeyPathRejectsOversizedFile(t *testing.T) {
	key := filepath.Join(t.TempDir(), "id_too_large")
	f, err := os.Create(key)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(maxPrivateKeySize + 1); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if err := validatePrivateKeyPath(key); err == nil {
		t.Fatal("oversized private key must be rejected")
	}
}
