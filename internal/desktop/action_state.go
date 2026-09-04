package desktop

import "github.com/bren-wp/Ghost-FTP/internal/model"

type transferActionState struct {
	Pause  bool
	Resume bool
	Cancel bool
	Retry  bool
	Clear  bool
}

func deriveTransferActionState(jobs []model.TransferJob, selected []int, connected, paused bool) transferActionState {
	state := transferActionState{}
	hasActive := false
	hasTerminal := false
	for _, job := range jobs {
		switch job.Status {
		case "queued", "running":
			hasActive = true
		case "done", "failed", "cancelled", "skipped":
			hasTerminal = true
		}
	}
	state.Pause = connected && hasActive && !paused
	state.Resume = connected && hasActive && paused
	state.Clear = hasTerminal
	if len(selected) == 0 {
		return state
	}

	cancelOK := true
	retryOK := connected
	for _, index := range selected {
		if index < 0 || index >= len(jobs) {
			cancelOK = false
			retryOK = false
			break
		}
		switch jobs[index].Status {
		case "queued", "running":
			retryOK = false
		case "failed", "cancelled":
			cancelOK = false
		default:
			cancelOK = false
			retryOK = false
		}
	}
	state.Cancel = cancelOK
	state.Retry = retryOK
	return state
}
