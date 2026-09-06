package remote

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestBestEffortRemoteFileSize(t *testing.T) {
	got := bestEffortRemoteFileSize(context.Background(), "/public/file.bin", func(_ context.Context, dir string) ([]model.Item, error) {
		if dir != "/public" {
			t.Fatalf("dir=%q", dir)
		}
		return []model.Item{{Name: "file.bin", Size: 8192}, {Name: "folder", IsDirectory: true}}, nil
	})
	if got != 8192 {
		t.Fatalf("size=%d", got)
	}
}

func TestProgressMonitorReportsActualStagingBytes(t *testing.T) {
	dir := t.TempDir()
	part := filepath.Join(dir, "part")
	type sample struct{ transferred, total int64 }
	samples := make(chan sample, 8)
	stop := startLocalFileProgressMonitor(context.Background(), part, 10, func(transferred, total int64) {
		samples <- sample{transferred, total}
	})
	if err := os.WriteFile(part, []byte("12345"), 0600); err != nil {
		t.Fatal(err)
	}
	time.Sleep(transferProgressInterval + 100*time.Millisecond)
	stop()
	close(samples)
	found := false
	for s := range samples {
		if s.transferred == 5 && s.total == 10 {
			found = true
		}
	}
	if !found {
		t.Fatal("monitor never reported actual staging size")
	}
}
