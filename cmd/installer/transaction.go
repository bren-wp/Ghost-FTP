package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

type fileBackup struct {
	target string
	backup string
}

func ensureInstallDir(dir string) error {
	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dir, 0755)
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(dir) || !info.IsDir() {
		return errors.New("instalacijska putanja nije sigurna mapa")
	}
	return nil
}

func backupExisting(target string) (fileBackup, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return fileBackup{target: target}, nil
	}
	if err != nil {
		return fileBackup{}, err
	}
	if !info.Mode().IsRegular() || security.IsReparsePoint(target) {
		return fileBackup{}, errors.New("postojeća instalacijska datoteka nije regularna datoteka")
	}
	src, err := os.Open(target)
	if err != nil {
		return fileBackup{}, err
	}
	defer src.Close()
	dst, err := os.CreateTemp(filepath.Dir(target), ".byftp-rollback-*.bak")
	if err != nil {
		return fileBackup{}, err
	}
	backup := dst.Name()
	if err := dst.Chmod(0600); err != nil {
		_ = dst.Close()
		_ = os.Remove(backup)
		return fileBackup{}, err
	}
	_, copyErr := io.Copy(dst, src)
	syncErr := dst.Sync()
	closeErr := dst.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(backup)
		if copyErr != nil {
			return fileBackup{}, copyErr
		}
		if syncErr != nil {
			return fileBackup{}, syncErr
		}
		return fileBackup{}, closeErr
	}
	return fileBackup{target: target, backup: backup}, nil
}

func (b fileBackup) rollback() error {
	if b.target == "" {
		return nil
	}
	if b.backup == "" {
		err := os.Remove(b.target)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return platform.ReplaceFile(b.backup, b.target)
}

func (b fileBackup) existed() bool { return b.backup != "" }

func (b fileBackup) cleanup() {
	if b.backup != "" {
		_ = os.Remove(b.backup)
	}
}
