package remote

import (
	"strings"
	"testing"
)

func TestAskpassEnvironmentRejectsCredentialWithoutTrustedHelper(t *testing.T) {
	s := &SFTP{passwordBlob: "protected-placeholder", exePath: ""}
	env, err := s.askpassEnvironment()
	if err == nil {
		t.Fatal("credential-bearing AskPass environment unexpectedly accepted without a trusted helper")
	}
	if env != nil {
		t.Fatalf("credential-bearing AskPass failure returned environment data: %#v", env)
	}
}

func TestAskpassEnvironmentWithoutCredentialsDoesNotRequireHelper(t *testing.T) {
	s := &SFTP{}
	env, err := s.askpassEnvironment()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range env {
		for _, prefix := range []string{
			"SSH_ASKPASS=",
			"SSH_ASKPASS_REQUIRE=",
			"GhostFTP_ASKPASS_TOKEN=",
			"GhostFTP_PASSWORD_BLOB=",
			"GhostFTP_PASSPHRASE_BLOB=",
		} {
			if strings.HasPrefix(item, prefix) {
				t.Fatalf("non-credential SFTP environment retained AskPass capability %q", item)
			}
		}
	}
}
