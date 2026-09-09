package transfer

import "errors"

var (
	errTransferOrderClosed    = errors.New("transfer manager is closed")
	errTransferOrderMissing   = errors.New("transfer was not found")
	errTransferOrderNotQueued = errors.New("only queued transfers can be reordered")
)

// MoveQueuedUp moves one waiting transfer ahead of the nearest earlier waiting
// transfer. Running and terminal jobs keep their exact slots in the history
// list; only the relative scheduler order of queued jobs changes.
func (m *Manager) MoveQueuedUp(id string) error {
	return m.moveQueued(id, -1)
}

// MoveQueuedDown moves one waiting transfer behind the nearest later waiting
// transfer. Running and terminal jobs are never reordered or mutated.
func (m *Manager) MoveQueuedDown(id string) error {
	return m.moveQueued(id, 1)
}

func (m *Manager) moveQueued(id string, direction int) error {
	if direction != -1 && direction != 1 {
		return errors.New("invalid queue reorder direction")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return errTransferOrderClosed
	}

	index := -1
	for i := range m.jobs {
		if m.jobs[i].ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		return errTransferOrderMissing
	}
	if m.jobs[index].Status != "queued" {
		return errTransferOrderNotQueued
	}

	neighbor := -1
	if direction < 0 {
		for i := index - 1; i >= 0; i-- {
			if m.jobs[i].Status == "queued" {
				neighbor = i
				break
			}
		}
	} else {
		for i := index + 1; i < len(m.jobs); i++ {
			if m.jobs[i].Status == "queued" {
				neighbor = i
				break
			}
	}
	if neighbor < 0 {
		// The requested job is already at the corresponding queued edge. This is
		// an idempotent no-op so a stale UI click cannot turn into an error after
		// another queue event changes availability.
		return nil
	}

	m.jobs[index], m.jobs[neighbor] = m.jobs[neighbor], m.jobs[index]
	m.emitLocked(Event{
		Type:   "state",
		Jobs:   append([]model.TransferJob(nil), m.jobs...),
		Paused: m.paused,
	})
	return nil
}
