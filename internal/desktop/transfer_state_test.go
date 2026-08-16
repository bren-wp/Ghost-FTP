package desktop

import (
	"fmt"
	"testing"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/transfer"
)

func TestApplyTransferEventsToJobsUpdatesAndAppends(t *testing.T) {
	jobs := []model.TransferJob{{ID: "a", Status: "queued"}, {ID: "b", Status: "queued"}}
	updated := model.TransferJob{ID: "b", Status: "done", Progress: 100}
	added := model.TransferJob{ID: "c", Status: "running"}
	got, changed := applyTransferEventsToJobs(jobs, []transfer.Event{
		{Type: "job", Job: &updated},
		{Type: "job", Job: &added},
	})
	if !changed || len(got) != 3 {
		t.Fatalf("changed=%v jobs=%d", changed, len(got))
	}
	if got[1].Status != "done" || got[2].ID != "c" {
		t.Fatalf("unexpected jobs: %#v", got)
	}
}

func TestApplyTransferEventsToJobsStateResetsIndex(t *testing.T) {
	jobs := []model.TransferJob{{ID: "old", Status: "done"}}
	state := []model.TransferJob{{ID: "new", Status: "queued"}}
	updated := model.TransferJob{ID: "new", Status: "running"}
	got, changed := applyTransferEventsToJobs(jobs, []transfer.Event{
		{Type: "state", Jobs: state},
		{Type: "job", Job: &updated},
	})
	if !changed || len(got) != 1 || got[0].ID != "new" || got[0].Status != "running" {
		t.Fatalf("unexpected state application: %#v", got)
	}
}

func TestApplyTransferEventsToJobsLargeBatch(t *testing.T) {
	jobs := make([]model.TransferJob, 20000)
	for i := range jobs {
		jobs[i] = model.TransferJob{ID: fmt.Sprintf("job-%05d", i), Status: "queued"}
	}
	events := make([]transfer.Event, 1000)
	updates := make([]model.TransferJob, len(events))
	for i := range events {
		updates[i] = model.TransferJob{ID: fmt.Sprintf("job-%05d", i*19), Status: "done", Progress: 100}
		events[i] = transfer.Event{Type: "job", Job: &updates[i]}
	}
	got, changed := applyTransferEventsToJobs(jobs, events)
	if !changed || len(got) != len(jobs) {
		t.Fatalf("changed=%v jobs=%d want=%d", changed, len(got), len(jobs))
	}
	for i := range events {
		idx := i * 19
		if got[idx].Status != "done" {
			t.Fatalf("job %d was not updated", idx)
		}
	}
}
