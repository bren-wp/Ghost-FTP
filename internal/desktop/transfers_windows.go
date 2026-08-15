//go:build windows

package desktop

import (
	"fmt"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/transfer"
	"brendigo.com/byftp/internal/usererror"
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
	text := fmt.Sprintf("%d aktivnih  •  %d na čekanju  •  %d završeno", running, queued, done)
	if skipped > 0 {
		text += fmt.Sprintf("  •  %d preskočeno", skipped)
	}
	if failed > 0 {
		text += fmt.Sprintf("  •  %d greška/otkazano", failed)
	}
	setText(a.transferSummary, text)
}

func (a *app) addTransfer(direction, local, remotePath, localRoot string) {
	a.setStatus("Dodavanje prijenosa…")
	a.goSafe(func() {
		_, err := a.engine.AddTransfer(direction, local, remotePath, localRoot)
		a.dispatch(func() {
			if err != nil {
				a.setStatus(usererror.Message(err, "Prijenos nije moguće pokrenuti."))
				platform.ErrorDialog("ByFTP — prijenos", "Prijenos nije pokrenut", usererror.Message(err, "Provjerite vezu i odabrane datoteke."))
				return
			}
			a.setStatus("Prijenos dodan u red čekanja.")
			a.refreshTransfers()
		})
	})
}

func upsertTransferJob(jobs []model.TransferJob, job model.TransferJob) []model.TransferJob {
	for i := range jobs {
		if jobs[i].ID == job.ID {
			jobs[i] = job
			return jobs
		}
	}
	return append(jobs, job)
}

func (a *app) applyTransferEvents(events []transfer.Event) bool {
	changed := false
	for _, event := range events {
		switch event.Type {
		case "state":
			a.transferJobs = append([]model.TransferJob(nil), event.Jobs...)
			changed = true
		case "job":
			if event.Job != nil {
				a.transferJobs = upsertTransferJob(a.transferJobs, *event.Job)
				changed = true
			}
		}
	}
	return changed
}

func (a *app) refreshTransfers() {
	events, seq := a.engine.TransferEvents(a.transferSeq)
	a.transferSeq = seq
	if !a.applyTransferEvents(events) {
		return
	}
	fillTransfers(a.transferList, a.transferJobs)
	a.updateTransferSummary()
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
	a.setStatus("Red prijenosa pauziran.")
	a.refreshTransfers()
}
func (a *app) resumeTransfers() {
	a.engine.ResumeTransfers()
	a.setStatus("Red prijenosa nastavljen.")
	a.refreshTransfers()
}
func (a *app) clearFinishedTransfers() {
	a.engine.ClearFinishedTransfers()
	a.setStatus("Završeni prijenosi uklonjeni iz reda.")
	a.refreshTransfers()
}

func (a *app) selectedTransferIDs(validStatus func(string) bool) ([]string, error) {
	indices := selectedIndices(a.transferList)
	if len(indices) == 0 {
		return nil, fmt.Errorf("nije odabran nijedan prijenos")
	}
	ids := make([]string, 0, len(indices))
	for _, idx := range indices {
		if idx < 0 || idx >= len(a.transferJobs) {
			return nil, fmt.Errorf("odabir prijenosa više nije važeći")
		}
		job := a.transferJobs[idx]
		if !validStatus(job.Status) {
			return nil, fmt.Errorf("jedan od odabranih prijenosa nema odgovarajući status")
		}
		ids = append(ids, job.ID)
	}
	return ids, nil
}

func (a *app) cancelSelectedTransfer() {
	ids, err := a.selectedTransferIDs(func(status string) bool { return status == "queued" || status == "running" })
	if err != nil {
		a.setStatus("Odaberite jedan ili više aktivnih prijenosa za otkazivanje.")
		return
	}
	a.goSafe(func() {
		err := a.engine.CancelTransfers(ids)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP — prijenosi", "Radnja nije uspjela", usererror.Message(err, "Odabrane prijenose trenutačno nije moguće otkazati."))
				return
			}
			a.setStatus(fmt.Sprintf("Otkazano prijenosa: %d", len(ids)))
			a.refreshTransfers()
		})
	})
}

func (a *app) retrySelectedTransfer() {
	ids, err := a.selectedTransferIDs(func(status string) bool { return status == "failed" || status == "cancelled" })
	if err != nil {
		a.setStatus("Odaberite jedan ili više neuspjelih ili otkazanih prijenosa za ponavljanje.")
		return
	}
	if !a.connected {
		a.setStatus("Povežite se s poslužiteljem prije ponavljanja prijenosa.")
		return
	}
	a.goSafe(func() {
		err := a.engine.RetryTransfers(ids)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP — prijenosi", "Radnja nije uspjela", usererror.Message(err, "Odabrane prijenose trenutačno nije moguće ponoviti."))
				return
			}
			a.setStatus(fmt.Sprintf("Ponovno dodano u red: %d", len(ids)))
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
