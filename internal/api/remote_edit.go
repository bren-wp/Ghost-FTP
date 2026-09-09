package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const MaxRemoteEditBytes int64 = 4 << 20

var (
	ErrRemoteEditConflict = errors.New("remote file changed since it was opened")
	ErrRemoteEditBinary   = errors.New("remote file is not UTF-8 text and cannot be edited safely")
	ErrRemoteEditTooLarge = errors.New("remote file is too large for the built-in editor")
)

type RemoteEditDocument struct {
	Path        string
	Text        string
	Revision    string
	Size        int64
	Permissions string
}

type remoteEditWorkspace struct {
	path string
	root *os.Root
	info os.FileInfo
}

func newRemoteEditWorkspace(base string) (*remoteEditWorkspace, error) {
	dir, err := os.MkdirTemp(base, ".GhostFTP-edit-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() { _ = os.Remove(dir) }
	if err := os.Chmod(dir, 0700); err != nil {
		cleanup()
		return nil, err
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || security.IsReparsePoint(dir) {
		cleanup()
		if err != nil {
			return nil, err
		}
		return nil, errors.New("remote edit workspace is not a private directory")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		cleanup()
		return nil, err
	}
	return &remoteEditWorkspace{path: dir, root: root, info: info}, nil
}

func (w *remoteEditWorkspace) cleanup() {
	if w == nil {
		return
	}
	if w.root != nil {
		for _, name := range []string{"content", "source"} {
			_ = w.root.Remove(name)
		}
		_ = w.root.Close()
		w.root = nil
	}
	if st, err := os.Lstat(w.path); err == nil && st.IsDir() && os.SameFile(w.info, st) && !security.IsReparsePoint(w.path) {
		_ = os.Remove(w.path)
	}
}

func (w *remoteEditWorkspace) pathFor(name string) string {
	return filepath.Join(w.path, name)
}

func (w *remoteEditWorkspace) writeSource(data []byte) (string, error) {
	f, err := w.root.OpenFile("source", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
	}
	closed := false
	ok := false
	defer func() {
		if !closed {
			_ = f.Close()
		}
		if !ok {
			_ = w.root.Remove("source")
		}
	}()
	if _, err := f.Write(data); err != nil {
		return "", err
	}
	if err := f.Sync(); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	closed = true
	st, err := w.root.Lstat("source")
	if err != nil || !st.Mode().IsRegular() || st.Mode()&os.ModeSymlink != 0 || st.Size() != int64(len(data)) || security.IsReparsePoint(w.pathFor("source")) {
		if err != nil {
			return "", err
		}
		return "", errors.New("remote edit staging file changed before upload")
	}
	ok = true
	return w.pathFor("source"), nil
}

func remoteEditRevision(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func validRemoteEditRevision(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func validateRemoteEditText(data []byte) error {
	if int64(len(data)) > MaxRemoteEditBytes {
		return ErrRemoteEditTooLarge
	}
	if !utf8.Valid(data) {
		return ErrRemoteEditBinary
	}
	for _, b := range data {
		if b == 0 {
			return ErrRemoteEditBinary
		}
	}
	return nil
}

func remoteEditPermissionMode(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) == 3 || len(raw) == 4 {
		for _, ch := range raw {
			if ch < '0' || ch > '7' {
				return ""
			}
		return raw
	}
	if len(raw) < 10 || len(raw) > 11 {
		return ""
	}
	raw = raw[:10]
	if !strings.ContainsRune("-bcdlps", rune(raw[0])) {
		return ""
	}
	digits := []byte{'0', '0', '0'}
	for group := 0; group < 3; group++ {
		base := 1 + group*3
		if !strings.ContainsRune("r-", rune(raw[base])) || !strings.ContainsRune("w-", rune(raw[base+1])) || !strings.ContainsRune("xstST-", rune(raw[base+2])) {
			return ""
		}
		value := 0
		if raw[base] == 'r' {
			value += 4
		}
		if raw[base+1] == 'w' {
			value += 2
		}
		if strings.ContainsRune("xst", rune(raw[base+2])) {
			value++
		}
		digits[group] = byte('0' + value)
	}
	special := 0
	if raw[3] == 's' || raw[3] == 'S' {
		special += 4
	}
	if raw[6] == 's' || raw[6] == 'S' {
		special += 2
	}
	if raw[9] == 't' || raw[9] == 'T' {
		special++
	}
	if special != 0 {
		return string([]byte{byte('0' + special), digits[0], digits[1], digits[2]})
	}
	return string(digits)
}

func editableRemoteItem(ctx context.Context, s remote.Session, remotePath string) (model.Item, error) {
	if err := security.ValidateRemoteFilePath(remotePath); err != nil {
		return model.Item{}, err
	}
	clean := path.Clean(remotePath)
	dir, name := path.Dir(clean), path.Base(clean)
	items, err := s.List(ctx, dir)
	if err != nil {
		return model.Item{}, err
	}
	for _, item := range items {
		if item.Name != name {
			continue
		}
		if item.IsDirectory || item.IsSymlink {
			return model.Item{}, errors.New("only regular remote files can be edited")
		}
		if item.Size > MaxRemoteEditBytes {
			return model.Item{}, ErrRemoteEditTooLarge
		}
		return item, nil
	}
	return model.Item{}, errors.New("remote file was not found in its parent directory")
}

func readRemoteEditFile(w *remoteEditWorkspace, name string) ([]byte, error) {
	before, err := w.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !before.Mode().IsRegular() || before.Mode()&os.ModeSymlink != 0 || before.Size() > MaxRemoteEditBytes || security.IsReparsePoint(w.pathFor(name)) {
		if before.Size() > MaxRemoteEditBytes {
			return nil, ErrRemoteEditTooLarge
		}
		return nil, errors.New("downloaded edit source is not a regular file")
	}
	f, err := w.root.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(before, opened) {
		if err != nil {
			return nil, err
		}
		return nil, errors.New("downloaded edit source changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxRemoteEditBytes+1))
	if err != nil {
		security.WipeBytes(data)
		return nil, err
	}
	if int64(len(data)) > MaxRemoteEditBytes {
		security.WipeBytes(data)
		return nil, ErrRemoteEditTooLarge
	}
	after, err := f.Stat()
	if err != nil || !os.SameFile(opened, after) || opened.Size() != after.Size() || int64(len(data)) != after.Size() || !opened.ModTime().Equal(after.ModTime()) {
		security.WipeBytes(data)
		if err != nil {
			return nil, err
		}
		return nil, errors.New("downloaded edit source changed while reading")
	}
	if err := validateRemoteEditText(data); err != nil {
		security.WipeBytes(data)
		return nil, err
	}
	return data, nil
}

func openRemoteEditWithSession(ctx context.Context, dataDir string, s remote.Session, remotePath string) (RemoteEditDocument, error) {
	item, err := editableRemoteItem(ctx, s, remotePath)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	workspace, err := newRemoteEditWorkspace(dataDir)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	defer workspace.cleanup()
	local := workspace.pathFor("content")
	if err := s.Download(ctx, remotePath, local, remote.TransferOptions{LocalRoot: workspace.path}); err != nil {
		return RemoteEditDocument{}, err
	}
	data, err := readRemoteEditFile(workspace, "content")
	if err != nil {
		return RemoteEditDocument{}, err
	}
	defer security.WipeBytes(data)
	return RemoteEditDocument{
		Path:        remotePath,
		Text:        string(data),
		Revision:    remoteEditRevision(data),
		Size:        int64(len(data)),
		Permissions: item.Permissions,
	}, nil
}

func saveRemoteEditWithSession(ctx context.Context, dataDir string, s remote.Session, remotePath, expectedRevision, text string) (RemoteEditDocument, error) {
	if !validRemoteEditRevision(expectedRevision) {
		return RemoteEditDocument{}, errors.New("remote edit revision is invalid")
	}
	content := []byte(text)
	defer security.WipeBytes(content)
	if err := validateRemoteEditText(content); err != nil {
		return RemoteEditDocument{}, err
	}

	current, err := openRemoteEditWithSession(ctx, dataDir, s, remotePath)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	if current.Revision != expectedRevision {
		return RemoteEditDocument{}, ErrRemoteEditConflict
	}

	clean := path.Clean(remotePath)
	dir, name := path.Dir(clean), path.Base(clean)
	mode := remoteEditPermissionMode(current.Permissions)
	if mode != "" {
		// Prove the server accepts the exact existing mode before changing content.
		// This makes permission preservation fail closed for servers that expose a
		// POSIX mode but cannot safely apply it through the active protocol.
		if err := s.Chmod(ctx, dir, name, mode); err != nil {
			return RemoteEditDocument{}, fmt.Errorf("remote file permissions cannot be preserved safely: %w", err)
		}
	}

	workspace, err := newRemoteEditWorkspace(dataDir)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	defer workspace.cleanup()
	source, err := workspace.writeSource(content)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	if err := s.Upload(ctx, source, remotePath, remote.TransferOptions{}); err != nil {
		return RemoteEditDocument{}, err
	}
	if mode != "" {
		if err := s.Chmod(ctx, dir, name, mode); err != nil {
			return RemoteEditDocument{}, fmt.Errorf("remote edit content was saved but original permissions could not be restored: %w", err)
		}
	}

	writtenRevision := remoteEditRevision(content)
	verified, err := openRemoteEditWithSession(ctx, dataDir, s, remotePath)
	if err != nil {
		return RemoteEditDocument{}, fmt.Errorf("remote edit was uploaded but read-back verification failed: %w", err)
	}
	if verified.Revision != writtenRevision {
		return RemoteEditDocument{}, errors.New("remote edit was uploaded but read-back content did not match")
	}
	return verified, nil
}

func (e *Engine) RemoteEditOpen(ctx context.Context, remotePath string) (RemoteEditDocument, error) {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	defer release()
	return openRemoteEditWithSession(opCtx, e.dataDir, s, remotePath)
}

func (e *Engine) RemoteEditSave(ctx context.Context, remotePath, expectedRevision, text string) (RemoteEditDocument, error) {
	s, opCtx, release, err := e.remote.Operation(ctx)
	if err != nil {
		return RemoteEditDocument{}, err
	}
	defer release()
	return saveRemoteEditWithSession(opCtx, e.dataDir, s, remotePath, expectedRevision, text)
}
