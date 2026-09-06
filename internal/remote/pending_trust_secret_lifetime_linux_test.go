//go:build linux

package remote

import (
	"testing"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func pendingTrustLinuxTestConfig() model.ConnectionConfig {
	return model.ConnectionConfig{
		Protocol:       "sftp",
		Host:           "example.test",
		Port:           22,
		Username:       "user",
		PrivateKeyPath: "key",
	}
}

func TestCancelPendingTrustForgetsOwnedLinuxSecrets(t *testing.T) {
	cfg := pendingTrustLinuxTestConfig()
	cfg.Password = "pending-password"
	cfg.Passphrase = "pending-passphrase"
	m := &Manager{}
	if err := m.stashPendingTrust(cfg, resolvedConnection{Config: cfg}, managerTestFingerprint); err != nil {
		t.Fatal(err)
	}
	passwordBlob := m.pendingTrust.passwordBlob
	passphraseBlob := m.pendingTrust.passphraseBlob
	defer security.ForgetProtectedSecret(passwordBlob)
	defer security.ForgetProtectedSecret(passphraseBlob)
	if !m.pendingTrust.ownsPasswordBlob || !m.pendingTrust.ownsPassphraseBlob {
		t.Fatal("plaintext trust credentials were not marked as pending-owned")
	}
	requireProtectedSecretAvailable(t, passwordBlob)
	requireProtectedSecretAvailable(t, passphraseBlob)

	m.CancelPendingTrust()

	requireProtectedSecretForgotten(t, passwordBlob)
	requireProtectedSecretForgotten(t, passphraseBlob)
}

func TestCancelPendingTrustPreservesBorrowedLinuxSecrets(t *testing.T) {
	passwordBlob, err := security.ProtectString("borrowed-profile-password")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetProtectedSecret(passwordBlob)
	passphraseBlob, err := security.ProtectString("borrowed-profile-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetProtectedSecret(passphraseBlob)

	cfg := pendingTrustLinuxTestConfig()
	m := &Manager{}
	if err := m.stashPendingTrust(cfg, resolvedConnection{
		Config:         cfg,
		PasswordBlob:   passwordBlob,
		PassphraseBlob: passphraseBlob,
	}, managerTestFingerprint); err != nil {
		t.Fatal(err)
	}
	if m.pendingTrust.ownsPasswordBlob || m.pendingTrust.ownsPassphraseBlob {
		t.Fatal("borrowed profile secrets became pending-owned")
	}

	m.CancelPendingTrust()

	requireProtectedSecretAvailable(t, passwordBlob)
	requireProtectedSecretAvailable(t, passphraseBlob)
}

func TestPendingTrustMismatchForgetsOwnedLinuxSecrets(t *testing.T) {
	cfg := pendingTrustLinuxTestConfig()
	cfg.Password = "pending-password"
	cfg.Passphrase = "pending-passphrase"
	m := &Manager{}
	if err := m.stashPendingTrust(cfg, resolvedConnection{Config: cfg}, managerTestFingerprint); err != nil {
		t.Fatal(err)
	}
	passwordBlob := m.pendingTrust.passwordBlob
	passphraseBlob := m.pendingTrust.passphraseBlob
	defer security.ForgetProtectedSecret(passwordBlob)
	defer security.ForgetProtectedSecret(passphraseBlob)

	confirmCfg := cfg
	confirmCfg.Password = ""
	confirmCfg.Passphrase = ""
	resolved := resolvedConnection{Config: confirmCfg}
	m.applyPendingTrust(confirmCfg, &resolved, "SHA256:mismatched")

	if resolved.PasswordBlob != "" || resolved.PassphraseBlob != "" {
		t.Fatal("mismatched pending trust credentials were transferred")
	}
	requireProtectedSecretForgotten(t, passwordBlob)
	requireProtectedSecretForgotten(t, passphraseBlob)
}

func TestExpiredPendingTrustForgetsOwnedLinuxSecrets(t *testing.T) {
	cfg := pendingTrustLinuxTestConfig()
	cfg.Password = "pending-password"
	cfg.Passphrase = "pending-passphrase"
	m := &Manager{}
	if err := m.stashPendingTrust(cfg, resolvedConnection{Config: cfg}, managerTestFingerprint); err != nil {
		t.Fatal(err)
	}
	passwordBlob := m.pendingTrust.passwordBlob
	passphraseBlob := m.pendingTrust.passphraseBlob
	defer security.ForgetProtectedSecret(passwordBlob)
	defer security.ForgetProtectedSecret(passphraseBlob)
	m.pendingTrust.expires = time.Now().Add(-time.Second)

	confirmCfg := cfg
	confirmCfg.Password = ""
	confirmCfg.Passphrase = ""
	resolved := resolvedConnection{Config: confirmCfg}
	m.applyPendingTrust(confirmCfg, &resolved, managerTestFingerprint)

	if resolved.PasswordBlob != "" || resolved.PassphraseBlob != "" {
		t.Fatal("expired pending trust credentials were transferred")
	}
	requireProtectedSecretForgotten(t, passwordBlob)
	requireProtectedSecretForgotten(t, passphraseBlob)
}

func TestPendingTrustOwnedSecretsTransferToSFTPSession(t *testing.T) {
	cfg := pendingTrustLinuxTestConfig()
	cfg.Password = "pending-password"
	cfg.Passphrase = "pending-passphrase"
	m := &Manager{}
	if err := m.stashPendingTrust(cfg, resolvedConnection{Config: cfg}, managerTestFingerprint); err != nil {
		t.Fatal(err)
	}
	passwordBlob := m.pendingTrust.passwordBlob
	passphraseBlob := m.pendingTrust.passphraseBlob
	defer security.ForgetProtectedSecret(passwordBlob)
	defer security.ForgetProtectedSecret(passphraseBlob)

	confirmCfg := cfg
	confirmCfg.Password = ""
	confirmCfg.Passphrase = ""
	resolved := resolvedConnection{
		Config:         confirmCfg,
		PasswordBlob:   "stale-profile-password",
		PassphraseBlob: "stale-profile-passphrase",
	}
	m.applyPendingTrust(confirmCfg, &resolved, managerTestFingerprint)
	if resolved.PasswordBlob != passwordBlob || resolved.PassphraseBlob != passphraseBlob {
		t.Fatal("captured pending trust credentials did not replace stale profile blobs")
	}
	if !resolved.ownsPasswordBlob || !resolved.ownsPassphraseBlob {
		t.Fatal("pending secret ownership was not transferred to resolved connection")
	}

	s := &SFTP{passwordBlob: resolved.PasswordBlob, passphraseBlob: resolved.PassphraseBlob}
	transferResolvedSecretOwnershipToSFTP(&resolved, s)
	if resolved.ownsPasswordBlob || resolved.ownsPassphraseBlob {
		t.Fatal("resolved connection retained ownership after SFTP transfer")
	}
	if !s.ownsPasswordBlob || !s.ownsPassphraseBlob {
		t.Fatal("SFTP session did not receive pending secret ownership")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	requireProtectedSecretForgotten(t, passwordBlob)
	requireProtectedSecretForgotten(t, passphraseBlob)
}
