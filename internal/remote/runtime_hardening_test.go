package remote

import (
	"context"
	"testing"
)

func TestNonNilContextUsesBackgroundForNilInput(t *testing.T) {
	ctx := nonNilContext(nil)
	if ctx == nil {
		t.Fatal("nil context was not normalized")
	}
	select {
	case <-ctx.Done():
		t.Fatal("background replacement context must not start cancelled")
	default:
	}
}

func TestProbePathForSessionUsesProtocolNamespace(t *testing.T) {
	if got := probePathForSession(nil); got != "/" {
		t.Fatalf("nil/default probe path=%q want /", got)
	}
	if got := probePathForSession(&managerTestSession{}); got != "." {
		t.Fatalf("SFTP probe path=%q want .", got)
	}
}

func TestOperationAcceptsNilContextWithoutPanic(t *testing.T) {
	m, _ := newManagerTestConnection()
	_, opCtx, release, err := m.Operation(nil)
	if err != nil {
		t.Fatal(err)
	}
	if opCtx == nil {
		release()
		t.Fatal("operation returned nil context")
	}
	release()
	if err := m.Disconnect(nil); err != nil {
		t.Fatalf("nil-context disconnect failed: %v", err)
	}
}

func TestDisconnectedManagerAcceptsNilContext(t *testing.T) {
	m := &Manager{}
	if err := m.Disconnect(nil); err != nil {
		t.Fatalf("nil-context disconnect on idle manager: %v", err)
	}
	if _, _, _, err := m.Operation(nil); err == nil {
		t.Fatal("disconnected manager should still reject operation")
	}
}

var _ context.Context = nonNilContext(nil)
