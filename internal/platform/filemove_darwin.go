//go:build darwin

package platform

import "os"

// RenameNoReplace uses an atomic hard-link creation to reserve the destination
// without replacing an existing entry, then removes the source name. The
// staged source and final destination are required to reside on the same local
// filesystem by the transfer activation path.
func RenameNoReplace(src, dst string) error {
	if err := os.Link(src, dst); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return nil
}
