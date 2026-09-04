package remote

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestConnectionIdentityMatchesEquivalentEndpointForms(t *testing.T) {
	base := model.ConnectionConfig{
		Protocol:    "sftp",
		Host:        "Example.TEST.",
		Port:        22,
		Username:    "alice",
		Fingerprint: "SHA256:test",
	}
	equivalent := base
	equivalent.Protocol = " SFTP "
	equivalent.Host = "example.test"
	if got, want := connectionIdentity(equivalent), connectionIdentity(base); got != want {
		t.Fatalf("equivalent DNS endpoint changed connection identity: %q != %q", got, want)
	}

	ipv6A := base
	ipv6A.Host = "[2001:db8::1]"
	ipv6B := base
	ipv6B.Host = "2001:db8::1"
	if got, want := connectionIdentity(ipv6A), connectionIdentity(ipv6B); got != want {
		t.Fatalf("equivalent IPv6 endpoint changed connection identity: %q != %q", got, want)
	}
}

func TestConnectionIdentityStillSeparatesSecurityBoundaries(t *testing.T) {
	base := model.ConnectionConfig{
		Protocol:    "sftp",
		Host:        "example.test",
		Port:        22,
		Username:    "alice",
		Fingerprint: "SHA256:first",
	}
	baseID := connectionIdentity(base)

	changedUsername := base
	changedUsername.Username = "Alice"
	if connectionIdentity(changedUsername) == baseID {
		t.Fatal("username change crossed connection identity boundary")
	}

	changedFingerprint := base
	changedFingerprint.Fingerprint = "SHA256:second"
	if connectionIdentity(changedFingerprint) == baseID {
		t.Fatal("host-key fingerprint change crossed connection identity boundary")
	}

	changedPort := base
	changedPort.Port = 2222
	if connectionIdentity(changedPort) == baseID {
		t.Fatal("port change crossed connection identity boundary")
	}
}
