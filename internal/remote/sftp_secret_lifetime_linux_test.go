//go:build linux

package remote

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func requireProtectedSecretAvailable(t *testing.T, blob string) {
	t.Helper()
	plain, err := security.UnprotectBytes(blob)
	if err != nil {
		t.Fatalf("protected secret should be available: %v", err)
	}
	security.WipeBytes(plain)
}

func requireProtectedSecretForgotten(t *testing.T, blob string) {
	t.Helper()
	plain, err := security.UnprotectBytes(blob)
	if err == nil {
		security.WipeBytes(plain)
		t.Fatal("protected secret remained available after owner cleanup")
	}
}

func TestSFTPCloseForgetsOwnedLinuxProtectedSecrets(t *testing.T) {
	passwordBlob, err := security.ProtectString("sftp-password-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetProtectedSecret(passwordBlob)

	passphraseBlob, err := security.ProtectString("sftp-passphrase-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetProtectedSecret(passphraseBlob)

	requireProtectedSecretAvailable(t, passwordBlob)
	requireProtectedSecretAvailable(t, passphraseBlob)

	s := &SFTP{
		passwordBlob:       passwordBlob,
		passphraseBlob:     passphraseBlob,
		ownsPasswordBlob:   true,
		ownsPassphraseBlob: true,
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if s.passwordBlob != "" || s.passphraseBlob != "" || s.ownsPasswordBlob || s.ownsPassphraseBlob {
		t.Fatal("SFTP close did not clear protected secret state")
	}

	requireProtectedSecretForgotten(t, passwordBlob)
	requireProtectedSecretForgotten(t, passphraseBlob)

	if err := s.Close(); err != nil {
		t.Fatalf("SFTP close should remain idempotent: %v", err)
	}
}

func TestSFTPClosePreservesBorrowedLinuxProtectedSecrets(t *testing.T) {
	passwordBlob, err := security.ProtectString("profile-password-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetProtectedSecret(passwordBlob)

	passphraseBlob, err := security.ProtectString("profile-passphrase-secret")
	if err != nil {
		t.Fatal(err)
	}
	defer security.ForgetProtectedSecret(passphraseBlob)

	s := &SFTP{passwordBlob: passwordBlob, passphraseBlob: passphraseBlob}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	requireProtectedSecretAvailable(t, passwordBlob)
	requireProtectedSecretAvailable(t, passphraseBlob)
}
