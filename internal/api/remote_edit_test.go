package api

import (
	"context"
	"errors"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
)

type remoteEditTestSession struct {
	remotePath   string
	content      []byte
	permissions  string
	sizeOverride int64
	isDirectory  bool
	isSymlink    bool
	downloads    int
	uploads      int
	chmodModes   []string
	chmodErr     error
}

func (*remoteEditTestSession) Protocol() string { return "sftp" }
func (*remoteEditTestSession) Host() string     { return "example.test" }
func (*remoteEditTestSession) Port() int        { return 22 }
func (s *remoteEditTestSession) List(context.Context, string) ([]model.Item, error) {
	size := int64(len(s.content))
	if s.sizeOverride != 0 {
		size = s.sizeOverride
	}
	return []model.Item{{
		Name: path.Base(s.remotePath), Size: size,
		IsDirectory: s.isDirectory, IsSymlink: s.isSymlink, Permissions: s.permissions,
	}}, nil
}
func (*remoteEditTestSession) Mkdir(context.Context, string, string) error          { return nil }
func (*remoteEditTestSession) Rename(context.Context, string, string, string) error { return nil }
func (*remoteEditTestSession) Delete(context.Context, string, string, bool) error   { return nil }
func (s *remoteEditTestSession) Chmod(_ context.Context, _, _ string, mode string) error {
	s.chmodModes = append(s.chmodModes, mode)
	return s.chmodErr
}
func (s *remoteEditTestSession) Upload(_ context.Context, local, _ string, _ remote.TransferOptions) error {
	data, err := os.ReadFile(local)
	if err != nil {
		return err
	}
	s.uploads++
	s.content = append(s.content[:0], data...)
	return nil
}
func (s *remoteEditTestSession) Download(_ context.Context, _ string, local string, _ remote.TransferOptions) error {
	s.downloads++
	return os.WriteFile(local, s.content, 0600)
}
func (*remoteEditTestSession) Close() error { return nil }

func newRemoteEditTestSession(content string) *remoteEditTestSession {
	return &remoteEditTestSession{
		remotePath:  "/site/config.txt",
		content:     []byte(content),
		permissions: "-rw-r-----",
	}
}

func assertNoRemoteEditWorkspace(t *testing.T, base string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(base, ".GhostFTP-edit-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("remote edit workspace leaked: %v", matches)
	}
}

func TestRemoteEditOpenReturnsBoundedUTF8RevisionAndCleansWorkspace(t *testing.T) {
	base := t.TempDir()
	s := newRemoteEditTestSession("hello\nworld\n")
	doc, err := openRemoteEditWithSession(context.Background(), base, s, s.remotePath)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Text != "hello\nworld\n" || doc.Size != int64(len(doc.Text)) {
		t.Fatalf("unexpected document: %+v", doc)
	}
	if !validRemoteEditRevision(doc.Revision) {
		t.Fatalf("invalid revision %q", doc.Revision)
	}
	if doc.Permissions != "-rw-r-----" {
		t.Fatalf("permissions=%q", doc.Permissions)
	}
	if s.downloads != 1 {
		t.Fatalf("downloads=%d, want 1", s.downloads)
	}
	assertNoRemoteEditWorkspace(t, base)
}

func TestRemoteEditRejectsBinaryAndOversizedFiles(t *testing.T) {
	base := t.TempDir()
	binary := newRemoteEditTestSession("")
	binary.content = []byte{0xff, 0xfe, 0x00}
	if _, err := openRemoteEditWithSession(context.Background(), base, binary, binary.remotePath); !errors.Is(err, ErrRemoteEditBinary) {
		t.Fatalf("binary error=%v, want ErrRemoteEditBinary", err)
	}
	assertNoRemoteEditWorkspace(t, base)

	large := newRemoteEditTestSession("small-placeholder")
	large.sizeOverride = MaxRemoteEditBytes + 1
	if _, err := openRemoteEditWithSession(context.Background(), base, large, large.remotePath); !errors.Is(err, ErrRemoteEditTooLarge) {
		t.Fatalf("large error=%v, want ErrRemoteEditTooLarge", err)
	}
	if large.downloads != 0 {
		t.Fatalf("oversized file was downloaded %d times", large.downloads)
	}
}

