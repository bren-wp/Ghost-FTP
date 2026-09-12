package remote

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestPendingTrustPreservesOwnedResolvedSecretCapabilities(t *testing.T) {
	m := &Manager{}
	resolved := resolvedConnection{
		PasswordBlob:       "owned-password-capability",
		PassphraseBlob:     "owned-passphrase-capability",
		ownsPasswordBlob:   true,
		ownsPassphraseBlob: true,
	}
	cfg := model.ConnectionConfig{
		Protocol:       "sftp",
		Host:           "example.test",
		Port:           22,
		Username:       "user",
		PrivateKeyPath: "/tmp/key",
	}
	if err := m.stashPendingTrust(cfg, resolved, "SHA256:test"); err != nil {
		t.Fatalf("stashPendingTrust: %v", err)
	}
	if m.pendingTrust.passwordBlob != resolved.PasswordBlob || !m.pendingTrust.ownsPasswordBlob {
		t.Fatal("owned password capability was not transferred into pending trust")
	}
	if m.pendingTrust.passphraseBlob != resolved.PassphraseBlob || !m.pendingTrust.ownsPassphraseBlob {
		t.Fatal("owned passphrase capability was not transferred into pending trust")
	}

	// Do not ask the platform secret store to destroy synthetic test handles.
	m.pendingTrust.ownsPasswordBlob = false
	m.pendingTrust.ownsPassphraseBlob = false
	m.clearPendingTrustLocked()
}
