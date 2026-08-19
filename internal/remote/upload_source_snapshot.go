package remote

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"brendigo.com/byftp/internal/security"
)

type uploadSourceSnapshot struct {
	dir     string
	path    string
	handle  *os.File
	initial os.FileInfo
}

func sameUploadSourceInfo(before, after os.FileInfo) bool {
	if before == nil || after == nil || !before.Mode().IsRegular() || !after.Mode().IsRegular() {
		return false
	}
	return os.SameFile(before, after) && before.Size() == after.Size() && before.ModTime().Equal(after.ModTime())
}

func validateUploadSourceInfo(local string, st os.FileInfo) error {
	if st == nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(local) {
		return errors.New("upload izvor mora biti obična lokalna datoteka bez preusmjeravanja")
	}
	return nil
}

func validateUploadSourcePath(local string) error {
	st, err := os.Lstat(local)
	if err != nil {
		return err
	}
	return validateUploadSourceInfo(local, st)
}

func copyUploadSnapshot(source *os.File, destination string, original os.FileInfo) error {
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	copied, copyErr := io.Copy(out, source)
	if copyErr == nil {
		copyErr = out.Sync()
	}
	closeErr := out.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(destination)
		return copyErr
	}
	after, err := source.Stat()
	if err != nil {
		_ = os.Remove(destination)
		return err
	}
	if !sameUploadSourceInfo(original, after) || copied != original.Size() {
		_ = os.Remove(destination)
		return errors.New("lokalni upload izvor se promijenio tijekom izrade sigurnog snapshota")
	}
	return nil
}

// prepareUploadSource binds an upload to one verified local filesystem object.
// It prefers a zero-copy hard-link inside a private temporary directory. When
// the system temp directory is on another volume or hard links are unavailable,
// it falls back to a byte copy read from the already verified open file handle.
func prepareUploadSource(local string) (*uploadSourceSnapshot, error) {
	before, err := os.Lstat(local)
	if err != nil {
		return nil, err
	}
	if err := validateUploadSourceInfo(local, before); err != nil {
		return nil, err
	}

	source, err := os.Open(local)
	if err != nil {
		return nil, err
	}
	defer source.Close()
	opened, err := source.Stat()
	if err != nil {
		return nil, err
	}
	if !sameUploadSourceInfo(before, opened) {
		return nil, errors.New("lokalni upload izvor se promijenio tijekom sigurnog otvaranja")
	}

	tempDir, err := os.MkdirTemp("", "byftp-upload-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = security.RemoveTreeNoFollow(tempDir) }
	if err := os.Chmod(tempDir, 0700); err != nil {
		cleanup()
		return nil, err
	}
	snapshotPath := filepath.Join(tempDir, "source")

	if linkErr := os.Link(local, snapshotPath); linkErr == nil {
		linked, statErr := os.Lstat(snapshotPath)
		if statErr != nil || validateUploadSourceInfo(snapshotPath, linked) != nil || !sameUploadSourceInfo(opened, linked) {
			cleanup()
			return nil, errors.New("lokalni upload izvor se promijenio tijekom izrade hard-link snapshota")
		}
	} else {
		// A link failure is a normal cross-volume/unsupported-filesystem case only
		// while the original pathname still names the verified object. If it was
		// swapped, fail closed instead of silently copying a stale handle.
		pathNow, statErr := os.Lstat(local)
		if statErr != nil || validateUploadSourceInfo(local, pathNow) != nil || !sameUploadSourceInfo(opened, pathNow) {
			cleanup()
			return nil, errors.New("lokalni upload izvor se promijenio prije izrade sigurnog snapshota")
		}
		if err := copyUploadSnapshot(source, snapshotPath, opened); err != nil {
			cleanup()
			return nil, fmt.Errorf("nije moguće izraditi sigurni upload snapshot: %w", err)
		}
	}

	handle, err := os.Open(snapshotPath)
	if err != nil {
		cleanup()
		return nil, err
	}
	pathInfo, err := os.Lstat(snapshotPath)
	if err != nil {
		_ = handle.Close()
		cleanup()
		return nil, err
	}
	handleInfo, err := handle.Stat()
	if err != nil {
		_ = handle.Close()
		cleanup()
		return nil, err
	}
	if validateUploadSourceInfo(snapshotPath, pathInfo) != nil || !sameUploadSourceInfo(pathInfo, handleInfo) {
		_ = handle.Close()
		cleanup()
		return nil, errors.New("sigurni upload snapshot nije stabilna obična datoteka")
	}
	return &uploadSourceSnapshot{dir: tempDir, path: snapshotPath, handle: handle, initial: handleInfo}, nil
}

func (s *uploadSourceSnapshot) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Verify confirms that the pathname handed to curl/OpenSSH still resolves to
// the exact open snapshot and that its ordinary size/mtime state did not change
// while the external process was reading it.
func (s *uploadSourceSnapshot) Verify() error {
	if s == nil || s.handle == nil || s.path == "" {
		return errors.New("upload snapshot nije dostupan")
	}
	pathInfo, err := os.Lstat(s.path)
	if err != nil {
		return err
	}
	if err := validateUploadSourceInfo(s.path, pathInfo); err != nil {
		return err
	}
	handleInfo, err := s.handle.Stat()
	if err != nil {
		return err
	}
	if !sameUploadSourceInfo(pathInfo, handleInfo) || !sameUploadSourceInfo(s.initial, handleInfo) {
		return errors.New("sigurni upload snapshot se promijenio tijekom prijenosa")
	}
	return nil
}

func (s *uploadSourceSnapshot) Close() error {
	if s == nil {
		return nil
	}
	var errs []error
	if s.handle != nil {
		if err := s.handle.Close(); err != nil {
			errs = append(errs, err)
		}
		s.handle = nil
	}
	if s.dir != "" {
		if err := security.RemoveTreeNoFollow(s.dir); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	s.path = ""
	s.dir = ""
	s.initial = nil
	return errors.Join(errs...)
}