func TestRemoteEditRejectsDirectoriesAndSymlinksBeforeDownload(t *testing.T) {
	for _, mutate := range []func(*remoteEditTestSession){
		func(s *remoteEditTestSession) { s.isDirectory = true },
		func(s *remoteEditTestSession) { s.isSymlink = true },
	} {
		s := newRemoteEditTestSession("content")
		mutate(s)
		if _, err := openRemoteEditWithSession(context.Background(), t.TempDir(), s, s.remotePath); err == nil {
			t.Fatal("expected non-regular remote entry rejection")
		}
		if s.downloads != 0 {
			t.Fatal("non-regular remote entry reached download")
		}
	}
}

func TestRemoteEditSaveDetectsConcurrentChangeBeforeUpload(t *testing.T) {
	base := t.TempDir()
	s := newRemoteEditTestSession("version one\n")
	doc, err := openRemoteEditWithSession(context.Background(), base, s, s.remotePath)
	if err != nil {
		t.Fatal(err)
	}
	s.content = []byte("changed elsewhere\n")
	_, err = saveRemoteEditWithSession(context.Background(), base, s, s.remotePath, doc.Revision, "my edit\n")
	if !errors.Is(err, ErrRemoteEditConflict) {
		t.Fatalf("save error=%v, want ErrRemoteEditConflict", err)
	}
	if s.uploads != 0 {
		t.Fatalf("conflicting edit uploaded %d times", s.uploads)
	}
	if string(s.content) != "changed elsewhere\n" {
		t.Fatalf("conflicting content was modified: %q", s.content)
	}
	assertNoRemoteEditWorkspace(t, base)
}

func TestRemoteEditSavePreservesReportedModeAndVerifiesReadBack(t *testing.T) {
	base := t.TempDir()
	s := newRemoteEditTestSession("old\n")
	s.permissions = "-rwxr-x---"
	doc, err := openRemoteEditWithSession(context.Background(), base, s, s.remotePath)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := saveRemoteEditWithSession(context.Background(), base, s, s.remotePath, doc.Revision, "new value\n")
	if err != nil {
		t.Fatal(err)
	}
	if string(s.content) != "new value\n" || updated.Text != "new value\n" {
		t.Fatalf("content not updated: remote=%q doc=%q", s.content, updated.Text)
	}
	if s.uploads != 1 {
		t.Fatalf("uploads=%d, want 1", s.uploads)
	}
	if len(s.chmodModes) != 2 || s.chmodModes[0] != "750" || s.chmodModes[1] != "750" {
		t.Fatalf("chmod modes=%v, want [750 750]", s.chmodModes)
	}
	if updated.Revision == doc.Revision || !validRemoteEditRevision(updated.Revision) {
		t.Fatalf("revision did not advance safely: before=%q after=%q", doc.Revision, updated.Revision)
	}
	assertNoRemoteEditWorkspace(t, base)
}

func TestRemoteEditSaveFailsBeforeUploadWhenReportedModeCannotBePreserved(t *testing.T) {
	base := t.TempDir()
	s := newRemoteEditTestSession("old\n")
	doc, err := openRemoteEditWithSession(context.Background(), base, s, s.remotePath)
	if err != nil {
		t.Fatal(err)
	}
	s.chmodErr = errors.New("chmod unsupported")
	_, err = saveRemoteEditWithSession(context.Background(), base, s, s.remotePath, doc.Revision, "new\n")
	if err == nil {
		t.Fatal("expected permission preservation failure")
	}
	if s.uploads != 0 || string(s.content) != "old\n" {
		t.Fatalf("save changed content despite failed permission preflight: uploads=%d content=%q", s.uploads, s.content)
	}
	assertNoRemoteEditWorkspace(t, base)
}

func TestRemoteEditPermissionModeHandlesSpecialBits(t *testing.T) {
	cases := map[string]string{
		"0644":       "0644",
		"755":        "755",
		"-rw-r--r--": "644",
		"-rwsr-xr-x": "4755",
		"drwxrwsr-t": "3775",
		"not-a-mode": "",
	}
	for input, want := range cases {
		if got := remoteEditPermissionMode(input); got != want {
			t.Fatalf("remoteEditPermissionMode(%q)=%q, want %q", input, got, want)
		}
	}
}
