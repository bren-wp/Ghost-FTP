package remote

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func directSFTPConfig(fingerprint string) model.ConnectionConfig {
	return model.ConnectionConfig{
		Protocol:    "sftp",
		Host:        "example.test",
		Port:        22,
		Username:    "alice",
		Fingerprint: fingerprint,
	}
}

func TestResolveAcceptsCanonicalDirectSFTPFingerprint(t *testing.T) {
	m := &Manager{}
	fingerprint := "SHA256:" + strings.Repeat("A", 43)
	resolved, _, err := m.Resolve("", directSFTPConfig(fingerprint))
	if err != nil {
		t.Fatalf("canonical direct fingerprint rejected: %v", err)
	}
	if resolved.Config.Fingerprint != fingerprint {
		t.Fatalf("fingerprint changed during resolve: %q != %q", resolved.Config.Fingerprint, fingerprint)
	}
}

func TestResolveRejectsMalformedDirectSFTPFingerprint(t *testing.T) {
	for _, fingerprint := range []string{
		"SHA256:short",
		"SHA256:" + strings.Repeat("A", 43) + "=",
		"SHA256:" + strings.Repeat("A", 42) + "!",
	} {
		m := &Manager{}
		if _, _, err := m.Resolve("", directSFTPConfig(fingerprint)); err == nil {
			t.Fatalf("malformed direct fingerprint accepted: %q", fingerprint)
		}
	}
}
