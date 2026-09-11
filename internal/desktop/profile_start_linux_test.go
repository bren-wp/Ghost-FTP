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

func linuxProfileStartTestDesktop() (*linuxDesktop, model.PublicProfile) {
	profile := model.PublicProfile{
		ID:         "profile-1",
		Protocol:   "ftps",
		Host:       "ftp.example.com",
		Port:       21,
		Username:   "alice",
		RemotePath: "/saved-home",
	}
	u := &linuxDesktop{
		protocol:          profile.Protocol,
		host:              profile.Host,
		port:              "21",
		username:          profile.Username,
		profiles:          []model.PublicProfile{profile},
		selectedProfileID: profile.ID,
		remoteCurrent:     profile.RemotePath,
	}
	return u, profile
}

func installLinuxProfileStartTestState(t *testing.T, u *linuxDesktop, state *linuxProfileStartState) {
	t.Helper()
	linuxProfileStartStates.Store(u, state)
	t.Cleanup(func() { linuxProfileStartStates.Delete(u) })
}

func TestLinuxProfileRemoteStartDoesNotOverwriteManualEditOnRepaint(t *testing.T) {
	u, _ := linuxProfileStartTestDesktop()
	state := &linuxProfileStartState{
		initialized:       true,
		selectedProfileID: u.selectedProfileID,
		accountKey:        u.linuxEditableAccountKey(),
		inheritedRemote:   "/saved-home",
	}
	installLinuxProfileStartTestState(t, u, state)

	u.remoteCurrent = "/manually-edited"
	u.enforceLinuxProfileStartDirectories()

	if u.remoteCurrent != "/manually-edited" {
		t.Fatalf("repaint overwrote explicit remote path: got %q", u.remoteCurrent)
	}
}

func TestLinuxProfileRemoteStartResetsInheritedPathOnAccountChange(t *testing.T) {
	u, _ := linuxProfileStartTestDesktop()
	oldAccountKey := u.linuxEditableAccountKey()
	state := &linuxProfileStartState{
		initialized:       true,
		selectedProfileID: u.selectedProfileID,
		accountKey:        oldAccountKey,
		inheritedRemote:   "/saved-home",
	}
	installLinuxProfileStartTestState(t, u, state)

	u.username = "bob"
	u.enforceLinuxProfileStartDirectories()

	if u.remoteCurrent != "/" {
		t.Fatalf("inherited remote start crossed account boundary: got %q, want /", u.remoteCurrent)
	}
}

func TestLinuxProfileRemoteStartResetsNavigatedOldAccountPathOnAccountChange(t *testing.T) {
	u, _ := linuxProfileStartTestDesktop()
	oldAccountKey := u.linuxEditableAccountKey()
	state := &linuxProfileStartState{
		initialized:       true,
		selectedProfileID: u.selectedProfileID,
		accountKey:        oldAccountKey,
		inheritedRemote:   "/saved-home",
	}
	installLinuxProfileStartTestState(t, u, state)

	u.remoteCurrent = "/navigated-on-old-account"
	u.username = "bob"
	u.enforceLinuxProfileStartDirectories()

	if u.remoteCurrent != "/" {
		t.Fatalf("navigated old-account path crossed account boundary: got %q, want /", u.remoteCurrent)
	}
}

func TestLinuxProfileExplicitRemoteStartSurvivesRepaintAfterAccountChange(t *testing.T) {
	u, _ := linuxProfileStartTestDesktop()
	oldAccountKey := u.linuxEditableAccountKey()
	state := &linuxProfileStartState{
		initialized:       true,
		selectedProfileID: u.selectedProfileID,
		accountKey:        oldAccountKey,
		inheritedRemote:   "/saved-home",
	}
	installLinuxProfileStartTestState(t, u, state)

	u.username = "bob"
	u.enforceLinuxProfileStartDirectories()
	if u.remoteCurrent != "/" {
		t.Fatalf("account change did not reset remote start before explicit edit: got %q", u.remoteCurrent)
	}

	u.remoteCurrent = "/new-account-home"
	u.enforceLinuxProfileStartDirectories()
	if u.remoteCurrent != "/new-account-home" {
		t.Fatalf("explicit new-account remote start was overwritten on repaint: got %q", u.remoteCurrent)
	}
}
