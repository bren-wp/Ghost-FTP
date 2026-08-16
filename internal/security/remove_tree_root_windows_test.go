//go:build windows

package security

import "testing"

func TestIsFilesystemRootRejectsWindowsVolumeRoots(t *testing.T) {
	for _, root := range []string{`C:\`, `D:\`, `\\server\share\`} {
		if !isFilesystemRoot(root) {
			t.Fatalf("Windows/UNC root %q was not recognized", root)
		}
	}
}
