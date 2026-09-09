package remote

import (
	"errors"
	"strings"
	"testing"
)

func TestSSHKeyscanFailureRedactsRawDiagnostic(t *testing.T) {
	raw := "ssh-keyscan: Could not resolve hostname secret-host.example for private-user: Name or service not known"
	err := sshKeyscanFailure(errors.New("exit status 255"), raw)
	got := err.Error()

	if !strings.Contains(strings.ToLower(got), "host resolution failed") {
		t.Fatalf("safe semantic resolution signal was lost: %q", got)
	}
	for _, sensitive := range []string{"secret-host.example", "private-user", "name or service not known"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(sensitive)) {
			t.Fatalf("ssh-keyscan diagnostic leaked %q: %q", sensitive, got)
		}
	}

	var te *toolError
	if !errors.As(err, &te) {
		t.Fatalf("wrapped error must retain private tool classification: %T", err)
	}
	if got := te.UserErrorKind(); got != "resolve" {
		t.Fatalf("UserErrorKind()=%q want resolve", got)
	}
}

func TestSSHKeyscanFailureDoesNotEchoUnknownDiagnostic(t *testing.T) {
	raw := "opaque failure at C:/Users/private-user/.ssh/config for secret-host.example"
	err := sshKeyscanFailure(errors.New("exit status 1"), raw)
	got := err.Error()
	for _, sensitive := range []string{"private-user", ".ssh/config", "secret-host.example", "opaque failure"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(sensitive)) {
			t.Fatalf("unknown ssh-keyscan diagnostic leaked %q: %q", sensitive, got)
		}
	}
}

func TestSSHKeyscanFailureNilCauseIsStable(t *testing.T) {
	if got, want := sshKeyscanFailure(nil, "secret-host.example").Error(), "nije moguće dohvatiti SFTP host ključ"; got != want {
		t.Fatalf("Error()=%q want %q", got, want)
	}
}
