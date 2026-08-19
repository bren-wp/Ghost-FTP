package remote

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestPrepareUploadSourceSurvivesOriginalPathReplacement(t *testing.T) {
	dir := t.TempDir()
	local := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(local, []byte("original-content"), 0600); err != nil {
		t.Fatal(err)
	}
	snap, err := prepareUploadSource(local)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()

	moved := local + ".old"
	if err := os.Rename(local, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("replacement-content"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(snap.Path())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original-content" {
		t.Fatalf("snapshot followed replacement path: %q", got)
	}
	if err := snap.Verify(); err != nil {
		t.Fatalf("independent snapshot did not remain valid: %v", err)
	}
}

func TestUploadSourceSnapshotDetectsSnapshotMutation(t *testing.T) {
	local := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(local, []byte("original-content"), 0600); err != nil {
		t.Fatal(err)
	}
	snap, err := prepareUploadSource(local)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	if err := os.WriteFile(snap.Path(), []byte("changed-and-longer-content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := snap.Verify(); err == nil {
		t.Fatal("changed upload snapshot must be rejected")
	}
}

func TestUploadSourceSnapshotDigestRejectsSameSizeAndMtimeTampering(t *testing.T) {
	local := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(local, []byte("original-content"), 0600); err != nil {
		t.Fatal(err)
	}
	snap, err := prepareUploadSource(local)
	if err != nil {
		t.Fatal(err)
	}
	defer snap.Close()
	originalMtime := snap.initial.ModTime()
	replacement := []byte("changed-content?")
	if int64(len(replacement)) != snap.initial.Size() {
		t.Fatalf("test replacement size mismatch: got=%d want=%d", len(replacement), snap.initial.Size())
	}
	if err := os.WriteFile(snap.Path(), replacement, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(snap.Path(), originalMtime, originalMtime); err != nil {
		t.Fatal(err)
	}
	if err := snap.Verify(); err == nil {
		t.Fatal("SHA-256 verification must reject same-size/same-mtime content tampering")
	}
}

func TestUploadSourceSnapshotCloseRemovesTempDirectory(t *testing.T) {
	local := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(local, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	snap, err := prepareUploadSource(local)
	if err != nil {
		t.Fatal(err)
	}
	tempDir := snap.dir
	if _, err := os.Stat(tempDir); err != nil {
		t.Fatal(err)
	}
	if err := snap.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(tempDir); !os.IsNotExist(err) {
		t.Fatalf("snapshot temp directory remains after close: %v", err)
	}
}

func TestPrepareUploadSourceRejectsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows symlink creation depends on host privileges; reparse validation is covered by production checks")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "target.txt")
	link := filepath.Join(dir, "source.txt")
	if err := os.WriteFile(target, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if _, err := prepareUploadSource(link); err == nil {
		t.Fatal("symlink upload source must be rejected")
	}
}

func TestValidateUploadSourcePathAcceptsRegularFile(t *testing.T) {
	local := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(local, []byte("content"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := validateUploadSourcePath(local); err != nil {
		t.Fatalf("regular upload source was rejected: %v", err)
	}
}
