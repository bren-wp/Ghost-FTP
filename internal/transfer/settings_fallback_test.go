package transfer

import (
	"path/filepath"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/config"
)

func countStateEvents(events []Event) int {
	count := 0
	for _, event := range events {
		if event.Type == "state" {
			count++
		}
	}
	return count
}

func TestManagerWithNilSettingsUsesSafeDefaults(t *testing.T) {
	m := New(staticProvider{session: panicSession{}}, nil)
	job, err := m.Add("download", filepath.Join(t.TempDir(), "download.txt"), "/download.txt")
	if err != nil {
		t.Fatal(err)
	}
	waitForStatus(t, m, job.ID, "done")
}

func TestPauseResumeAreIdempotent(t *testing.T) {
	m := New(disconnectedProvider{}, config.NewSettings(config.New(t.TempDir())))
	_, seq := m.Events(0)

	m.Pause()
	events, seq := m.Events(seq)
	if got := countStateEvents(events); got != 1 {
		t.Fatalf("first Pause emitted %d state events, want 1", got)
	}
	m.Pause()
	events, seq = m.Events(seq)
	if got := countStateEvents(events); got != 0 {
		t.Fatalf("repeated Pause emitted %d state events, want 0", got)
	}

	m.Resume()
	events, seq = m.Events(seq)
	if got := countStateEvents(events); got != 1 {
		t.Fatalf("first Resume emitted %d state events, want 1", got)
	}
	m.Resume()
	events, _ = m.Events(seq)
	if got := countStateEvents(events); got != 0 {
		t.Fatalf("repeated Resume emitted %d state events, want 0", got)
	}
}
