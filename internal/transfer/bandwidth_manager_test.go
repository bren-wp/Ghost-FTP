package transfer

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
)

type bandwidthCaptureSession struct {
	mu              sync.Mutex
	uploadOptions   []remote.TransferOptions
	downloadOptions []remote.TransferOptions
	firstUpload     chan struct{}
	failFirstUpload bool
}

func (s *bandwidthCaptureSession) Protocol() string {
	return "ftp"
}

func (s *bandwidthCaptureSession) Host() string {
	return "example.test"
}

func (s *bandwidthCaptureSession) Port() int {
	return 21
}

func (s *bandwidthCaptureSession) List(context.Context, string) ([]model.Item, error) {
	return nil, nil
}

func (s *bandwidthCaptureSession) Mkdir(context.Context, string, string) error {
	return nil
}

func (s *bandwidthCaptureSession) Rename(context.Context, string, string, string) error {
	return nil
}

func (s *bandwidthCaptureSession) Delete(context.Context, string, string, bool) error {
	return nil
}

func (s *bandwidthCaptureSession) Chmod(context.Context, string, string, string) error {
	return nil
}

func (s *bandwidthCaptureSession) Upload(_ context.Context, _, _ string, options remote.TransferOptions) error {
	s.mu.Lock()
	s.uploadOptions = append(s.uploadOptions, options)
	call := len(s.uploadOptions)
	fail := s.failFirstUpload && call == 1
	first := s.firstUpload
	s.mu.Unlock()
	if call == 1 && first != nil {
		close(first)
	}
	if fail {
		return errors.New("connection reset by peer")
	}
	return nil
}

func (s *bandwidthCaptureSession) Download(_ context.Context, _, _ string, options remote.TransferOptions) error {
	s.mu.Lock()
	s.downloadOptions = append(s.downloadOptions, options)
	s.mu.Unlock()
	return nil
}

func (s *bandwidthCaptureSession) Close() error {
	return nil
}

func TestManagerPropagatesIndependentAggregateBandwidthCaps(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	candidate := config.DefaultSettings()
	candidate.Parallelism = 4
	candidate.UploadLimitKiBPerSecond = 4096
	candidate.DownloadLimitKiBPerSecond = 8192
	if _, err := settings.Set(candidate); err != nil {
		t.Fatal(err)
	}

	session := &bandwidthCaptureSession{}
	m := New(staticProvider{session: session}, settings)
	upload, err := m.Add("upload", filepath.Join(dir, "upload.bin"), "/upload.bin")
	if err != nil {
		t.Fatal(err)
	}
	download, err := m.Add("download", filepath.Join(dir, "download.bin"), "/download.bin")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, upload.ID, "done")
	waitForStatus(t, m, download.ID, "done")

	session.mu.Lock()
	defer session.mu.Unlock()
	if len(session.uploadOptions) != 1 || len(session.downloadOptions) != 1 {
		t.Fatalf("unexpected transfer calls: upload=%d download=%d", len(session.uploadOptions), len(session.downloadOptions))
	}
	if got, want := session.uploadOptions[0].BandwidthLimitBytesPerSecond, int64(1048576); got != want {
		t.Fatalf("upload transport cap=%d want=%d", got, want)
	}
	if got, want := session.downloadOptions[0].BandwidthLimitBytesPerSecond, int64(2097152); got != want {
		t.Fatalf("download transport cap=%d want=%d", got, want)
	}
}

func TestRetryAttemptSamplesNewBandwidthLimitWithoutMutatingFirstAttempt(t *testing.T) {
	dir := t.TempDir()
	settings := config.NewSettings(config.New(dir))
	candidate := config.DefaultSettings()
	candidate.Parallelism = 2
	candidate.UploadLimitKiBPerSecond = 2048
	candidate.AutoRetryCount = 1
	candidate.RetryDelaySeconds = 1
	if _, err := settings.Set(candidate); err != nil {
		t.Fatal(err)
	}

	session := &bandwidthCaptureSession{firstUpload: make(chan struct{}), failFirstUpload: true}
	m := New(staticProvider{session: session}, settings)
	job, err := m.Add("upload", filepath.Join(dir, "retry.bin"), "/retry.bin")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.firstUpload:
	case <-time.After(2 * time.Second):
		t.Fatal("first upload attempt did not start")
	}

	updated := candidate
	updated.UploadLimitKiBPerSecond = 8192
	if _, err := settings.Set(updated); err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "done")

	session.mu.Lock()
	defer session.mu.Unlock()
	if len(session.uploadOptions) != 2 {
		t.Fatalf("upload attempts=%d want=2", len(session.uploadOptions))
	}
	if got, want := session.uploadOptions[0].BandwidthLimitBytesPerSecond, int64(1048576); got != want {
		t.Fatalf("first attempt cap changed unexpectedly: got=%d want=%d", got, want)
	}
	if got, want := session.uploadOptions[1].BandwidthLimitBytesPerSecond, int64(4194304); got != want {
		t.Fatalf("retry did not sample new cap: got=%d want=%d", got, want)
	}
}
