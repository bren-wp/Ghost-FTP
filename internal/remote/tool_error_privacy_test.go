package remote

import (
	"fmt"
	"strings"
	"testing"
)

func TestToolErrorPublicStringRedactsRawDiagnostics(t *testing.T) {
	raw := "Load key C:/Users/private-user/.ssh/customer-prod: incorrect passphrase for secret-host.example"
	err := &toolError{tool: "sftp", code: 255, message: raw}

	if got, want := err.Error(), "sftp operation failed (exit code 255)"; got != want {
		t.Fatalf("Error()=%q want %q", got, want)
	}

	wrapped := fmt.Errorf("transfer failed: %w", err).Error()
	for _, sensitive := range []string{"private-user", "customer-prod", "secret-host.example", "passphrase"} {
		if strings.Contains(strings.ToLower(wrapped), strings.ToLower(sensitive)) {
			t.Fatalf("generic wrapped error leaked child-process diagnostic %q: %q", sensitive, wrapped)
		}
	}
}

func TestToolErrorUnknownToolNameIsNotReflected(t *testing.T) {
	err := &toolError{tool: "secret-host.example/private-user", code: -1, message: "opaque"}
	if got, want := err.Error(), "network tool operation failed"; got != want {
		t.Fatalf("Error()=%q want %q", got, want)
	}
}

func TestToolErrorPrivateDiagnosticsStillDriveClassification(t *testing.T) {
	err := &toolError{
		tool:    "sftp",
		code:    255,
		message: "Load key C:/Users/private-user/.ssh/customer-prod: incorrect passphrase supplied to decrypt private key",
	}
	if got := err.UserErrorKind(); got != "sftp_settings" {
		t.Fatalf("UserErrorKind()=%q want sftp_settings", got)
	}
	if strings.Contains(err.Error(), "private-user") || strings.Contains(err.Error(), "customer-prod") {
		t.Fatalf("public error leaked private diagnostic: %q", err.Error())
	}
}

func TestToolErrorPrivateDiagnosticsPreserveRetryClassification(t *testing.T) {
	retryable := &toolError{
		tool:    "sftp",
		code:    255,
		message: "Connection reset by peer at secret-host.example",
	}
	if !IsRetryable(retryable) {
		t.Fatal("transient SFTP diagnostic must remain retryable after public error redaction")
	}

	nonRetryable := &toolError{
		tool:    "sftp",
		code:    255,
		message: "Permission denied (publickey,password) for private-user@secret-host.example",
	}
	if IsRetryable(nonRetryable) {
		t.Fatal("authentication diagnostic must remain non-retryable after public error redaction")
	}
}
