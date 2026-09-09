//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxTransferActionStateUsesSharedStatusPolicy(t *testing.T) {
	u := &linuxDesktop{
		connected:        true,
		selectedTransfer: 0,
		transferJobs: []model.TransferJob{
			{ID: "active", Status: "running"},
			{ID: "failed", Status: "failed"},
		},
	}

	state := u.linuxTransferActionState()
	if !state.Pause || state.Resume || !state.Cancel || state.Retry || !state.Clear {
		t.Fatalf("unexpected action state for running selection: %+v", state)
	}

	u.selectedTransfer = 1
	state = u.linuxTransferActionState()
	if state.Cancel || !state.Retry || !state.Clear {
		t.Fatalf("failed selection should expose retry but not cancel: %+v", state)
	}

	u.connected = false
	state = u.linuxTransferActionState()
	if state.Retry || state.Pause || state.Resume {
		t.Fatalf("disconnected Linux UI must not expose connection-bound queue actions: %+v", state)
	}
}

func TestLinuxTransferActionStateRejectsStaleSelection(t *testing.T) {
	u := &linuxDesktop{
		connected:        true,
		selectedTransfer: 9,
		transferJobs:     []model.TransferJob{{ID: "queued", Status: "queued"}},
	}
	state := u.linuxTransferActionState()
	if state.Cancel || state.Retry {
		t.Fatalf("stale selection must not expose item mutation actions: %+v", state)
	}
	if !state.Pause {
		t.Fatalf("queue-wide pause must remain available for active work: %+v", state)
	}
}

func TestLinuxBusyQueueMutationsAreInert(t *testing.T) {
	pause := &linuxDesktop{
		busy:             true,
		connected:        true,
		selectedTransfer: 0,
		transferJobs:     []model.TransferJob{{ID: "queued", Status: "queued"}},
	}
	pause.pauseTransfersLinux()
	if pause.queuePaused {
		t.Fatal("busy pause click must be inert")
	}

	resume := &linuxDesktop{
		busy:             true,
		connected:        true,
		queuePaused:      true,
		selectedTransfer: 0,
		transferJobs:     []model.TransferJob{{ID: "queued", Status: "queued"}},
	}
	resume.resumeTransfersLinux()
	if !resume.queuePaused {
		t.Fatal("busy resume click must be inert")
	}

	cancel := &linuxDesktop{
		busy:             true,
		connected:        true,
		selectedTransfer: 0,
		transferJobs:     []model.TransferJob{{ID: "queued", Status: "queued"}},
	}
	cancel.cancelSelectedTransferLinux()
	if cancel.transferJobs[0].Status != "queued" {
		t.Fatalf("busy cancel click mutated transfer: %+v", cancel.transferJobs[0])
	}

	retry := &linuxDesktop{
		busy:             true,
		connected:        true,
		selectedTransfer: 0,
		transferJobs:     []model.TransferJob{{ID: "failed", Status: "failed"}},
	}
	retry.retrySelectedTransferLinux()
	if retry.transferJobs[0].Status != "failed" {
		t.Fatalf("busy retry click mutated transfer: %+v", retry.transferJobs[0])
	}

	clear := &linuxDesktop{
		busy:             true,
		selectedTransfer: 0,
		transferJobs:     []model.TransferJob{{ID: "done", Status: "done"}},
	}
	clear.clearFinishedTransfersLinux()
	if len(clear.transferJobs) != 1 || clear.transferJobs[0].Status != "done" {
		t.Fatalf("busy clear click mutated queue: %+v", clear.transferJobs)
	}
}
