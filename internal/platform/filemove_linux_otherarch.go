//go:build linux && !amd64 && !arm64 && !386

package platform

import (
	"errors"
	"fmt"
	"os"
)

// Official ByFTP Linux packages target amd64, arm64 and 386 and use kernel
// RENAME_NOREPLACE. Other Linux architectures retain safe exclusive-link
// semantics for regular staged files instead of failing to compile or using a
// check-then-rename overwrite window.
func RenameNoReplace(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 {
		return errors.New("sigurno no-replace premještanje podržava samo obične datoteke na ovoj Linux arhitekturi")
	}
	if err := os.Link(src, dst); err != nil {
		return err
	}
	if err := os.Remove(src); err != nil {
		return fmt.Errorf("odredište je sigurno kreirano bez prepisivanja, ali izvor nije uklonjen: %w", err)
	}
	return nil
}
