package remote

import (
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/profilebinding"
)

func TestApplyPendingTrustPrefersCapturedCredentialOverReresolvedProfile(t *testing.T) {
	cfg := model.ConnectionConfig{
		Protocol:       "sftp",
		Host:           "example.test",
		Port:           22,
		Username:       "user",
		PrivateKeyPath: "key",
	}
	m := &Manager{pendingTrust: pendingTrustState{
		endpointKey:    profilebinding.EndpointKey(cfg.Protocol, cfg.Host, cfg.Port),
		username:       cfg.Username,
		keyPath:        cfg.PrivateKeyPath,
		fingerprint:    managerTestFingerprint,
		passwordBlob:   "captured-password",
		passphraseBlob: "captured-passphrase",
		expires:        time.Now().Add(time.Minute),
	}}
	resolved := resolvedConnection{
		Config:         cfg,
		PasswordBlob:   "reresolved-profile-password",
		PassphraseBlob: "reresolved-profile-passphrase",
	}

	m.applyPendingTrust(cfg, &resolved, managerTestFingerprint)

	if resolved.PasswordBlob != "captured-password" {
		t.Fatalf("password blob=%q, want captured trust-attempt credential", resolved.PasswordBlob)
	}
	if resolved.PassphraseBlob != "captured-passphrase" {
		t.Fatalf("passphrase blob=%q, want captured trust-attempt credential", resolved.PassphraseBlob)
	}
	if m.pendingTrust.passwordBlob != "" || m.pendingTrust.passphraseBlob != "" {
		t.Fatal("pending trust state was not consumed")
	}
}
