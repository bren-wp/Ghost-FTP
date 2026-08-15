package remote

import (
	"context"
	"testing"
	"time"

	"brendigo.com/byftp/internal/model"
)

type managerTestSession struct{}

func (managerTestSession) Protocol() string { return "sftp" }
func (managerTestSession) Host() string     { return "example.test" }
func (managerTestSession) Port() int        { return 22 }
func (managerTestSession) List(ctx context.Context, _ string) ([]model.Item, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
func (managerTestSession) Mkdir(context.Context, string, string) error                   { return nil }
func (managerTestSession) Rename(context.Context, string, string, string) error          { return nil }
func (managerTestSession) Delete(context.Context, string, string, bool) error            { return nil }
func (managerTestSession) Chmod(context.Context, string, string, string) error           { return nil }
func (managerTestSession) Upload(context.Context, string, string, TransferOptions) error { return nil }
func (managerTestSession) Download(context.Context, string, string, TransferOptions) error {
	return nil
}
func (managerTestSession) Close() error { return nil }

func TestOperationContextIsCancelledByDisconnect(t *testing.T) {
	m := &Manager{}
	sessionCtx, sessionCancel := context.WithCancel(context.Background())
	m.session = managerTestSession{}
	m.sessionCtx = sessionCtx
	m.sessionCancel = sessionCancel

	_, opCtx, release, err := m.Operation(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := m.Disconnect(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-opCtx.Done():
	case <-time.After(time.Second):
		t.Fatal("disconnect did not cancel active remote operation context")
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
