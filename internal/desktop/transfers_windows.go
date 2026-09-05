//go:build windows

package desktop

import (
	"fmt"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/transfer"
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
	a.setStatus(a.tr("status.queued"))
	a.goSafe(func() {
		_, err := a.engine.AddTransfer(direction, local, remotePath, localRoot)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil {
				a.setStatus(a.userMessage(err, "error.generic"))
				platform.ErrorDialog("Ghost FTP", a.tr("status.failed"), a.userMessage(err, "error.generic"))
				return
			}
			a.setStatus(a.tr("status.queued"))
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
	for i, job := range a.transferJobs {
		if _, ok := selected[job.ID]; ok {
			setListRowSelected(a.transferList, i, true)
		}
	}
}

func (a *app) refreshTransfers() {
	selected := a.selectedTransferIDSet()
	events, seq := a.engine.TransferEvents(a.transferSeq)
	a.transferSeq = seq
	if !a.applyTransferEvents(events) {
		a.updateTransferSummary()
		a.updateActionControls()
		return
	}
	fillTransfers(a.transferList, a.transferJobs)
	a.restoreTransferSelection(selected)
	a.updateTransferSummary()
	a.updateActionControls()
	refreshPanels := false
	for _, j := range a.transferJobs {
		if j.Status == "done" && !a.seenDone[j.ID] {
			a.seenDone[j.ID] = true
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
	a.setStatus(a.tr("transfer.pause"))
	a.refreshTransfers()
	a.updateActionControls()
}

func (a *app) resumeTransfers() {
	a.engine.ResumeTransfers()
	a.queuePaused = false
	a.setStatus(a.tr("transfer.resume"))
	a.refreshTransfers()
	a.updateActionControls()
}

func (a *app) clearFinishedTransfers() {
	a.engine.ClearFinishedTransfers()
	// IDs are used only to avoid refreshing both file panels repeatedly for the
	// same completed job. Once terminal jobs are removed, retain no stale IDs.
	a.seenDone = make(map[string]bool)
	a.setStatus(a.tr("transfer.clear"))
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
			return nil, fmt.Errorf("transfer selection is no longer valid")
		}
		job := a.transferJobs[idx]
		if !validStatus(job.Status) {
			return nil, fmt.Errorf("one of the selected transfers has an incompatible status")
		}
		ids = append(ids, job.ID)
	}
	return ids, nil
}

func (a *app) cancelSelectedTransfer() {
	ids, err := a.selectedTransferIDs(func(status string) bool { return status == "queued" || status == "running" })
	if err != nil {
		a.setStatus(a.tr("common.cancel"))
		return
	}
	a.goSafe(func() {
		err := a.engine.CancelTransfers(ids)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("Ghost FTP", a.tr("status.failed"), a.userMessage(err, "error.generic"))
				return
			}
			a.setStatus(fmt.Sprintf("%s: %d", a.tr("status.cancelled"), len(ids)))
			a.refreshTransfers()
		})
	})
}

func (a *app) retrySelectedTransfer() {
	ids, err := a.selectedTransferIDs(func(status string) bool { return status == "failed" || status == "cancelled" })
	if err != nil {
		a.setStatus(a.tr("transfer.retry"))
		return
	}
	if !a.connected {
		a.setStatus(a.tr("error.not_connected"))
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
				platform.ErrorDialog("Ghost FTP", a.tr("status.failed"), a.userMessage(err, "error.generic"))
				return
			}
			a.setStatus(fmt.Sprintf("%s: %d", a.tr("status.queued"), len(ids)))
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
