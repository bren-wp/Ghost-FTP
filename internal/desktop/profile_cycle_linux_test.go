//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func linuxProfileCycleTestDesktop() *linuxDesktop {
	return &linuxDesktop{
		profileIndex:  -1,
		protocol:      "ftps",
		host:          "before.example",
		port:          "21",
		username:      "before",
		localCurrent:  "/before-local",
		remoteCurrent: "/before-remote",
		profiles: []model.PublicProfile{{
			ID:         "profile-1",
			Name:       "Profile One",
			Protocol:   "sftp",
			Host:       "after.example",
			Port:       22,
			Username:   "after",
			LocalPath:  "/after-local",
			RemotePath: ".",
		}},
	}
}

func TestLinuxProfileCycleBlockedWhileConnected(t *testing.T) {
	u := linuxProfileCycleTestDesktop()
	u.connected = true
	u.cycleProfile()
	if u.profileIndex != -1 || u.selectedProfileID != "" || u.host != "before.example" || u.remoteCurrent != "/before-remote" {
		t.Fatalf("profile switching mutated an active connection draft: index=%d id=%q host=%q remote=%q", u.profileIndex, u.selectedProfileID, u.host, u.remoteCurrent)
	}
}

func TestLinuxProfileCycleBlockedWhileBusy(t *testing.T) {
	u := linuxProfileCycleTestDesktop()
	u.busy = true
	u.cycleProfile()
	if u.profileIndex != -1 || u.selectedProfileID != "" || u.host != "before.example" || u.localCurrent != "/before-local" {
		t.Fatalf("profile switching mutated busy state: index=%d id=%q host=%q local=%q", u.profileIndex, u.selectedProfileID, u.host, u.localCurrent)
	}
}

func TestLinuxProfileCycleLoadsProfileWhenIdleAndDisconnected(t *testing.T) {
	u := linuxProfileCycleTestDesktop()
	u.cycleProfile()
	if u.profileIndex != 0 || u.selectedProfileID != "profile-1" || u.protocol != "sftp" || u.host != "after.example" || u.port != "22" || u.username != "after" || u.localCurrent != "/after-local" || u.remoteCurrent != "." {
		t.Fatalf("idle profile cycle did not load expected draft: %+v", u)
	}
}
