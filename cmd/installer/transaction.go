package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

var openInstallerBackupSource = os.Open

type fileBackup struct {
	target          string
	backup          string
	original        os.FileInfo
	originalDigest  [sha256.Size]byte
	activated       bool
	installed       os.FileInfo
	installedDigest [sha256.Size]byte
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

func safeInstallerRegular(path string, info os.FileInfo) error {
	if info == nil || !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(path) {
		return errors.New("instalacijska datoteka nije regularna datoteka bez preusmjeravanja")
	}
	return nil
}

func digestStableInstallerFile(path string) (os.FileInfo, [sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	before, err := os.Lstat(path)
	if err != nil {
		return nil, zero, err
	}
	if err := safeInstallerRegular(path, before); err != nil {
		return nil, zero, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, zero, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil {
		return nil, zero, err
	}
	if err := safeInstallerRegular(path, opened); err != nil || !os.SameFile(before, opened) {
		return nil, zero, errors.New("instalacijska datoteka se promijenila tijekom sigurnog otvaranja")
	}
	if before.Size() != opened.Size() || !before.ModTime().Equal(opened.ModTime()) {
		return nil, zero, errors.New("instalacijska datoteka se promijenila prije čitanja")
	}

	h := sha256.New()
	read, err := io.Copy(h, f)
	if err != nil {
		return nil, zero, err
	}
	if read != opened.Size() {
		return nil, zero, errors.New("instalacijsku datoteku nije moguće stabilno pročitati")
	}
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))

	after, err := f.Stat()
	if err != nil {
		return nil, zero, err
	}
	if !os.SameFile(opened, after) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		return nil, zero, errors.New("instalacijska datoteka se promijenila tijekom čitanja")
	}
	current, err := os.Lstat(path)
	if err != nil {
		return nil, zero, err
	}
	if err := safeInstallerRegular(path, current); err != nil || !os.SameFile(after, current) || after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) {
		return nil, zero, errors.New("instalacijska putanja se promijenila tijekom provjere")
	}
	return after, digest, nil
}

func backupExisting(target string) (fileBackup, error) {
	info, err := os.Lstat(target)
	if errors.Is(err, os.ErrNotExist) {
		return fileBackup{target: target}, nil
	}
	if err != nil {
		return fileBackup{}, err
	}
	if err := safeInstallerRegular(target, info); err != nil {
		return fileBackup{}, errors.New("postojeća instalacijska datoteka nije regularna datoteka")
	}

	// Open only after Lstat, then prove that the opened handle still refers to
	// exactly the filesystem object that was validated above. Tests can replace
	// this opener to deterministically exercise the Lstat->Open race window.
	src, err := openInstallerBackupSource(target)
	if err != nil {
		return fileBackup{}, err
	}
	defer src.Close()
	opened, err := src.Stat()
	if err != nil {
		return fileBackup{}, err
	}
	if err := safeInstallerRegular(target, opened); err != nil || !os.SameFile(info, opened) || info.Size() != opened.Size() || !info.ModTime().Equal(opened.ModTime()) {
		return fileBackup{}, errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnog otvaranja")
	}

	dst, err := os.CreateTemp(filepath.Dir(target), ".byftp-rollback-*.bak")
	if err != nil {
		return fileBackup{}, err
	}
	backup := dst.Name()
	cleanup := func() {
		_ = dst.Close()
		_ = os.Remove(backup)
	}
	if err := dst.Chmod(0600); err != nil {
		cleanup()
		return fileBackup{}, err
	}

	h := sha256.New()
	copied, copyErr := io.Copy(io.MultiWriter(dst, h), src)
	if copyErr == nil && copied != opened.Size() {
		copyErr = errors.New("postojeću instalacijsku datoteku nije moguće stabilno kopirati")
	}
	var digest [sha256.Size]byte
	copy(digest[:], h.Sum(nil))

	// A second full read through the same already-open handle catches in-place
	// mutation during backup even when size/mtime are restored to old values.
	var verifyDigest [sha256.Size]byte
	if copyErr == nil {
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			copyErr = err
		} else {
			verifyHash := sha256.New()
			verified, err := io.Copy(verifyHash, src)
			if err != nil {
				copyErr = err
			} else if verified != copied {
				copyErr = errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnosne kopije")
			}
			copy(verifyDigest[:], verifyHash.Sum(nil))
			if copyErr == nil && digest != verifyDigest {
				copyErr = errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnosne kopije")
			}
		}
	}

	after, statErr := src.Stat()
	if statErr == nil && (!os.SameFile(opened, after) || opened.Size() != after.Size() || !opened.ModTime().Equal(after.ModTime())) {
		statErr = errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnosne kopije")
	}
	current, currentErr := os.Lstat(target)
	if currentErr == nil {
		if err := safeInstallerRegular(target, current); err != nil || !os.SameFile(after, current) || after.Size() != current.Size() || !after.ModTime().Equal(current.ModTime()) {
			currentErr = errors.New("instalacijska putanja se promijenila tijekom sigurnosne kopije")
		}
	}

	syncErr := dst.Sync()
	closeErr := dst.Close()
	if copyErr != nil || statErr != nil || currentErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(backup)
		for _, candidate := range []error{copyErr, statErr, currentErr, syncErr, closeErr} {
			if candidate != nil {
				return fileBackup{}, candidate
			}
		}
	}
	return fileBackup{target: target, backup: backup, original: after, originalDigest: digest}, nil
}

