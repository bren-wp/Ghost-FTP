package main

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

type installDirectoryGuard struct {
	root   string
	dir    string
	handle *os.File
}

func pinInstallDirectory(root, dir string) (*installDirectoryGuard, error) {
	if root == "" || dir == "" {
		return nil, errors.New("instalacijska mapa nije dostupna za sigurnosno vezivanje")
	}
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return nil, err
	}
	dirAbs, err := filepath.Abs(filepath.Clean(dir))
	if err != nil {
		return nil, err
	}
	if err := security.EnsureNoRedirectDirectory(rootAbs, dirAbs); err != nil {
		return nil, err
	}

	before, err := os.Lstat(dirAbs)
	if err != nil {
		return nil, err
	}
	if !before.IsDir() || before.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(dirAbs) {
		return nil, errors.New("instalacijska mapa mora biti obična lokalna mapa bez preusmjeravanja")
	}

	h, err := os.Open(dirAbs)
	if err != nil {
		return nil, err
	}
	opened, err := h.Stat()
	if err != nil {
		_ = h.Close()
		return nil, err
	}
	if !opened.IsDir() || !os.SameFile(before, opened) {
		_ = h.Close()
		return nil, errors.New("instalacijska mapa se promijenila tijekom sigurnog otvaranja")
	}

	guard := &installDirectoryGuard{root: rootAbs, dir: dirAbs, handle: h}
	if err := guard.verify(); err != nil {
		_ = h.Close()
		return nil, err
	}
	return guard, nil
}

func (g *installDirectoryGuard) verify() error {
	if g == nil || g.handle == nil || g.root == "" || g.dir == "" {
		return errors.New("identitet instalacijske mape nije dostupan")
	}
	if err := security.EnsureNoRedirectDirectory(g.root, g.dir); err != nil {
		return err
	}
	opened, err := g.handle.Stat()
	if err != nil {
		return err
	}
	current, err := os.Lstat(g.dir)
	if err != nil {
		return err
	}
	if !opened.IsDir() || !current.IsDir() || current.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(g.dir) {
		return errors.New("instalacijska mapa više nije sigurna lokalna mapa")
	}
	if !os.SameFile(opened, current) {
		return errors.New("instalacijska mapa je zamijenjena tijekom instalacije")
	}
	return nil
}

func (g *installDirectoryGuard) close() {
	if g != nil && g.handle != nil {
		_ = g.handle.Close()
		g.handle = nil
	}
}
