package remote

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type TransferProgressFunc func(transferred, total int64)

const transferProgressInterval = 500 * time.Millisecond

func reportTransferProgress(report TransferProgressFunc, transferred, total int64) {
	if report == nil {
		return
	}
	if transferred < 0 {
		transferred = 0
	}
	if total < 0 {
		total = 0
	}
	if total > 0 && transferred > total {
		transferred = total
	}
	report(transferred, total)
}

func bestEffortRemoteFileSize(ctx context.Context, remotePath string, list func(context.Context, string) ([]model.Item, error)) int64 {
	if list == nil {
		return 0
	}
	cleaned := path.Clean(remotePath)
	dir, base := path.Split(cleaned)
	if base == "" || base == "." || base == ".." {
		return 0
	}
	if dir == "" {
		dir = "."
	}
	dir = path.Clean(dir)
	items, err := list(ctx, dir)
	if err != nil {
		return 0
	}
	item, ok := remoteEntry(items, base)
	if !ok || item.IsDirectory || item.IsSymlink || item.Size < 0 {
		return 0
	}
	return item.Size
}

func startLocalFileProgressMonitor(ctx context.Context, file string, total int64, report TransferProgressFunc) func() {
	if report == nil {
		return func() {}
	}
	reportTransferProgress(report, 0, total)
	done := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		ticker := time.NewTicker(transferProgressInterval)
		defer ticker.Stop()
		reportCurrent := func() {
			st, err := os.Stat(file)
			if err != nil || !st.Mode().IsRegular() {
				return
			}
			reportTransferProgress(report, st.Size(), total)
		}
		for {
			select {
			case <-ticker.C:
				reportCurrent()
			case <-done:
				reportCurrent()
				return
			case <-ctx.Done():
				reportCurrent()
				return
			}
		}
	}()
	return func() {
		select {
		case <-stopped:
			return
		default:
		}
		close(done)
		<-stopped
	}
}
