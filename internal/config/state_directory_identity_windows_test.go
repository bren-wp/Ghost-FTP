//go:build windows

package config

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func setWindowsDirectoryCreationTime(t *testing.T, path string, creation syscall.Filetime) {
	t.Helper()

	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(
		pathUTF16,
		syscall.FILE_WRITE_ATTRIBUTES,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS,
		0,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer syscall.CloseHandle(handle)

	if err := syscall.SetFileTime(handle, &creation, nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestWindowsStateDirectoryIdentitySurvivesCreationTimeCollision(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	before, err := stateDirectoryInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	beforeData, ok := before.Sys().(*syscall.Win32FileAttributeData)
	if !ok || beforeData == nil {
		t.Fatal("missing Win32 directory metadata")
	}

	original := filepath.Join(base, "state-original")
	if err := os.Rename(dir, original); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}

	// Make the replacement deliberately share the original creation timestamp.
	// The old implementation could then accept it because os.SameFile had not
	// resolved the original FileInfo's lazy Windows file ID before the rename.
	setWindowsDirectoryCreationTime(t, dir, beforeData.CreationTime)

	after, err := stateDirectoryInfo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if sameStateDirectoryIdentity(before, after) {
		t.Fatal("replacement directory with matching creation time reused the bound identity")
	}
}
