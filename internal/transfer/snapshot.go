package transfer

import "github.com/bren-wp/Ghost-FTP/internal/model"

// Snapshot returns an isolated copy of the queue plus the manager's
// authoritative paused state. Callers can render queue controls without
// maintaining a second scheduler state machine.
func (m *Manager) Snapshot() ([]model.TransferJob, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]model.TransferJob(nil), m.jobs...), m.paused
}
