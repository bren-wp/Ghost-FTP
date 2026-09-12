package transfer

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestSnapshotCopiesJobsAndPausedState(t *testing.T) {
	manager := &Manager{
		jobs: []model.TransferJob{{
			ID:     "job-1",
			Status: "queued",
		}},
		paused: true,
	}

	jobs, paused := manager.Snapshot()
	if !paused {
		t.Fatal("Snapshot() lost the authoritative paused state")
	}
	if len(jobs) != 1 || jobs[0].ID != "job-1" || jobs[0].Status != "queued" {
		t.Fatalf("Snapshot() returned unexpected jobs: %#v", jobs)
	}

	jobs[0].Status = "failed"
	jobs = append(jobs, model.TransferJob{ID: "injected", Status: "done"})

	again, stillPaused := manager.Snapshot()
	if !stillPaused {
		t.Fatal("mutating the returned snapshot changed manager pause state")
	}
	if len(again) != 1 {
		t.Fatalf("mutating the returned slice changed manager queue length: %d", len(again))
	}
	if again[0].Status != "queued" {
		t.Fatalf("mutating the returned snapshot changed manager job state: %q", again[0].Status)
	}
}
