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

// readStableDirectory opens the already inspected directory and verifies that
// the handle still refers to the same filesystem object before reading entries.
// This prevents a path swap to a symlink/junction from turning ReadDir into an
// unintended traversal of the replacement target.
func readStableDirectory(target string, before os.FileInfo) ([]os.DirEntry, error) {
	f, err := os.Open(target)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !opened.IsDir() || !sameRegularObject(before, opened) {
		return nil, errors.New("lokalna mapa se promijenila tijekom sigurnog otvaranja")
	}
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	afterRead, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if !sameRegularObject(opened, afterRead) {
		return nil, errors.New("lokalna mapa se promijenila tijekom čitanja")
	}
	pathNow, err := os.Lstat(target)
	if err != nil {
		return nil, err
	}
	if pathNow.Mode()&os.ModeSymlink != 0 || isReparsePoint(target) || !sameRegularObject(opened, pathNow) {
		return nil, errors.New("lokalna mapa je zamijenjena tijekom brisanja")
	}
	return entries, nil
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
	entries, err := readStableDirectory(target, st)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		parentNow, err := os.Lstat(target)
		if err != nil {
			return err
		}
		if parentNow.Mode()&os.ModeSymlink != 0 || isReparsePoint(target) || !sameRegularObject(st, parentNow) {
			return errors.New("lokalna mapa je zamijenjena tijekom rekurzivnog brisanja")
		}
		if err := removeTreeNoFollow(filepath.Join(target, entry.Name()), depth+1, guard); err != nil {
			return err
		}
	}
	final, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if final.Mode()&os.ModeSymlink != 0 || isReparsePoint(target) || !sameRegularObject(st, final) {
		return errors.New("lokalna mapa je zamijenjena prije završnog brisanja")
	}
	return os.Remove(target)
}
