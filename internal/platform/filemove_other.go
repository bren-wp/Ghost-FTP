//go:build !windows && !linux && !darwin

package platform

import (
	"errors"
	"fmt"
	"os"
)

// RenameNoReplace is a conservative regular-file fallback for Unix-like
// platforms without a native exclusive-rename implementation in ByFTP.
// Creating the destination hard link is atomic and fails if dst already exists;
// unlike check-then-rename it can never silently overwrite a competing file.
func RenameNoReplace(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("sigurno no-replace premještanje podržava samo obične datoteke na ovoj platformi")
	}
	if err := os.Link(src, dst); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("odredište je sigurno kreirano bez prepisivanja, ali izvor nije uklonjen: %w", err)
	}
	return nil
}
