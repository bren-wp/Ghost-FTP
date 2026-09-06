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
