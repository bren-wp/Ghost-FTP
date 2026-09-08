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

type removeTreeHooks struct {
	beforeDescend func()
}

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

func isFilesystemRoot(target string) bool {
	cleaned := filepath.Clean(target)
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return false
	}
	volumeRoot := filepath.VolumeName(abs) + string(filepath.Separator)
	return filepath.Clean(abs) == filepath.Clean(volumeRoot)
}

func sameRegularObject(before, after os.FileInfo) bool {
	if before == nil || after == nil {
		return false
	}
	return before.Mode().Type() == after.Mode().Type() && os.SameFile(before, after)
}

// openStableRootDirectory opens name relative to an already trusted parent root
// and proves that the resulting directory handle still refers to the object
// inspected by Lstat. All recursive mutation then proceeds through this held
// root instead of rebuilding child pathnames from a mutable directory name.
func openStableRootDirectory(parent *os.Root, name string, before os.FileInfo) (*os.Root, []os.DirEntry, error) {
	child, err := parent.OpenRoot(name)
	if err != nil {
		return nil, nil, err
	}
	opened, err := child.Stat(".")
	if err != nil {
		child.Close()
		return nil, nil, err
	}
	if !opened.IsDir() || !sameRegularObject(before, opened) {
		child.Close()
		return nil, nil, errors.New("lokalna mapa se promijenila tijekom sigurnog otvaranja")
	}

	dir, err := child.Open(".")
	if err != nil {
		child.Close()
		return nil, nil, err
	}
	entries, readErr := dir.ReadDir(-1)
	closeErr := dir.Close()
	if readErr != nil {
		child.Close()
		return nil, nil, readErr
	}
	if closeErr != nil {
		child.Close()
		return nil, nil, closeErr
	}
	return child, entries, nil
}

func RemoveTreeNoFollow(root string) error {
	return removeTreeNoFollowWithHooks(root, nil)
}

func removeTreeNoFollowWithHooks(root string, hooks *removeTreeHooks) error {
	root = filepath.Clean(root)
	if root == "." || root == "" || isFilesystemRoot(root) {
		return errors.New("nije dopušteno brisanje korijenske lokalne mape")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	parentPath := filepath.Dir(abs)
	name := filepath.Base(abs)
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		return err
	}
	defer parent.Close()
	return removeRootEntry(parent, name, abs, 0, &removeTreeGuard{}, hooks)
}

func removeRootEntry(parent *os.Root, name, displayPath string, depth int, guard *removeTreeGuard, hooks *removeTreeHooks) error {
	if err := guard.step(depth); err != nil {
		return err
	}
	before, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if before.Mode()&os.ModeSymlink != 0 || isReparsePoint(displayPath) || !before.IsDir() {
		return parent.Remove(name)
	}

	child, entries, err := openStableRootDirectory(parent, name, before)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if hooks != nil && hooks.beforeDescend != nil {
			hooks.beforeDescend()
		}
		if err := removeRootEntry(child, entry.Name(), filepath.Join(displayPath, entry.Name()), depth+1, guard, hooks); err != nil {
			child.Close()
			return err
		}
	}
	if err := child.Close(); err != nil {
		return err
	}

	current, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if current.Mode()&os.ModeSymlink != 0 || isReparsePoint(displayPath) || !sameRegularObject(before, current) {
		return errors.New("lokalna mapa je zamijenjena tijekom rekurzivnog brisanja")
	}
	return parent.Remove(name)
}
