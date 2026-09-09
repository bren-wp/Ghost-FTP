package remote

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestFTPCommandUnsupportedUsesTransientPrivateDiagnostic(t *testing.T) {
	raw := "500 Unknown command MLSD on secret-host.example for private-user"
	err := requireToolError(t, newToolError("curl", errors.New("tool process failed"), raw))
	if !ftpCommandUnsupported(err) {
		t.Fatal("redacted curl error must still classify an unsupported MLSD command")
	}
	if err.UserErrorKind() != "ftp_unsupported" {
		t.Fatalf("UserErrorKind()=%q want ftp_unsupported", err.UserErrorKind())
	}
	stored := fmt.Sprintf("%+v", *err)
	for _, sensitive := range []string{"500", "secret-host.example", "private-user"} {
		if strings.Contains(strings.ToLower(stored), strings.ToLower(sensitive)) {
			t.Fatalf("toolError retained unsupported-command diagnostic %q: %q", sensitive, stored)
		}
	}
}

func TestFTPCommandUnsupportedDoesNotMisclassifyPrivateAuthOrTransportDiagnostic(t *testing.T) {
	for _, raw := range []string{
		"530 Login incorrect for private-user@secret-host.example",
		"Connection timed out to secret-host.example",
		"425 Can't open data connection",
	} {
		err := requireToolError(t, newToolError("curl", errors.New("tool process failed"), raw))
		if ftpCommandUnsupported(err) {
			t.Fatalf("private diagnostic was misclassified as unsupported command: kind=%q", err.UserErrorKind())
		}
	}
}
