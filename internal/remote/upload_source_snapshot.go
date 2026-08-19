package remote

import (
	"bytes"
	"crypto/sha256"
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
	digest  [sha256.Size]byte
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

// copyUploadSnapshot copies one already-verified open source object into a
// private file and proves that a second full read of the same source handle has
// exactly the same SHA-256 digest. This detects ordinary concurrent in-place
// writes and prevents a torn local read from becoming a committed remote file.
func copyUploadSnapshot(source *os.File, destination string, original os.FileInfo) ([sha256.Size]byte, error) {
	var zero [sha256.Size]byte
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return zero, err
	}
	copyHash := sha256.New()
	copied, copyErr := io.Copy(io.MultiWriter(out, copyHash), source)
	if copyErr == nil {
		copyErr = out.Sync()
	}
	closeErr := out.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(destination)
		return zero, copyErr
	}
	afterCopy, err := source.Stat()
	if err != nil {
		_ = os.Remove(destination)
		return zero, err
	}
	if !sameUploadSourceInfo(original, afterCopy) || copied != original.Size() {
		_ = os.Remove(destination)
		return zero, errors.New("lokalni upload izvor se promijenio tijekom izrade sigurnog snapshota")
	}
	if _, err := source.Seek(0, io.SeekStart); err != nil {
		_ = os.Remove(destination)
		return zero, err
	}
	verifyHash := sha256.New()
	verifiedBytes, err := io.Copy(verifyHash, source)
	if err != nil {
		_ = os.Remove(destination)
		return zero, err
	}
	afterVerify, err := source.Stat()
	if err != nil {
		_ = os.Remove(destination)
		return zero, err
	}
	if !sameUploadSourceInfo(original, afterVerify) || verifiedBytes != original.Size() || !bytes.Equal(copyHash.Sum(nil), verifyHash.Sum(nil)) {
		_ = os.Remove(destination)
		return zero, errors.New("lokalni upload izvor nije ostao sadržajno stabilan tijekom izrade snapshota")
	}
	var digest [sha256.Size]byte
	copy(digest[:], copyHash.Sum(nil))
	return digest, nil
}

// prepareUploadSource binds an upload to one verified local filesystem object.
// The child network tool never reopens the user-controlled original pathname.
// Instead, ByFTP creates a byte-for-byte private snapshot from the verified open
// handle and validates its content before it can be used for a remote commit.
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
	digest, err := copyUploadSnapshot(source, snapshotPath, opened)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("nije moguće izraditi sigurni upload snapshot: %w", err)
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
	if validateUploadSourceInfo(snapshotPath, pathInfo) != nil || !sameUploadSourceInfo(pathInfo, handleInfo) || handleInfo.Size() != opened.Size() {
		_ = handle.Close()
		cleanup()
		return nil, errors.New("sigurni upload snapshot nije stabilna obična datoteka")
	}
	return &uploadSourceSnapshot{dir: tempDir, path: snapshotPath, handle: handle, initial: handleInfo, digest: digest}, nil
}

func (s *uploadSourceSnapshot) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Verify is the final local-source boundary. It confirms filesystem identity and
// byte content after curl/OpenSSH has finished reading the snapshot, then always
// removes the local snapshot before returning. A cleanup failure is an error, so
// the caller can delete the remote temp object instead of committing while a
// sensitive local copy remains unexpectedly behind.
func (s *uploadSourceSnapshot) Verify() (retErr error) {
	if s == nil {
		return errors.New("upload snapshot nije dostupan")
	}
	defer func() {
		if closeErr := s.Close(); closeErr != nil {
			cleanupErr := fmt.Errorf("nije moguće ukloniti lokalni upload snapshot: %w", closeErr)
			retErr = errors.Join(retErr, cleanupErr)
		}
	}()
	if s.handle == nil || s.path == "" {
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
	if _, err := s.handle.Seek(0, io.SeekStart); err != nil {
		return err
	}
	h := sha256.New()
	readBytes, err := io.Copy(h, s.handle)
	if err != nil {
		return err
	}
	afterHash, err := s.handle.Stat()
	if err != nil {
		return err
	}
	if !sameUploadSourceInfo(s.initial, afterHash) || readBytes != s.initial.Size() || !bytes.Equal(h.Sum(nil), s.digest[:]) {
		return errors.New("sigurni upload snapshot se sadržajno promijenio tijekom prijenosa")
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
	s.digest = [sha256.Size]byte{}
	return errors.Join(errs...)
}
