package transfer

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/config"
)

func newPausedQueueManager(t *testing.T) (*Manager, string) {
	t.Helper()
	dir := t.TempDir()
	m := New(disconnectedProvider{}, config.NewSettings(config.New(dir)))
	m.Pause()
	return m, dir
}

func addQueuedDownloads(t *testing.T, m *Manager, dir string, names ...string) []string {
	t.Helper()
	requests := make([]Request, 0, len(names))
	for _, name := range names {
		requests = append(requests, Request{
			Direction:  "download",
			LocalPath:  filepath.Join(dir, name),
			RemotePath: "/" + name,
		})
	}
	jobs, err := m.AddBatch(requests)
	if err != nil {
		t.Fatalf("AddBatch: %v", err)
	}
	ids := make([]string, len(jobs))
	for i := range jobs {
		ids[i] = jobs[i].ID
	}
	return ids
}

func queueIDs(m *Manager) []string {
	jobs := m.List()
	ids := make([]string, len(jobs))
	for i := range jobs {
		ids[i] = jobs[i].ID
	}
	return ids
}

func assertQueueIDs(t *testing.T, m *Manager, want ...string) {
	t.Helper()
	got := queueIDs(m)
	if len(got) != len(want) {
		t.Fatalf("queue length = %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("queue[%d] = %q, want %q; full order=%v", i, got[i], want[i], got)
		}
	}
}

func TestMoveQueuedUpAndDownChangesOnlySchedulerOrder(t *testing.T) {
	m, dir := newPausedQueueManager(t)
	ids := addQueuedDownloads(t, m, dir, "a.txt", "b.txt", "c.txt")

	if err := m.MoveQueuedUp(ids[2]); err != nil {
		t.Fatalf("MoveQueuedUp: %v", err)
	}
	assertQueueIDs(t, m, ids[0], ids[2], ids[1])

	if err := m.MoveQueuedUp(ids[2]); err != nil {
		t.Fatalf("MoveQueuedUp second time: %v", err)
	}
	assertQueueIDs(t, m, ids[2], ids[0], ids[1])

	if err := m.MoveQueuedDown(ids[2]); err != nil {
		t.Fatalf("MoveQueuedDown: %v", err)
	}
	assertQueueIDs(t, m, ids[0], ids[2], ids[1])
}

func TestMoveQueuedSkipsRunningAndTerminalSlots(t *testing.T) {
	m, dir := newPausedQueueManager(t)
	ids := addQueuedDownloads(t, m, dir, "a.txt", "b.txt", "c.txt", "d.txt")

	m.mu.Lock()
	m.jobs[1].Status = "running"
	m.jobs[2].Status = "done"
	runningID := m.jobs[1].ID
	terminalID := m.jobs[2].ID
	m.mu.Unlock()

	if err := m.MoveQueuedUp(ids[3]); err != nil {
		t.Fatalf("MoveQueuedUp: %v", err)
	}
	assertQueueIDs(t, m, ids[3], runningID, terminalID, ids[0])

	jobs := m.List()
	if jobs[1].ID != runningID || jobs[1].Status != "running" {
		t.Fatalf("running job was mutated: %#v", jobs[1])
	}
	if jobs[2].ID != terminalID || jobs[2].Status != "done" {
		t.Fatalf("terminal job was mutated: %#v", jobs[2])
	}
}

func TestMoveQueuedRejectsMissingAndNonQueuedJobs(t *testing.T) {
	m, dir := newPausedQueueManager(t)
	ids := addQueuedDownloads(t, m, dir, "a.txt")

	if err := m.MoveQueuedUp("missing"); !errors.Is(err, errTransferOrderMissing) {
		t.Fatalf("missing transfer error = %v, want %v", err, errTransferOrderMissing)
	}

	m.mu.Lock()
	m.jobs[0].Status = "running"
	m.mu.Unlock()
	if err := m.MoveQueuedDown(ids[0]); !errors.Is(err, errTransferOrderNotQueued) {
		t.Fatalf("non-queued transfer error = %v, want %v", err, errTransferOrderNotQueued)
	}
}

func TestMoveQueuedAtEdgeIsIdempotentAndDoesNotEmitNoise(t *testing.T) {
	m, dir := newPausedQueueManager(t)
	ids := addQueuedDownloads(t, m, dir, "a.txt", "b.txt")
	_, before := m.Events(0)

	if err := m.MoveQueuedUp(ids[0]); err != nil {
		t.Fatalf("MoveQueuedUp at edge: %v", err)
	}
	if err := m.MoveQueuedDown(ids[1]); err != nil {
		t.Fatalf("MoveQueuedDown at edge: %v", err)
	}
	assertQueueIDs(t, m, ids[0], ids[1])

	events, after := m.Events(before)
	if after != before || len(events) != 0 {
		t.Fatalf("edge no-op emitted queue events: before=%d after=%d events=%#v", before, after, events)
	}
}

func TestMoveQueuedEmitsFullStateSnapshot(t *testing.T) {
	m, dir := newPausedQueueManager(t)
	ids := addQueuedDownloads(t, m, dir, "a.txt", "b.txt", "c.txt")
	_, before := m.Events(0)

	if err := m.MoveQueuedUp(ids[2]); err != nil {
		t.Fatalf("MoveQueuedUp: %v", err)
	}
	events, after := m.Events(before)
	if after <= before {
		t.Fatalf("event sequence did not advance: before=%d after=%d", before, after)
	}
	if len(events) != 1 || events[0].Type != "state" {
		t.Fatalf("reorder events = %#v, want one state snapshot", events)
	}
	if !events[0].Paused {
		t.Fatal("state snapshot lost paused queue state")
	}
	if len(events[0].Jobs) != 3 || events[0].Jobs[0].ID != ids[0] || events[0].Jobs[1].ID != ids[2] || events[0].Jobs[2].ID != ids[1] {
		t.Fatalf("state snapshot order = %#v", events[0].Jobs)
	}
}

func TestMoveQueuedRejectsClosedManager(t *testing.T) {
	m, dir := newPausedQueueManager(t)
	ids := addQueuedDownloads(t, m, dir, "a.txt")
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()

	if err := m.MoveQueuedDown(ids[0]); !errors.Is(err, errTransferOrderClosed) {
		t.Fatalf("closed manager error = %v, want %v", err, errTransferOrderClosed)
	}
}
