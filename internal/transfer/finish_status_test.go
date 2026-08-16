package transfer

import (
	"context"
	"testing"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/remote"
)

func managerWithRunningJob() *Manager {
	return &Manager{
		jobs:           []model.TransferJob{{ID: "job-1", Status: "running"}},
		running:        1,
		cancels:        map[string]context.CancelFunc{},
		jobConnections: map[string]string{},
	}
}

func TestFinishJobKeepsSuccessfulResultWhenCancelArrivesAfterSuccess(t *testing.T) {
	m := managerWithRunningJob()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.finishJob(ctx, "job-1", nil)
	job := m.List()[0]
	if job.Status != "done" || job.Progress != 100 || job.Error != "" {
		t.Fatalf("successful transfer after late cancel = status %q progress %g error %q", job.Status, job.Progress, job.Error)
	}
}

func TestFinishJobKeepsSkippedResultWhenCancelArrivesAfterSkip(t *testing.T) {
	m := managerWithRunningJob()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.finishJob(ctx, "job-1", remote.ErrSkipped)
	job := m.List()[0]
	if job.Status != "skipped" || job.Progress != 100 {
		t.Fatalf("skipped transfer after late cancel = status %q progress %g", job.Status, job.Progress)
	}
}

func TestFinishJobMarksActualCancellation(t *testing.T) {
	m := managerWithRunningJob()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	m.finishJob(ctx, "job-1", context.Canceled)
	job := m.List()[0]
	if job.Status != "cancelled" {
		t.Fatalf("cancelled transfer status = %q", job.Status)
	}
}
