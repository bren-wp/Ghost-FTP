package remote

import "testing"

func TestFTPCommandUnsupportedUsesPrivateToolDiagnostic(t *testing.T) {
	err := &toolError{
		tool:    "curl",
		code:    22,
		message: "500 Unknown command MLSD on secret-host.example for private-user",
	}
	if !ftpCommandUnsupported(err) {
		t.Fatal("redacted curl error must still classify an unsupported MLSD command")
	}
	if got := err.Error(); got == err.message {
		t.Fatal("unsupported-command classification must not expose the raw curl diagnostic")
	}
}

func TestFTPCommandUnsupportedDoesNotMisclassifyPrivateAuthOrTransportDiagnostic(t *testing.T) {
	for _, err := range []*toolError{
		{tool: "curl", code: 67, message: "530 Login incorrect for private-user@secret-host.example"},
		{tool: "curl", code: 28, message: "Connection timed out to secret-host.example"},
		{tool: "curl", code: 7, message: "425 Can't open data connection"},
	} {
		if ftpCommandUnsupported(err) {
			t.Fatalf("private diagnostic was misclassified as unsupported command: kind=%q", err.UserErrorKind())
		}
	}
}
