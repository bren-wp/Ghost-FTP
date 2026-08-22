package desktop

import (
	"testing"

	"brendigo.com/byftp/internal/model"
)

func TestDeriveTransferActionState(t *testing.T) {
	jobs := []model.TransferJob{
		{ID: "q", Status: "queued"},
		{ID: "f", Status: "failed"},
		{ID: "d", Status: "done"},
	}
	state := deriveTransferActionState(jobs, []int{0}, true, false)
	if !state.Pause || state.Resume || !state.Cancel || state.Retry || !state.Clear {
		t.Fatalf("unexpected active selection state: %+v", state)
	}

	state = deriveTransferActionState(jobs, []int{1}, true, true)
	if state.Pause || !state.Resume || state.Cancel || !state.Retry || !state.Clear {
		t.Fatalf("unexpected retry selection state: %+v", state)
	}

	state = deriveTransferActionState(jobs, []int{1, 2}, true, false)
	if state.Cancel || state.Retry {
		t.Fatalf("mixed terminal selection must not enable cancel/retry: %+v", state)
	}
}

func TestDeriveTransferActionStateRequiresConnectionForRetry(t *testing.T) {
	jobs := []model.TransferJob{{ID: "f", Status: "failed"}}
	state := deriveTransferActionState(jobs, []int{0}, false, false)
	if state.Retry {
		t.Fatal("retry must be disabled while disconnected")
	}
	if !state.Clear {
		t.Fatal("terminal job should allow clear")
	}
}

func TestDeriveTransferActionStateDisablesQueueControlsWhileDisconnected(t *testing.T) {
	jobs := []model.TransferJob{{ID: "q", Status: "queued"}}
	for _, paused := range []bool{false, true} {
		state := deriveTransferActionState(jobs, nil, false, paused)
		if state.Pause || state.Resume {
			t.Fatalf("queue pause/resume must be disabled while disconnected: %+v", state)
		}
	}
}
