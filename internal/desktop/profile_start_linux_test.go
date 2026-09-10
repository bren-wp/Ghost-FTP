//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestLinuxProtocolRemoteDefaultMatchesTransportSemantics(t *testing.T) {
	for _, test := range []struct {
		protocol string
		want     string
	}{
		{protocol: "ftp", want: "/"},
		{protocol: "ftps", want: "/"},
		{protocol: " SFTP ", want: "."},
	} {
		if got := linuxProtocolRemoteDefault(test.protocol); got != test.want {
			t.Fatalf("linuxProtocolRemoteDefault(%q) = %q, want %q", test.protocol, got, test.want)
		}
	}
}

func TestLinuxProfileConnectionMatchIncludesExactUsername(t *testing.T) {
	profile := model.PublicProfile{
		Protocol: "SFTP",
		Host:     "Example.COM.",
		Port:     22,
		Username: "alice",
	}
	u := &linuxDesktop{
		protocol: "sftp",
		host:     "example.com",
		port:     "22",
		username: "alice",
	}
	if !u.linuxProfileMatchesConnectionFields(profile) {
		t.Fatal("canonical same-account fields did not match selected profile")
	}

	u.username = "bob"
	if u.linuxProfileMatchesConnectionFields(profile) {
		t.Fatal("profile remote start crossed username/account boundary")
	}
}

func TestLinuxProfileConnectionMatchRejectsInvalidEditablePort(t *testing.T) {
	profile := model.PublicProfile{
		Protocol: "ftps",
		Host:     "ftp.example.com",
		Port:     21,
		Username: "alice",
	}
	u := &linuxDesktop{
		protocol: "ftps",
		host:     "ftp.example.com",
		port:     "not-a-port",
		username: "alice",
	}
	if u.linuxProfileMatchesConnectionFields(profile) {
		t.Fatal("invalid editable port was accepted as profile identity")
	}
}
