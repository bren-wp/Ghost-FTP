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
