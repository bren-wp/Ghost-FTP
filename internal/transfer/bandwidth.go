package transfer

import (
	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

const bytesPerKiB = int64(1024)

// bandwidthLimitBytesPerSecond converts the persisted aggregate directional
// budget into a conservative per-worker transport cap. The scheduler can run at
// most Settings.Parallelism workers, so dividing the aggregate budget across all
// configured slots guarantees that simultaneously active transfers cannot
// exceed the configured directional ceiling. Idle slots deliberately do not
// lend bandwidth to active slots; this keeps the aggregate guarantee stable as
// jobs start and finish without mutating a live transport process.
//
// A limit of zero means unlimited. Settings changes are sampled immediately
// before each transfer attempt, so an already-running attempt keeps its stable
// transport snapshot while a new or retried attempt observes the new value.
func bandwidthLimitBytesPerSecond(settings model.Settings, direction string) int64 {
	limitKiB := 0
	switch direction {
	case "upload":
		limitKiB = settings.UploadLimitKiBPerSecond
	case "download":
		limitKiB = settings.DownloadLimitKiBPerSecond
	default:
		return 0
	}
	if limitKiB <= 0 {
		return 0
	}

	parallelism := settings.Parallelism
	if parallelism < config.MinParallelism || parallelism > config.MaxParallelism {
		// Effective settings are already validated. If a caller nevertheless
		// supplies malformed state, use the most conservative slot count so the
		// aggregate ceiling cannot be accidentally weakened.
		parallelism = config.MaxParallelism
	}
	return int64(limitKiB) * bytesPerKiB / int64(parallelism)
}
