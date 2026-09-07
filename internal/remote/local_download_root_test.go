package remote

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalDownloadTargetActivatesInsideRoot(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "nested", "file.txt")
	target, err := prepareLocalDownloadTarget(local, root, false)
	if err != nil {
		t.Fatalf("prepareLocalDownloadTarget: %v", err)
	}
	defer target.Close()
	if err := os.WriteFile(target.partPath, []byte("downloaded"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.activate(false, false); err != nil {
		t.Fatalf("activate: %v", err)
	}
	got, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "downloaded" {
		t.Fatalf("downloaded content=%q", got)
	}
}

func TestLocalDownloadTargetRejectsUnchangedSentinel(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "file.txt")
	target, err := prepareLocalDownloadTarget(local, root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := target.validatePart(); err == nil {
		t.Fatal("unchanged prepared staging file was accepted as transferred data")
	}
	target.cleanupPart()
}

func TestLocalDownloadTargetRejectsLateNestedRedirect(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	localDir := filepath.Join(root, "nested")
	local := filepath.Join(localDir, "file.txt")
	target, err := prepareLocalDownloadTarget(local, root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := os.WriteFile(target.partPath, []byte("safe"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(localDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, localDir); err != nil {
		t.Skipf("symlink nije dostupan: %v", err)
	}
	if err := target.activate(false, false); err == nil {
		t.Fatal("late nested symlink redirect was accepted")
	}
	if _, err := os.Lstat(filepath.Join(outside, "file.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("download escaped root through late redirect: %v", err)
	}
}

func TestLocalDownloadTargetRechecksSkipExistingAtCommit(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "file.txt")
	target, err := prepareLocalDownloadTarget(local, root, true)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := os.WriteFile(target.partPath, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("raced"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.activate(false, true); !errors.Is(err, ErrSkipped) {
		t.Fatalf("late existing target error=%v want ErrSkipped", err)
	}
	got, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "raced" {
		t.Fatalf("late existing target was overwritten: %q", got)
	}
}

func TestLocalDownloadTargetKeepsVerifiedBackup(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "nested", "file.txt")
	if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(local, []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	target, err := prepareLocalDownloadTarget(local, root, false)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	if err := os.WriteFile(target.partPath, []byte("new"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := target.activate(true, false); err != nil {
		t.Fatalf("activate with backup: %v", err)
	}
	got, err := os.ReadFile(local)
	if err != nil || string(got) != "new" {
		t.Fatalf("active file=%q err=%v", got, err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(local), "file.txt.GhostFTP-backup-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("backup matches=%v want one", matches)
	}
	backup, err := os.ReadFile(matches[0])
	if err != nil || string(backup) != "old" {
		t.Fatalf("backup=%q err=%v", backup, err)
	}
}

func TestLocalPathRelativeToRootRejectsEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	if _, err := localPathRelativeToRoot(root, outside); err == nil {
		t.Fatal("path outside root was accepted")
	}
}