func (b *fileBackup) verifyBeforeInstall() error {
	if b == nil || b.target == "" {
		return errors.New("instalacijska transakcija nije ispravna")
	}
	if !b.existed() {
		if _, err := os.Lstat(b.target); errors.Is(err, os.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
		return errors.New("ciljna instalacijska datoteka pojavila se tijekom instalacije")
	}
	current, digest, err := digestStableInstallerFile(b.target)
	if err != nil {
		return err
	}
	if b.original == nil || !os.SameFile(b.original, current) || b.original.Size() != current.Size() || !b.original.ModTime().Equal(current.ModTime()) || digest != b.originalDigest {
		return errors.New("postojeća instalacijska datoteka promijenjena je nakon izrade sigurnosne kopije")
	}
	return nil
}

func (b *fileBackup) recordActivated(expected [sha256.Size]byte) error {
	if b == nil {
		return errors.New("instalacijska transakcija nije ispravna")
	}
	// Set activated before verification. If verification cannot prove ownership,
	// rollback will fail closed instead of deleting or replacing an unknown path.
	b.activated = true
	b.installedDigest = expected
	info, digest, err := digestStableInstallerFile(b.target)
	if err != nil {
		return fmt.Errorf("instalirana datoteka nije mogla biti potvrđena: %w", err)
	}
	if digest != expected {
		return errors.New("instalirana datoteka ne odgovara provjerenom instalacijskom paketu")
	}
	b.installed = info
	return nil
}

func (b fileBackup) verifyInstalledForRollback() error {
	if !b.activated {
		return nil
	}
	if b.installed == nil {
		return errors.New("nije moguće dokazati vlasništvo nad aktiviranom instalacijskom datotekom")
	}
	current, digest, err := digestStableInstallerFile(b.target)
	if err != nil {
		return err
	}
	if !os.SameFile(b.installed, current) || digest != b.installedDigest {
		return errors.New("instalirana datoteka promijenjena je prije rollbacka")
	}
	return nil
}

func (b fileBackup) rollback() error {
	if b.target == "" || !b.activated {
		return nil
	}
	if err := b.verifyInstalledForRollback(); err != nil {
		return err
	}
	if b.backup == "" {
		err := os.Remove(b.target)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if err := platform.ReplaceFile(b.backup, b.target); err != nil {
		return err
	}
	_, digest, err := digestStableInstallerFile(b.target)
	if err != nil {
		return fmt.Errorf("vraćena instalacijska datoteka nije mogla biti potvrđena: %w", err)
	}
	if digest != b.originalDigest {
		return errors.New("vraćena instalacijska datoteka ne odgovara sigurnosnoj kopiji")
	}
	return nil
}

func (b fileBackup) existed() bool { return b.backup != "" }

func (b fileBackup) cleanup() {
	if b.backup != "" {
		_ = os.Remove(b.backup)
	}
}
