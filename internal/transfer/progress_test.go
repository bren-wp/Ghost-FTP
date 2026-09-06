package transfer

import (
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestUpdateProgressCalculatesRuntimeMetrics(t *testing.T) {
	started := time.Now().UTC().Add(-2 * time.Second).Format(time.RFC3339Nano)
	m := &Manager{jobs: []model.TransferJob{{ID: "job", Status: "running", StartedAt: started}}}
	m.updateProgress("job", 512, 1024)
	job := m.jobs[0]
	if job.BytesTransferred != 512 || job.BytesTotal != 1024 {
		t.Fatalf("bytes=%d/%d", job.BytesTransferred, job.BytesTotal)
	}
	if job.Progress < 49.9 || job.Progress > 50.1 {
		t.Fatalf("progress=%f", job.Progress)
	}
	if job.BytesPerSecond <= 0 {
		t.Fatalf("speed=%f", job.BytesPerSecond)
	}
	if job.ETASeconds < 1 || job.ETASeconds > 3 {
		t.Fatalf("eta=%d", job.ETASeconds)
	}
	if len(m.events) != 1 || m.events[0].Job == nil {
		t.Fatalf("expected one progress event, got %#v", m.events)
	}
}

func TestResetTransferMetricsDefersClockUntilFirstProgressSample(t *testing.T) {
	job := model.TransferJob{
		Status:           "running",
		Progress:         75,
		BytesTransferred: 75,
		BytesTotal:       100,
		BytesPerSecond:   25,
		ETASeconds:       1,
		StartedAt:        time.Now().UTC().Add(-time.Minute).Format(time.RFC3339Nano),
	}
	resetTransferMetrics(&job, true)
	if job.StartedAt != "" || job.Progress != 0 || job.BytesTransferred != 0 || job.BytesTotal != 0 || job.BytesPerSecond != 0 || job.ETASeconds != 0 {
		t.Fatalf("metrics were not fully reset: %+v", job)
	}

	job.ID = "job"
	m := &Manager{jobs: []model.TransferJob{job}}
	before := time.Now().UTC().Add(-time.Second)
	m.updateProgress("job", 0, 1024)
	started, err := time.Parse(time.RFC3339Nano, m.jobs[0].StartedAt)
	if err != nil {
		t.Fatalf("StartedAt parse: %v", err)
	}
	if started.Before(before) || started.After(time.Now().UTC().Add(time.Second)) {
		t.Fatalf("StartedAt=%v was not set by first progress sample", started)
	}
	if m.jobs[0].BytesPerSecond != 0 || m.jobs[0].ETASeconds != 0 {
		t.Fatalf("first zero-byte sample should not invent speed/ETA: %+v", m.jobs[0])
	}
}

func TestUpdateProgressClampsReportedBytes(t *testing.T) {
	started := time.Now().UTC().Add(-time.Second).Format(time.RFC3339Nano)
	m := &Manager{jobs: []model.TransferJob{{ID: "job", Status: "running", StartedAt: started}}}
	m.updateProgress("job", 4096, 1024)
	job := m.jobs[0]
	if job.BytesTransferred != 1024 || job.Progress != 100 || job.ETASeconds != 0 {
		t.Fatalf("clamped job=%+v", job)
	}
}

func TestUpdateProgressIgnoresFinishedJob(t *testing.T) {
	m := &Manager{jobs: []model.TransferJob{{ID: "job", Status: "done", Progress: 100}}}
	m.updateProgress("job", 1, 2)
	if m.jobs[0].BytesTransferred != 0 || len(m.events) != 0 {
		t.Fatalf("finished job mutated: %+v events=%d", m.jobs[0], len(m.events))
	}
}
