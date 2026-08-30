package desktop

import "testing"

func TestValidateRawConnectionInputAcceptsCanonicalPort(t *testing.T) {
	port, err := validateRawConnectionInput("sftp", "example.test", "22", "deploy")
	if err != nil {
		t.Fatalf("canonical connection input rejected: %v", err)
	}
	if port != 22 {
		t.Fatalf("unexpected parsed port: %d", port)
	}
}

func TestValidateRawConnectionInputRejectsNonCanonicalPortText(t *testing.T) {
	for _, raw := range []string{" 22", "22 ", "22\r\n", "22\t"} {
		if _, err := validateRawConnectionInput("sftp", "example.test", raw, "deploy"); err == nil {
			t.Fatalf("non-canonical raw port %q unexpectedly accepted", raw)
		}
	}
}

func TestValidateRawConnectionInputRejectsUsernameControlsBeforeNormalization(t *testing.T) {
	for _, username := range []string{"deploy\r", "deploy\n", "deploy\x00root"} {
		if _, err := validateRawConnectionInput("sftp", "example.test", "22", username); err == nil {
			t.Fatalf("raw username control %q unexpectedly accepted", username)
		}
	}
}

func TestValidateRawConnectionInputDoesNotTrimUsername(t *testing.T) {
	username := " deploy account "
	port, err := validateRawConnectionInput("ftp", "example.test", "21", username)
	if err != nil {
		t.Fatalf("backend-compatible username unexpectedly rejected: %v", err)
	}
	if port != 21 {
		t.Fatalf("unexpected parsed port: %d", port)
	}
}

func TestValidateRawConnectionInputKeepsHostFailClosed(t *testing.T) {
	if _, err := validateRawConnectionInput("ftp", " example.test", "21", "deploy"); err == nil {
		t.Fatal("raw host whitespace unexpectedly accepted")
	}
}
