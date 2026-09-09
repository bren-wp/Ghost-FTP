package remote

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func requireToolError(t *testing.T, err error) *toolError {
	t.Helper()
	var te *toolError
	if !errors.As(err, &te) || te == nil {
		t.Fatalf("expected toolError, got %T: %v", err, err)
	}
	return te
}

func TestToolErrorPublicStringRedactsAndDoesNotRetainRawDiagnostics(t *testing.T) {
	raw := "Load key C:/Users/private-user/.ssh/customer-prod: incorrect passphrase for secret-host.example"
	err := newToolError("sftp", errors.New("tool process failed"), raw)
	te := requireToolError(t, err)

	if got, want := te.Error(), "sftp credential settings invalid"; got != want {
		t.Fatalf("Error()=%q want %q", got, want)
	}

	wrapped := fmt.Errorf("transfer failed: %w", te).Error()
	stored := fmt.Sprintf("%+v", *te)
	for _, sensitive := range []string{"private-user", "customer-prod", "secret-host.example", "passphrase"} {
		for label, value := range map[string]string{"public error": wrapped, "retained toolError": stored} {
			if strings.Contains(strings.ToLower(value), strings.ToLower(sensitive)) {
				t.Fatalf("%s leaked child-process diagnostic %q: %q", label, sensitive, value)
			}
		}
	}
}

func TestToolErrorAuthenticationKeepsSafeSignalWithoutRawReply(t *testing.T) {
	raw := "530 Login incorrect for private-user@secret-host.example"
	err := requireToolError(t, newToolError("curl", errors.New("tool process failed"), raw))
	got := err.Error()
	if !strings.Contains(strings.ToLower(got), "login") {
		t.Fatalf("public error lost safe authentication signal: %q", got)
	}
	if err.UserErrorKind() != "auth" {
		t.Fatalf("UserErrorKind()=%q want auth", err.UserErrorKind())
	}
	for _, sensitive := range []string{"530", "private-user", "secret-host.example"} {
		if strings.Contains(strings.ToLower(got), strings.ToLower(sensitive)) {
			t.Fatalf("public error leaked raw authentication diagnostic %q: %q", sensitive, got)
		}
		if strings.Contains(strings.ToLower(fmt.Sprintf("%+v", *err)), strings.ToLower(sensitive)) {
			t.Fatalf("toolError retained raw authentication diagnostic %q", sensitive)
		}
	}
}

func TestToolErrorUnknownToolNameIsNotRetainedOrReflected(t *testing.T) {
	err := requireToolError(t, newToolError("secret-host.example/private-user", errors.New("tool process failed"), "opaque"))
	if got, want := err.Error(), "network tool operation failed"; got != want {
		t.Fatalf("Error()=%q want %q", got, want)
	}
	if err.tool != "" {
		t.Fatalf("unknown tool identifier must be discarded, got %q", err.tool)
	}
}

func TestNilToolErrorIsStableAndRedacted(t *testing.T) {
	var err *toolError
	if got, want := err.Error(), "network tool operation failed"; got != want {
		t.Fatalf("nil Error()=%q want %q", got, want)
	}
}

func TestToolErrorPrivateDiagnosticsStillDriveClassification(t *testing.T) {
	raw := "Load key C:/Users/private-user/.ssh/customer-prod: incorrect passphrase supplied to decrypt private key"
	err := requireToolError(t, newToolError("sftp", errors.New("tool process failed"), raw))
	if got := err.UserErrorKind(); got != "sftp_settings" {
		t.Fatalf("UserErrorKind()=%q want sftp_settings", got)
	}
	stored := fmt.Sprintf("%+v", *err)
	for _, sensitive := range []string{"private-user", "customer-prod", "passphrase"} {
		if strings.Contains(strings.ToLower(stored), strings.ToLower(sensitive)) {
			t.Fatalf("toolError retained private diagnostic %q: %q", sensitive, stored)
		}
	}
}

func TestToolErrorPrivateDiagnosticsPreserveRetryClassification(t *testing.T) {
	retryable := newToolError("sftp", errors.New("tool process failed"), "Connection reset by peer at secret-host.example")
	if !IsRetryable(retryable) {
		t.Fatal("transient SFTP diagnostic must remain retryable after diagnostic retention hardening")
	}

	nonRetryable := newToolError("sftp", errors.New("tool process failed"), "Permission denied (publickey,password) for private-user@secret-host.example")
	if IsRetryable(nonRetryable) {
		t.Fatal("authentication diagnostic must remain non-retryable after diagnostic retention hardening")
	}
}
