package transfer

import (
	"errors"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

var (
	errTransferOrderClosed    = errors.New("transfer manager is closed")
	errTransferOrderMissing   = errors.New("transfer was not found")
	errTransferOrderNotQueued = errors.New("only queued transfers can be reordered")
)

type queuedMove int

const (
	queuedMoveTop queuedMove = iota
	queuedMoveUp
	queuedMoveDown
	queuedMoveBottom
)

// MoveQueuedTop moves one waiting transfer to the first queued scheduler slot.
// Running and terminal jobs keep their exact list/history slots.
func (m *Manager) MoveQueuedTop(id string) error {
	return m.moveQueued(id, queuedMoveTop)
}

// MoveQueuedUp moves one waiting transfer ahead of the nearest earlier waiting
// transfer. Running and terminal jobs keep their exact list/history slots.
func (m *Manager) MoveQueuedUp(id string) error {
	return m.moveQueued(id, queuedMoveUp)
}

// MoveQueuedDown moves one waiting transfer behind the nearest later waiting
// transfer. Running and terminal jobs are never reordered or mutated.
func (m *Manager) MoveQueuedDown(id string) error {
	return m.moveQueued(id, queuedMoveDown)
}

// MoveQueuedBottom moves one waiting transfer to the final queued scheduler
// slot without moving running or terminal history entries.
func (m *Manager) MoveQueuedBottom(id string) error {
	return m.moveQueued(id, queuedMoveBottom)
}

func (m *Manager) moveQueued(id string, move queuedMove) error {
	if move < queuedMoveTop || move > queuedMoveBottom {
		return errors.New("invalid queue reorder operation")
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

	queuedSlots := make([]int, 0, len(m.jobs))
	queuedPosition := -1
	for i := range m.jobs {
		if m.jobs[i].Status != "queued" {
			continue
		}
		if i == index {
			queuedPosition = len(queuedSlots)
		}
		queuedSlots = append(queuedSlots, i)
	}
	if queuedPosition < 0 {
		return errTransferOrderNotQueued
	}

	destination := queuedPosition
	switch move {
	case queuedMoveTop:
		destination = 0
	case queuedMoveUp:
		if queuedPosition > 0 {
			destination--
		}
	case queuedMoveDown:
		if queuedPosition+1 < len(queuedSlots) {
			destination++
		}
	case queuedMoveBottom:
		destination = len(queuedSlots) - 1
	}
	if destination == queuedPosition {
		// Edge moves are idempotent. A stale UI click after another queue state
		// update must not emit noise or become an operational error.
		return nil
	}

	selected := m.jobs[index]
	if destination < queuedPosition {
		for pos := queuedPosition; pos > destination; pos-- {
			m.jobs[queuedSlots[pos]] = m.jobs[queuedSlots[pos-1]]
		}
	} else {
		for pos := queuedPosition; pos < destination; pos++ {
			m.jobs[queuedSlots[pos]] = m.jobs[queuedSlots[pos+1]]
		}
	}
	m.jobs[queuedSlots[destination]] = selected

	m.emitLocked(Event{
		Type:   "state",
		Jobs:   append([]model.TransferJob(nil), m.jobs...),
		Paused: m.paused,
	})
	return nil
}
