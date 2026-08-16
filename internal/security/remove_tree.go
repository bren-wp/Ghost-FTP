package security

import (
	"errors"
	"os"
	"path/filepath"
)

const (
	maxRemoveTreeDepth = 128
	maxRemoveTreeItems = 200000
)

type removeTreeGuard struct{ items int }

// IsReparsePoint reports whether path is a Windows reparse/junction-like entry.
// On non-Windows builds it always returns false.
func IsReparsePoint(path string) bool { return isReparsePoint(path) }

func (g *removeTreeGuard) step(depth int) error {
	if depth > maxRemoveTreeDepth {
		return errors.New("struktura lokalne mape je preduboka za sigurno brisanje")
	}
	g.items++
	if g.items > maxRemoveTreeItems {
		return errors.New("lokalna mapa sadrži previše stavki za jedno brisanje")
	}
	return nil
}

// RemoveTreeNoFollow recursively removes a local tree without traversing
// symbolic links or Windows reparse/junction points. Link-like entries are
// removed as entries only; their targets are never visited.
func isFilesystemRoot(target string) bool {
	cleaned := filepath.Clean(target)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return false
	}
	volumeRoot := filepath.VolumeName(abs) + string(filepath.Separator)
	return filepath.Clean(abs) == filepath.Clean(volumeRoot)
}

func RemoveTreeNoFollow(root string) error {
	root = filepath.Clean(root)
	if root == "." || root == "" || isFilesystemRoot(root) {
		return errors.New("nije dopušteno brisanje korijenske lokalne mape")
	}
	return removeTreeNoFollow(root, 0, &removeTreeGuard{})
}

func removeTreeNoFollow(target string, depth int, guard *removeTreeGuard) error {
	if err := guard.step(depth); err != nil {
		return err
	}
	st, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.Mode()&os.ModeSymlink != 0 || isReparsePoint(target) || !st.IsDir() {
		return os.Remove(target)
	}
	entries, err := os.ReadDir(target)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := removeTreeNoFollow(filepath.Join(target, entry.Name()), depth+1, guard); err != nil {
			return err
		}
	}
	return os.Remove(target)
}
