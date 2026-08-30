package security

import (
	"strings"
	"testing"
)

func TestValidateSFTPFingerprintAcceptsCanonicalDigest(t *testing.T) {
	fingerprint := "SHA256:" + strings.Repeat("A", 43)
	if err := ValidateSFTPFingerprint(fingerprint); err != nil {
		t.Fatalf("canonical fingerprint rejected: %v", err)
	}
}

func TestValidateSFTPFingerprintRejectsMalformedValues(t *testing.T) {
	for _, fingerprint := range []string{
		"",
		"SHA256:",
		"SHA256:short",
		"SHA256:" + strings.Repeat("A", 42),
		"SHA256:" + strings.Repeat("A", 44),
		"SHA256:" + strings.Repeat("A", 43) + "=",
		"SHA256:" + strings.Repeat("A", 42) + "!",
		"SHA256:" + strings.Repeat("A", 42) + "\t",
		" SHA256:" + strings.Repeat("A", 43),
	} {
		if err := ValidateSFTPFingerprint(fingerprint); err == nil {
			t.Fatalf("malformed fingerprint accepted: %q", fingerprint)
		}
	}
}
