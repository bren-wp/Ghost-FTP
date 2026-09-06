//go:build linux

package security

import (
	"bytes"
	"strings"
	"testing"
)

func TestLinuxProtectedSecretRoundTripAndForget(t *testing.T) {
	blob, err := ProtectString("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("ProtectString failed: %v", err)
	}
	if !strings.HasPrefix(blob, linuxSecretPrefix) {
		t.Fatalf("unexpected Linux secret envelope: %q", blob)
	}
	if strings.Contains(blob, "correct-horse-battery-staple") {
		t.Fatal("plaintext secret leaked into broker token")
	}

	plain, err := UnprotectBytes(blob)
	if err != nil {
		t.Fatalf("UnprotectBytes failed: %v", err)
	}
	if !bytes.Equal(plain, []byte("correct-horse-battery-staple")) {
		WipeBytes(plain)
		t.Fatal("broker returned the wrong secret")
	}
	WipeBytes(plain)

	ForgetProtectedSecret(blob)
	if _, err := UnprotectBytes(blob); err == nil {
		t.Fatal("forgotten broker secret was still available")
	}
}

func TestLinuxProtectedSecretRejectsMalformedEnvelope(t *testing.T) {
	if _, err := UnprotectBytes("linux-secret-v1:not-a-valid-token"); err == nil {
		t.Fatal("malformed broker envelope was accepted")
	}
}

func TestLinuxPersistentSecretStorageIsUnavailable(t *testing.T) {
	if PersistentSecretStorageAvailable() {
		t.Fatal("Linux must not claim OS-bound persistent secret storage")
	}
}
