//go:build darwin

package platform

import (
	"errors"
	"fmt"
	"os"
)

// RenameNoReplace uses an exclusive hard-link creation for regular files on
// macOS. link(2) fails atomically when dst already exists, so a competing file
// can never be overwritten. ByFTP uses this primitive for staged local files.
// If unlinking src fails after the link succeeds, both names are left in place
// and the error is surfaced rather than risking deletion of unrelated data.
func RenameNoReplace(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("sigurno no-replace premještanje na macOS-u podržava samo obične datoteke")
	}
	if err := os.Link(src, dst); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("odredište je sigurno kreirano bez prepisivanja, ali izvor nije uklonjen: %w", err)
	}
	return nil
}
