package transfer

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestBandwidthLimitIsDirectionalAndAggregate(t *testing.T) {
	settings := model.Settings{}
	settings.Parallelism = 4
	settings.UploadLimitKiBPerSecond = 4096
	settings.DownloadLimitKiBPerSecond = 8192

	if got, want := bandwidthLimitBytesPerSecond(settings, "upload"), int64(1048576); got != want {
		t.Fatalf("upload per-slot cap = %d, want %d", got, want)
	}
	if got, want := bandwidthLimitBytesPerSecond(settings, "download"), int64(2097152); got != want {
		t.Fatalf("download per-slot cap = %d, want %d", got, want)
	}
}

func TestBandwidthUnlimitedDoesNotThrottle(t *testing.T) {
	settings := model.Settings{}
	settings.Parallelism = 8
	if got := bandwidthLimitBytesPerSecond(settings, "upload"); got != 0 {
		t.Fatalf("unlimited upload must return zero transport cap, got %d", got)
	}
	if got := bandwidthLimitBytesPerSecond(settings, "download"); got != 0 {
		t.Fatalf("unlimited download must return zero transport cap, got %d", got)
	}
}

func TestBandwidthMalformedParallelismFailsConservatively(t *testing.T) {
	settings := model.Settings{}
	settings.Parallelism = 0
	settings.UploadLimitKiBPerSecond = 8
	if got, want := bandwidthLimitBytesPerSecond(settings, "upload"), int64(1024); got != want {
		t.Fatalf("invalid parallelism should use eight conservative slots: got %d want %d", got, want)
	}
}

func TestBandwidthUnknownDirectionIsUnthrottled(t *testing.T) {
	settings := model.Settings{}
	settings.Parallelism = 1
	settings.UploadLimitKiBPerSecond = 1024
	settings.DownloadLimitKiBPerSecond = 1024
	if got := bandwidthLimitBytesPerSecond(settings, "sideways"); got != 0 {
		t.Fatalf("unknown direction returned a bandwidth cap: %d", got)
	}
}
