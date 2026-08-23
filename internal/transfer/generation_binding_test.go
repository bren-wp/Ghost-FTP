package transfer

import (
	"context"
	"errors"
	"testing"

	"github.com/bren-wp/by-ftp/internal/model"
	"github.com/bren-wp/by-ftp/internal/remote"
)

type generationHookProvider struct {
	identity string
	hook     func()
}

func (p *generationHookProvider) Operation(context.Context) (remote.Session, context.Context, func(), error) {
	return nil, nil, func() {}, errors.New("operation is not used by generation binding tests")
}

func (p *generationHookProvider) ConnectionIdentity() (string, error) {
	if p.hook != nil {
		p.hook()
	}
	return p.identity, nil
}

func incrementGeneration(m *Manager) {
	m.mu.Lock()
	m.generation++
	m.mu.Unlock()
}

func TestReserveBatchRejectsGenerationChangeDuringConnectionIdentity(t *testing.T) {
	provider := &generationHookProvider{identity: "connection-before-reconnect"}
	m := New(provider, nil)
	provider.hook = func() { incrementGeneration(m) }

	reservation, err := m.ReserveBatch([]Request{{
		Direction: "upload", LocalPath: "local.txt", RemotePath: "/remote.txt",
	}})
	if err == nil {
		if reservation != nil {
			reservation.Cancel()
		}
		t.Fatal("reservation must fail when queue generation changes during ConnectionIdentity")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.reserved != 0 {
		t.Fatalf("failed reservation leaked capacity: reserved=%d", m.reserved)
	}
	if len(m.jobs) != 0 {
		t.Fatalf("failed reservation mutated queue: %+v", m.jobs)
	}
}

func TestRetryBatchRejectsGenerationChangeDuringConnectionIdentity(t *testing.T) {
	provider := &generationHookProvider{identity: "connection-before-reconnect"}
	m := New(provider, nil)
	m.mu.Lock()
	m.paused = true // old buggy code would otherwise reach pump with nil settings
	m.jobs = []model.TransferJob{{ID: "job-1", Status: "failed", Error: "old failure", Progress: 25, Attempts: 2}}
	m.jobConnections["job-1"] = provider.identity
	m.mu.Unlock()
	provider.hook = func() { incrementGeneration(m) }

	if err := m.RetryBatch([]string{"job-1"}); err == nil {
		t.Fatal("retry must fail when queue generation changes during ConnectionIdentity")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if got := m.jobs[0]; got.Status != "failed" || got.Error != "old failure" || got.Progress != 25 || got.Attempts != 2 {
		t.Fatalf("failed retry mutated job: %+v", got)
	}
}
