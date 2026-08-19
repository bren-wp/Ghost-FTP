package transfer

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestSelectedIDsTrimsAndDeduplicates(t *testing.T) {
	got, err := selectedIDs([]string{"  one  ", "two", "one"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("selected ID count=%d want 2", len(got))
	}
	if _, ok := got["one"]; !ok {
		t.Fatal("trimmed ID one missing")
	}
	if _, ok := got["two"]; !ok {
		t.Fatal("ID two missing")
	}
}

func TestSelectedIDsRejectsInvalidSelection(t *testing.T) {
	if _, err := selectedIDs(nil); err == nil {
		t.Fatal("empty selection should fail")
	}
	if _, err := selectedIDs([]string{"ok", "   "}); err == nil {
		t.Fatal("blank transfer ID should fail")
	}
}

func TestWaitWorkersAcceptsNilContext(t *testing.T) {
	m := &Manager{}
	if err := m.waitWorkers(nil); err != nil {
		t.Fatalf("nil-context wait failed: %v", err)
	}
}

func TestWaitWorkersTimeoutUsesSharedIdleSignal(t *testing.T) {
	m := &Manager{workers: 1, workersIdle: make(chan struct{})}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := m.waitWorkers(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("waitWorkers error=%v want deadline exceeded", err)
	}
	m.workerExited()
	if err := m.waitWorkers(context.Background()); err != nil {
		t.Fatalf("idle worker signal did not complete: %v", err)
	}
}

func TestValidateRequestRequiresConcreteRemoteFile(t *testing.T) {
	local := filepath.Join(t.TempDir(), "file.txt")
	for _, remotePath := range []string{"/", ".", "/public_html/"} {
		err := validateRequest(Request{Direction: "upload", LocalPath: local, RemotePath: remotePath})
		if err == nil {
			t.Fatalf("directory-like remote target %q was accepted", remotePath)
		}
	}
	if err := validateRequest(Request{Direction: "upload", LocalPath: local, RemotePath: "/public_html/index.html"}); err != nil {
		t.Fatalf("valid remote file was rejected: %v", err)
	}
}
