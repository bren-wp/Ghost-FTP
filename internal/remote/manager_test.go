package remote

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/bren-wp/by-ftp/internal/model"
)

type managerTestSession struct {
	closed chan struct{}
}

func (*managerTestSession) Protocol() string { return "sftp" }
func (*managerTestSession) Host() string     { return "example.test" }
func (*managerTestSession) Port() int        { return 22 }
func (*managerTestSession) List(ctx context.Context, _ string) ([]model.Item, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (*managerTestSession) Mkdir(context.Context, string, string) error          { return nil }
func (*managerTestSession) Rename(context.Context, string, string, string) error { return nil }
func (*managerTestSession) Delete(context.Context, string, string, bool) error   { return nil }
func (*managerTestSession) Chmod(context.Context, string, string, string) error  { return nil }
func (*managerTestSession) Upload(context.Context, string, string, TransferOptions) error {
	return nil
}
func (*managerTestSession) Download(context.Context, string, string, TransferOptions) error {
	return nil
}
func (s *managerTestSession) Close() error {
	if s.closed != nil {
		close(s.closed)
	}
	return nil
}

func newManagerTestConnection() (*Manager, *managerTestSession) {
	m := &Manager{}
	s := &managerTestSession{closed: make(chan struct{})}
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	m.session = s
	m.sessionCtx = sessionCtx
	m.sessionCancel = sessionCancel
	return m, s
}

func TestDisconnectWaitsForActiveOperationRelease(t *testing.T) {
	m, session := newManagerTestConnection()
	_, opCtx, release, err := m.Operation(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	disconnectDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		disconnectDone <- m.Disconnect(ctx)
	}()

	select {
	case <-opCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("disconnect did not cancel active remote operation context")
	}

	select {
	case <-session.closed:
		t.Fatal("session was closed before active operation released it")
	default:
	}
	select {
	case err := <-disconnectDone:
		t.Fatalf("disconnect returned before active operation release: %v", err)
	default:
	}

	release()
	select {
	case err := <-disconnectDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("disconnect did not finish after active operation release")
	}
	select {
	case <-session.closed:
	default:
		t.Fatal("session was not closed after active operation released it")
	}
	m.mu.RLock()
	closing := m.closing != nil
	m.mu.RUnlock()
	if closing {
		t.Fatal("disconnect returned before closing state was cleared")
	}
}

func TestDisconnectTimeoutDefersCloseAndBlocksReconnect(t *testing.T) {
	m, session := newManagerTestConnection()
	_, opCtx, release, err := m.Operation(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	err = m.Disconnect(ctx)
	if !errors.Is(err, ErrDisconnectTimeout) {
		t.Fatalf("disconnect error=%v, want ErrDisconnectTimeout", err)
	}

	select {
	case <-opCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("timed-out disconnect did not cancel active operation context")
	}
	select {
	case <-session.closed:
		t.Fatal("timed-out disconnect closed adapter before operation release")
	default:
	}
	if _, _, _, err := m.Operation(context.Background()); err == nil {
		t.Fatal("new operation was admitted while old session was closing")
	}
	if _, err := m.Connect(context.Background(), "", model.ConnectionConfig{}, "", false); !errors.Is(err, ErrSessionClosing) {
		t.Fatalf("reconnect error=%v, want ErrSessionClosing", err)
	}

	release()
	select {
	case <-session.closed:
	case <-time.After(time.Second):
		t.Fatal("deferred cleanup did not close session after release")
	}

	if err := m.Disconnect(context.Background()); err != nil {
		t.Fatalf("second disconnect after cleanup: %v", err)
	}
	m.mu.RLock()
	closing := m.closing != nil
	m.mu.RUnlock()
	if closing {
		t.Fatal("closing state remained after completed deferred cleanup")
	}
}

func TestDisconnectCancellationDefersClose(t *testing.T) {
	m, session := newManagerTestConnection()
	_, opCtx, release, err := m.Operation(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = m.Disconnect(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("disconnect error=%v, want context.Canceled", err)
	}
	// context.AfterFunc propagira session cancellation asinkrono. Ne zahtijevaj
	// sinkrono izvršenje callbacka; zahtijevaj da se signal ipak pojavi brzo.
	select {
	case <-opCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("cancelled disconnect did not cancel active session context")
	}
	select {
	case <-session.closed:
		t.Fatal("cancelled disconnect closed adapter before operation release")
	default:
	}

	release()
	select {
	case <-session.closed:
	case <-time.After(time.Second):
		t.Fatal("cleanup did not finish after release following cancelled wait")
	}
	if err := m.Disconnect(context.Background()); err != nil {
		t.Fatalf("cleanup state after cancelled wait: %v", err)
	}
}

func TestSecondDisconnectWaitsForExistingCloseState(t *testing.T) {
	m, _ := newManagerTestConnection()
	_, _, release, err := m.Operation(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	err = m.Disconnect(ctx)
	cancel()
	if !errors.Is(err, ErrDisconnectTimeout) {
		t.Fatalf("first disconnect error=%v", err)
	}

	done := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		done <- m.Disconnect(ctx)
	}()
	select {
	case err := <-done:
		t.Fatalf("second disconnect returned before release: %v", err)
	default:
	}

	release()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("second disconnect did not observe deferred close completion")
	}
}

func TestOperationReleaseIsIdempotent(t *testing.T) {
	m, _ := newManagerTestConnection()
	_, _, release, err := m.Operation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	release()
	release()
	if err := m.Disconnect(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestOperationRejectsDisconnectedManager(t *testing.T) {
	m := &Manager{}
	if _, _, _, err := m.Operation(context.Background()); err == nil {
		t.Fatal("expected disconnected manager to reject operation")
	}
}

func TestApplyPendingTrustUsesProtectedSecretsOnce(t *testing.T) {
	m := &Manager{pendingTrust: pendingTrustState{
		host: "example.test", port: 22, username: "user", keyPath: "key", fingerprint: "SHA256:test",
		passwordBlob: "protected-password", passphraseBlob: "protected-passphrase", expires: time.Now().Add(time.Minute),
	}}
	resolved := resolvedConnection{Config: model.ConnectionConfig{Host: "example.test", Port: 22, Username: "user", PrivateKeyPath: "key"}}
	m.applyPendingTrust(resolved.Config, &resolved, "SHA256:test")
	if resolved.PasswordBlob != "protected-password" || resolved.PassphraseBlob != "protected-passphrase" {
		t.Fatalf("pending protected credentials were not applied: %#v", resolved)
	}
	if m.pendingTrust.passwordBlob != "" || m.pendingTrust.passphraseBlob != "" {
		t.Fatal("pending trust credentials were not cleared after one use")
	}
}

func TestApplyPendingTrustRejectsDifferentHost(t *testing.T) {
	m := &Manager{pendingTrust: pendingTrustState{
		host: "first.example", port: 22, username: "user", fingerprint: "SHA256:test",
		passwordBlob: "protected-password", expires: time.Now().Add(time.Minute),
	}}
	resolved := resolvedConnection{Config: model.ConnectionConfig{Host: "second.example", Port: 22, Username: "user"}}
	m.applyPendingTrust(resolved.Config, &resolved, "SHA256:test")
	if resolved.PasswordBlob != "" {
		t.Fatal("pending credential crossed connection identity boundary")
	}
	if m.pendingTrust.passwordBlob != "" {
		t.Fatal("mismatched pending credential was not discarded")
	}
}
