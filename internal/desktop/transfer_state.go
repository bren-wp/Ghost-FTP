package desktop

import (
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/transfer"
)

// applyTransferEventsToJobs applies an event batch in O(current jobs + events)
// time. Earlier UI code linearly scanned the full job slice for every event,
// which became noticeably expensive when a large queue emitted hundreds of
// updates at once.
func applyTransferEventsToJobs(jobs []model.TransferJob, events []transfer.Event) ([]model.TransferJob, bool) {
	if len(events) == 0 {
		return jobs, false
	}
	index := make(map[string]int, len(jobs))
	rebuildIndex := func() {
		clear(index)
		for i := range jobs {
			index[jobs[i].ID] = i
		}
	}
	rebuildIndex()
	changed := false
	for _, event := range events {
		switch event.Type {
		case "state":
			jobs = append([]model.TransferJob(nil), event.Jobs...)
			rebuildIndex()
			changed = true
		case "job":
			if event.Job == nil {
				continue
			}
			if i, ok := index[event.Job.ID]; ok {
				jobs[i] = *event.Job
			} else {
				index[event.Job.ID] = len(jobs)
				jobs = append(jobs, *event.Job)
			}
			changed = true
		}
	}
	return jobs, changed
}
