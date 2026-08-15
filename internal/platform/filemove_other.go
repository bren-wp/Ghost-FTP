//go:build !windows

package platform

import (
	"errors"
	"os"
)

// RenameNoReplace provides the same semantic contract for non-Windows test and
// development builds. The Windows release implementation is atomic at the OS
// move call; this fallback additionally checks the destination first.
func RenameNoReplace(src, dst string) error {
	if _, err := os.Lstat(dst); err == nil {
		return os.ErrExist
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(src, dst)
}
