package transfer

import (
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func resetTransferMetrics(job *model.TransferJob, _ bool) {
	if job == nil {
		return
	}
	job.Progress = 0
	job.BytesTransferred = 0
	job.BytesTotal = 0
	job.BytesPerSecond = 0
	job.ETASeconds = 0
	// Start the throughput clock on the first real transport progress sample,
	// not when a worker is merely scheduled. This excludes destination checks,
	// upload snapshot creation and other pre-transfer setup from speed/ETA.
	job.StartedAt = ""
}

func (m *Manager) updateProgress(id string, transferred, total int64) {
	if transferred < 0 {
		transferred = 0
	}
	if total < 0 {
		total = 0
	}
	if total > 0 && transferred > total {
		transferred = total
	}
	now := time.Now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.jobs {
		job := &m.jobs[i]
		if job.ID != id || job.Status != "running" {
			continue
		}
		if job.StartedAt == "" {
			job.StartedAt = now.Format(time.RFC3339Nano)
		}
		job.BytesTransferred = transferred
		job.BytesTotal = total
		if total > 0 {
			job.Progress = float64(transferred) * 100 / float64(total)
			if job.Progress > 100 {
				job.Progress = 100
			}
		}
		started, err := time.Parse(time.RFC3339Nano, job.StartedAt)
		if err == nil {
			elapsed := now.Sub(started).Seconds()
			if elapsed >= 0.25 && transferred > 0 {
				job.BytesPerSecond = float64(transferred) / elapsed
				if total > transferred && job.BytesPerSecond > 0 {
					job.ETASeconds = int64(float64(total-transferred)/job.BytesPerSecond + 0.5)
				} else {
					job.ETASeconds = 0
				}
			}
		}
		copy := *job
		m.emitLocked(Event{Type: "job", Job: &copy})
		return
	}
}
