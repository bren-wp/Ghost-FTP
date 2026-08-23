package transfer

import (
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
)

func TestEventsVracaNeovisneSnimke(t *testing.T) {
	m := &Manager{}
	job := model.TransferJob{ID: "posao-1", Status: "queued", LocalPath: `C:\\sigurno\\datoteka.txt`, RemotePath: "/datoteka.txt"}
	stateJobs := []model.TransferJob{{ID: "posao-2", Status: "running"}}

	m.mu.Lock()
	m.emitLocked(Event{Type: "job", Job: &job})
	m.emitLocked(Event{Type: "state", Jobs: stateJobs})
	m.mu.Unlock()

	first, _ := m.Events(0)
	if len(first) != 2 || first[0].Job == nil || len(first[1].Jobs) != 1 {
		t.Fatalf("neočekivani događaji: %#v", first)
	}
	first[0].Job.Status = "izmijenjeno-izvana"
	first[1].Jobs[0].Status = "izmijenjeno-izvana"

	second, _ := m.Events(0)
	if second[0].Job.Status != "queued" {
		t.Fatalf("vanjska izmjena oštetila je spremljeni job event: %#v", second[0])
	}
	if second[1].Jobs[0].Status != "running" {
		t.Fatalf("vanjska izmjena oštetila je spremljeni state event: %#v", second[1])
	}
}
