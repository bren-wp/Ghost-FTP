package main

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

var openInstallerBackupSource = os.Open

type fileBackup struct {
	target          string
	backup          string
	original        os.FileInfo
	originalDigest  [sha256.Size]byte
	backupInfo      os.FileInfo
	activated       bool
	installed       os.FileInfo
	installedDigest [sha256.Size]byte
}

func ensureInstallDir(dir string) error {
	if dir == "" {
		return errors.New("instalacijska putanja nije zadana")
	}

	info, err := os.Lstat(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		info, err = os.Lstat(dir)
	}
	if err != nil {
		return err
	}

	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(dir) {
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

func sameStableInstallerFile(a, b os.FileInfo) bool {
	return a != nil && b != nil &&
		os.SameFile(a, b) &&
		a.Size() == b.Size() &&
		a.ModTime().Equal(b.ModTime())
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
	if err := safeInstallerRegular(path, opened); err != nil || !sameStableInstallerFile(before, opened) {
		return nil, zero, errors.New("instalacijska datoteka se promijenila tijekom sigurnog otvaranja")
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
	if !sameStableInstallerFile(opened, after) {
		return nil, zero, errors.New("instalacijska datoteka se promijenila tijekom čitanja")
	}

	current, err := os.Lstat(path)
	if err != nil {
		return nil, zero, err
	}
	if err := safeInstallerRegular(path, current); err != nil || !sameStableInstallerFile(after, current) {
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

	src, err := openInstallerBackupSource(target)
	if err != nil {
		return fileBackup{}, err
	}
	defer src.Close()

	opened, err := src.Stat()
	if err != nil {
		return fileBackup{}, err
	}
	if err := safeInstallerRegular(target, opened); err != nil || !sameStableInstallerFile(info, opened) {
		return fileBackup{}, errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnog otvaranja")
	}

	dst, err := os.CreateTemp(filepath.Dir(target), ".GhostFTP-rollback-*.bak")
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

	// Read the already-open source again. This detects content changes that do
	// not necessarily alter size or modification time during the first copy.
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
			} else {
				var verifyDigest [sha256.Size]byte
				copy(verifyDigest[:], verifyHash.Sum(nil))
				if digest != verifyDigest {
					copyErr = errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnosne kopije")
				}
			}
		}
	}

	after, statErr := src.Stat()
	if statErr == nil && !sameStableInstallerFile(opened, after) {
		statErr = errors.New("postojeća instalacijska datoteka se promijenila tijekom sigurnosne kopije")
	}

	current, currentErr := os.Lstat(target)
	if currentErr == nil {
		if err := safeInstallerRegular(target, current); err != nil || !sameStableInstallerFile(after, current) {
			currentErr = errors.New("instalacijska putanja se promijenila tijekom sigurnosne kopije")
		}
	}

	syncErr := dst.Sync()
	closeErr := dst.Close()
	if err := errors.Join(copyErr, statErr, currentErr, syncErr, closeErr); err != nil {
		_ = os.Remove(backup)
		return fileBackup{}, err
	}

	// Verify the persisted backup itself before it becomes part of the
	// transaction. This prevents a corrupted .bak from being trusted later.
	backupInfo, backupDigest, err := digestStableInstallerFile(backup)
	if err != nil {
		_ = os.Remove(backup)
		return fileBackup{}, fmt.Errorf("sigurnosnu kopiju nije moguće potvrditi: %w", err)
	}
	if backupDigest != digest || backupInfo.Size() != opened.Size() {
		_ = os.Remove(backup)
		return fileBackup{}, errors.New("sigurnosna kopija ne odgovara postojećoj instalacijskoj datoteci")
	}

	return fileBackup{
		target:         target,
		backup:         backup,
		original:       after,
		originalDigest: digest,
		backupInfo:     backupInfo,
	}, nil
}

func (b *fileBackup) verifyBeforeInstall() error {
	if b == nil || b.target == "" {
		return errors.New("instalacijska transakcija nije ispravna")
	}

	if !b.existed() {
		_, err := os.Lstat(b.target)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		return errors.New("ciljna instalacijska datoteka pojavila se tijekom instalacije")
	}

	current, digest, err := digestStableInstallerFile(b.target)
	if err != nil {
		return err
	}
	if !sameStableInstallerFile(b.original, current) || digest != b.originalDigest {
		return errors.New("postojeća instalacijska datoteka promijenjena je nakon izrade sigurnosne kopije")
	}

	return nil
}

func (b *fileBackup) recordActivated(expected [sha256.Size]byte) error {
	if b == nil || b.target == "" {
		return errors.New("instalacijska transakcija nije ispravna")
	}

	// Mark activation before verification so the transaction knows that the
	// target path was changed even if post-install verification fails.
	b.activated = true

	info, digest, err := digestStableInstallerFile(b.target)
	if err != nil {
		return fmt.Errorf("instalirana datoteka nije mogla biti potvrđena: %w", err)
	}

	// Store what was actually observed. If the payload digest does not match,
	// rollback can still prove that this exact installed object has not changed.
	b.installed = info
	b.installedDigest = digest

	if digest != expected {
		return errors.New("instalirana datoteka ne odgovara provjerenom instalacijskom paketu")
	}
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

func (b fileBackup) verifyBackupForRollback() error {
	if b.backup == "" {
		return errors.New("sigurnosna kopija nije dostupna")
	}
	if b.backupInfo == nil {
		return errors.New("identitet sigurnosne kopije nije dostupan")
	}

	current, digest, err := digestStableInstallerFile(b.backup)
	if err != nil {
		return fmt.Errorf("sigurnosnu kopiju nije moguće potvrditi: %w", err)
	}
	if !sameStableInstallerFile(b.backupInfo, current) || digest != b.originalDigest {
		return errors.New("sigurnosna kopija promijenjena je prije rollbacka")
	}
	return nil
}

func (b fileBackup) stageRollbackBackup() (string, error) {
	if err := b.verifyBackupForRollback(); err != nil {
		return "", err
	}

	src, err := os.Open(b.backup)
	if err != nil {
		return "", err
	}
	defer src.Close()

	dst, err := os.CreateTemp(filepath.Dir(b.target), ".GhostFTP-restore-*.tmp")
	if err != nil {
		return "", err
	}
	tmp := dst.Name()
	cleanup := func() {
		_ = dst.Close()
		_ = os.Remove(tmp)
	}

	mode := os.FileMode(0700)
	if b.original != nil && b.original.Mode().Perm() != 0 {
		mode = b.original.Mode().Perm()
	}
	if err := dst.Chmod(mode); err != nil {
		cleanup()
		return "", err
	}

	copied, copyErr := io.Copy(dst, src)
	if copyErr == nil && b.original != nil && copied != b.original.Size() {
		copyErr = errors.New("sigurnosnu kopiju nije moguće potpuno pripremiti za rollback")
	}
	syncErr := dst.Sync()
	closeErr := dst.Close()
	if err := errors.Join(copyErr, syncErr, closeErr); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}

	_, digest, err := digestStableInstallerFile(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("pripremljenu rollback datoteku nije moguće potvrditi: %w", err)
	}
	if digest != b.originalDigest {
		_ = os.Remove(tmp)
		return "", errors.New("pripremljena rollback datoteka ne odgovara sigurnosnoj kopiji")
	}

	return tmp, nil
}

func (b fileBackup) rollback() error {
	if b.target == "" || !b.activated {
		return nil
	}
	if err := b.verifyInstalledForRollback(); err != nil {
		return err
	}

	// Fresh install: only remove the exact file object previously observed as
	// installed. verifyInstalledForRollback above protects against replacement.
	if b.backup == "" {
		err := os.Remove(b.target)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	// Never consume the only rollback backup directly. Restore from a freshly
	// verified staging copy so the original .bak survives a failed replacement.
	restoreTmp, err := b.stageRollbackBackup()
	if err != nil {
		return err
	}
	defer os.Remove(restoreTmp)

	if err := platform.ReplaceFile(restoreTmp, b.target); err != nil {
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

func (b fileBackup) existed() bool {
	return b.backup != ""
}

func (b fileBackup) cleanup() {
	if b.backup != "" {
		_ = os.Remove(b.backup)
	}
}
