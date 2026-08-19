package remote

import "testing"

func TestHostKeyConstraintAvoidsLegacyRSASHA1(t *testing.T) {
	if got := hostKeyConstraintForScannedKey("ssh-rsa"); got != "" {
		t.Fatalf("RSA key type must not force legacy ssh-rsa HostKeyAlgorithms, got %q", got)
	}
}

func TestHostKeyConstraintKeepsModernPinnedKeyTypes(t *testing.T) {
	for _, keyType := range []string{"ssh-ed25519", "ecdsa-sha2-nistp256", "ecdsa-sha2-nistp384", "ecdsa-sha2-nistp521"} {
		if got := hostKeyConstraintForScannedKey(keyType); got != keyType {
			t.Fatalf("key type %q changed to %q", keyType, got)
		}
	}
}
