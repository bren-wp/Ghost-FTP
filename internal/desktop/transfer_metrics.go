package desktop

import (
	"fmt"
	"math"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func transferProgressText(job model.TransferJob) string {
	progress := job.Progress
	if job.BytesTotal > 0 {
		progress = float64(job.BytesTransferred) * 100 / float64(job.BytesTotal)
	}
	if progress > 0 && progress <= 1 && job.BytesTotal == 0 {
		progress *= 100
	}
	progress = math.Max(0, math.Min(100, progress))
	if job.BytesTransferred > 0 || job.BytesTotal > 0 {
		return fmt.Sprintf("%.0f%% · %s", progress, formatTransferBytes(job.BytesTransferred))
	}
	return fmt.Sprintf("%.0f%%", progress)
}

func transferRuntimeSuffix(job model.TransferJob) string {
	if job.BytesPerSecond <= 0 || (job.Status != "running" && job.Status != "done") {
		return ""
	}
	text := " · " + formatTransferBytes(int64(job.BytesPerSecond)) + "/s"
	if job.Status == "running" && job.ETASeconds > 0 {
		text += " · " + formatTransferETA(job.ETASeconds)
	}
	return text
}

func formatTransferBytes(bytes int64) string {
	if bytes < 0 {
		bytes = 0
	}
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	value := float64(bytes)
	units := []string{"KB", "MB", "GB", "TB"}
	for _, suffix := range units {
		value /= unit
		if value < unit || suffix == units[len(units)-1] {
			if value >= 100 {
				return fmt.Sprintf("%.0f %s", value, suffix)
			}
			if value >= 10 {
				return fmt.Sprintf("%.1f %s", value, suffix)
			}
			return fmt.Sprintf("%.2f %s", value, suffix)
		}
	}
	return fmt.Sprintf("%d B", bytes)
}

func formatTransferETA(seconds int64) string {
	if seconds < 0 {
		seconds = 0
	}
	d := time.Duration(seconds) * time.Second
	hours := int64(d / time.Hour)
	minutes := int64((d % time.Hour) / time.Minute)
	secs := int64((d % time.Minute) / time.Second)
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%02d:%02d", minutes, secs)
}
