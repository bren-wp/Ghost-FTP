//go:build windows

package desktop

import (
	"fmt"

	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/transfer"
)

func (a *app) updateTransferSummary() {
	queued, running, done, failed, skipped := 0, 0, 0, 0, 0
	for _, job := range a.transferJobs {
		switch job.Status {
		case "queued":
			queued++
		case "running":
			running++
		case "done":
			done++
		case "failed", "cancelled":
			failed++
		case "skipped":
			skipped++
		}
	}
	text := a.tr("transfer.summary", running, queued, done)
	if skipped > 0 {
		text += a.tr("transfer.summary_skipped", skipped)
	}
	if failed > 0 {
		text += a.tr("transfer.summary_failed", failed)
	}
	setText(a.transferSummary, text)
}

func (a *app) addTransfer(direction, local, remotePath, localRoot string) {
	generation := a.connectionGeneration
	a.setStatus(a.tr("transfer.adding"))
	a.goSafe(func() {
		_, err := a.engine.AddTransfer(direction, local, remotePath, localRoot)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil {
				a.setStatus(a.userMessage(err, "transfer.start_failed_status"))
				platform.ErrorDialog(a.tr("transfer.tree_title"), a.tr("transfer.start_failed_title"), a.userMessage(err, "transfer.start_failed_body"))
				return
			}
			a.setStatus(a.tr("transfer.queued"))
			a.refreshTransfers()
		})
	})
}

func (a *app) applyTransferEvents(events []transfer.Event) bool {
	for _, event := range events {
		if event.Type == "state" {
			a.queuePaused = event.Paused
		}
	}
	jobs, changed := applyTransferEventsToJobs(a.transferJobs, events)
	a.transferJobs = jobs
	return changed
}

func (a *app) selectedTransferIDSet() map[string]struct{} {
	selected := make(map[string]struct{})
	for _, index := range selectedIndices(a.transferList) {
		if index >= 0 && index < len(a.transferJobs) {
			selected[a.transferJobs[index].ID] = struct{}{}
		}
	}
	return selected
}

func (a *app) restoreTransferSelection(selected map[string]struct{}) {
	if len(selected) == 0 {
		return
	}
	for index, job := range a.transferJobs {
		if _, ok := selected[job.ID]; ok {
			setListRowSelected(a.transferList, index, true)
		}
	}
}

func (a *app) refreshTransfers() {
	selected := a.selectedTransferIDSet()
	events, seq := a.engine.TransferEvents(a.transferSeq)
	a.transferSeq = seq
	if !a.applyTransferEvents(events) {
		a.updateActionControls()
		return
	}
	a.fillTransferList(a.transferList, a.transferJobs)
	a.restoreTransferSelection(selected)
	a.updateTransferSummary()
	a.updateActionControls()
	refreshPanels := false
	for _, job := range a.transferJobs {
		if job.Status == "done" && !a.seenDone[job.ID] {
			a.seenDone[job.ID] = true
			refreshPanels = true
		}
	}
	if refreshPanels {
		a.refreshLocal(a.localCurrent)
		if a.connected {
			a.refreshRemote(a.remoteCurrent)
		}
	}
}

func (a *app) pauseTransfers() {
	a.engine.PauseTransfers()
	a.queuePaused = true
	a.setStatus(a.tr("transfer.pause_status"))
	a.refreshTransfers()
	a.updateActionControls()
}

func (a *app) resumeTransfers() {
	a.engine.ResumeTransfers()
	a.queuePaused = false
	a.setStatus(a.tr("transfer.resume_status"))
	a.refreshTransfers()
	a.updateActionControls()
}

func (a *app) clearFinishedTransfers() {
	a.engine.ClearFinishedTransfers()
	// IDs are only used to avoid refreshing both file panels repeatedly for the
	// same completed job. Once terminal jobs are removed, keep no stale IDs.
	a.seenDone = make(map[string]bool)
	a.setStatus(a.tr("transfer.clear_status"))
	a.refreshTransfers()
	a.updateActionControls()
}

func (a *app) selectedTransferIDs(validStatus func(string) bool) ([]string, error) {
	indices := selectedIndices(a.transferList)
	if len(indices) == 0 {
		return nil, fmt.Errorf("no transfer selected")
	}
	ids := make([]string, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(a.transferJobs) {
			return nil, fmt.Errorf("transfer selection is stale")
		}
		job := a.transferJobs[idx]
		if !validStatus(job.Status) {
			return nil, fmt.Errorf("selected transfer has an incompatible status")
		}
		ids = append(ids, job.ID)
	}
	return ids, nil
}

func (a *app) cancelSelectedTransfer() {
	ids, err := a.selectedTransferIDs(func(status string) bool { return status == "queued" || status == "running" })
	if err != nil {
		a.setStatus(a.tr("transfer.select_active_cancel"))
		return
	}
	a.goSafe(func() {
		err := a.engine.CancelTransfers(ids)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog(a.tr("section.transfers"), a.tr("action.failed_title"), a.userMessage(err, "transfer.cancel_failed_body"))
				return
			}
			a.setStatus(a.tr("transfer.cancelled_count", len(ids)))
			a.refreshTransfers()
		})
	})
}

func (a *app) retrySelectedTransfer() {
	ids, err := a.selectedTransferIDs(func(status string) bool { return status == "failed" || status == "cancelled" })
	if err != nil {
		a.setStatus(a.tr("transfer.select_retry"))
		return
	}
	if !a.connected {
		a.setStatus(a.tr("transfer.retry_requires_connection"))
		return
	}
	generation := a.connectionGeneration
	a.goSafe(func() {
		err := a.engine.RetryTransfers(ids)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil {
				platform.ErrorDialog(a.tr("section.transfers"), a.tr("action.failed_title"), a.userMessage(err, "transfer.retry_failed_body"))
				return
			}
			a.setStatus(a.tr("transfer.requeued_count", len(ids)))
			a.refreshTransfers()
		})
	})
}

func (a *app) hasActiveTransfers() bool {
	for _, job := range a.transferJobs {
		if job.Status == "queued" || job.Status == "running" {
			return true
		}
	}
	return false
}
