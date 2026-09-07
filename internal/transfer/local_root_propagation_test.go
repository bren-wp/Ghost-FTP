package transfer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
)

type rootCaptureSession struct {
	root string
}

func (s *rootCaptureSession) Protocol() string { return "sftp" }
func (s *rootCaptureSession) Host() string     { return "example.test" }
func (s *rootCaptureSession) Port() int        { return 22 }
func (s *rootCaptureSession) List(context.Context, string) ([]model.Item, error) {
	return nil, nil
}
func (s *rootCaptureSession) Mkdir(context.Context, string, string) error { return nil }
func (s *rootCaptureSession) Rename(context.Context, string, string, string) error {
	return nil
}
func (s *rootCaptureSession) Delete(context.Context, string, string, bool) error { return nil }
func (s *rootCaptureSession) Chmod(context.Context, string, string, string) error {
	return nil
}
func (s *rootCaptureSession) Upload(context.Context, string, string, remote.TransferOptions) error {
	return nil
}
func (s *rootCaptureSession) Download(_ context.Context, _, _ string, options remote.TransferOptions) error {
	s.root = options.LocalRoot
	return nil
}
func (s *rootCaptureSession) Close() error { return nil }

type rootCaptureProvider struct {
	session *rootCaptureSession
}

func (p rootCaptureProvider) Operation(ctx context.Context) (remote.Session, context.Context, func(), error) {
	return p.session, ctx, func() {}, nil
}
func (rootCaptureProvider) ConnectionIdentity() (string, error) { return "root-test", nil }

func TestRunAttemptPreservesExplicitDownloadRoot(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "tree", "file.txt")
	if err := os.MkdirAll(filepath.Dir(local), 0755); err != nil {
		t.Fatal(err)
	}
	session := &rootCaptureSession{}
	m := New(rootCaptureProvider{session: session}, config.NewSettings(config.New(t.TempDir())))
	job := model.TransferJob{ID: "test", Direction: "download", LocalPath: local, LocalRoot: root, RemotePath: "/file.txt"}
	if err := m.runAttempt(context.Background(), job, config.DefaultSettings()); err != nil {
		t.Fatalf("runAttempt: %v", err)
	}
	if session.root != root {
		t.Fatalf("download LocalRoot=%q want %q", session.root, root)
	}
}

func TestRunAttemptDerivesSingleFileDownloadRoot(t *testing.T) {
	root := t.TempDir()
	local := filepath.Join(root, "file.txt")
	session := &rootCaptureSession{}
	m := New(rootCaptureProvider{session: session}, config.NewSettings(config.New(t.TempDir())))
	job := model.TransferJob{ID: "test", Direction: "download", LocalPath: local, RemotePath: "/file.txt"}
	if err := m.runAttempt(context.Background(), job, config.DefaultSettings()); err != nil {
		t.Fatalf("runAttempt: %v", err)
	}
	if session.root != filepath.Dir(local) {
		t.Fatalf("derived LocalRoot=%q want %q", session.root, filepath.Dir(local))
	}
}
