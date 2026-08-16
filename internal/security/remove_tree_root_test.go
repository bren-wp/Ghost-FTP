package security

import (
	"path/filepath"
	"testing"
)

func TestRemoveTreeNoFollowRejectsCurrentFilesystemRoot(t *testing.T) {
	root := filepath.VolumeName(string(filepath.Separator)) + string(filepath.Separator)
	if !isFilesystemRoot(root) {
		t.Fatalf("filesystem root %q was not recognized", root)
	}
	if err := RemoveTreeNoFollow(root); err == nil {
		t.Fatal("filesystem root deletion was not rejected")
	}
}
