//go:build linux

package desktop

import (
	"strconv"
	"strings"
	"sync"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/profilebinding"
)

type linuxProfileStartState struct {
	initialized       bool
	selectedProfileID string
	accountKey        string
	inheritedRemote   string
	verifiedLocal     string
	verifiedRemote    string
}

var linuxProfileStartStates sync.Map

func linuxProfileStartStateFor(u *linuxDesktop) *linuxProfileStartState {
	if value, ok := linuxProfileStartStates.Load(u); ok {
		return value.(*linuxProfileStartState)
	}
	state := &linuxProfileStartState{}
	actual, _ := linuxProfileStartStates.LoadOrStore(u, state)
	return actual.(*linuxProfileStartState)
}

func (u *linuxDesktop) selectedLinuxProfile() (model.PublicProfile, bool) {
	if u == nil || u.selectedProfileID == "" {
		return model.PublicProfile{}, false
	}
	for index := range u.profiles {
		if u.profiles[index].ID == u.selectedProfileID {
			return u.profiles[index], true
		}
	}
	return model.PublicProfile{}, false
}

func linuxProtocolRemoteDefault(protocol string) string {
	if strings.EqualFold(strings.TrimSpace(protocol), "sftp") {
		return "."
	}
	return "/"
}

func (u *linuxDesktop) linuxProfileMatchesConnectionFields(profile model.PublicProfile) bool {
	port, err := strconv.Atoi(strings.TrimSpace(u.port))
	if err != nil {
		return false
	}
	return profilebinding.AccountMatches(
		profile.Protocol, profile.Host, profile.Port, profile.Username,
		u.protocol, u.host, port, u.username,
	)
}

func (u *linuxDesktop) linuxEditableAccountKey() string {
	if u == nil {
		return ""
	}
	portText := strings.TrimSpace(u.port)
	port, err := strconv.Atoi(portText)
	if err != nil {
		return "invalid\x00" + strings.ToLower(strings.TrimSpace(u.protocol)) + "\x00" + strings.ToLower(strings.TrimSpace(u.host)) + "\x00" + portText + "\x00" + u.username
	}
	return profilebinding.EndpointKey(u.protocol, u.host, port) + "\x00" + u.username
}

// enforceLinuxProfileStartDirectories runs on the UI goroutine before the
// bookmark header is painted. cycleProfile historically copied start paths
// directly into the editable path fields. This guard turns local copies into
// drafts: the previous verified local base is restored immediately and the
// configured local start is committed only by refreshLocal after LocalList
// succeeds.
//
// Remote starts are event-bound rather than repaint-bound. Selecting a profile
// installs that profile's saved/default remote start once. If the editable
// protocol/host/port/username later crosses an account boundary, the inherited
// remote start is reset only while it is still unchanged. A user-edited remote
// path is therefore treated as an explicit start for the new account and is
// never overwritten merely because the window repaints.
func (u *linuxDesktop) enforceLinuxProfileStartDirectories() {
	if u == nil {
		return
	}
	state := linuxProfileStartStateFor(u)
	currentAccountKey := u.linuxEditableAccountKey()
	if !state.initialized {
		state.initialized = true
		state.selectedProfileID = u.selectedProfileID
		state.accountKey = currentAccountKey
		state.verifiedLocal = u.localCurrent
		state.verifiedRemote = u.remoteCurrent
		if profile, ok := u.selectedLinuxProfile(); ok {
			state.inheritedRemote = strings.TrimSpace(profile.RemotePath)
			if state.inheritedRemote == "" {
				state.inheritedRemote = linuxProtocolRemoteDefault(u.protocol)
			}
		}
		return
	}

	profile, hasProfile := u.selectedLinuxProfile()
	if u.selectedProfileID != state.selectedProfileID {
		previousLocal := state.verifiedLocal
		state.selectedProfileID = u.selectedProfileID
		state.accountKey = currentAccountKey
		state.inheritedRemote = ""

		if !hasProfile {
			return
		}

		localTarget := strings.TrimSpace(profile.LocalPath)
		if localTarget != "" && u.localCurrent == profile.LocalPath {
			u.localCurrent = previousLocal
			u.selectedLocal = -1
			if !u.busy {
				u.refreshLocal(profile.LocalPath)
			}
		}

		state.inheritedRemote = strings.TrimSpace(profile.RemotePath)
		if state.inheritedRemote == "" {
			state.inheritedRemote = linuxProtocolRemoteDefault(u.protocol)
		}
		u.remoteCurrent = state.inheritedRemote
	}

	if hasProfile && !u.connected && currentAccountKey != state.accountKey {
		if u.remoteCurrent == state.inheritedRemote {
			u.remoteCurrent = linuxProtocolRemoteDefault(u.protocol)
			state.inheritedRemote = u.remoteCurrent
		} else {
			// The remote path no longer equals the profile-derived value, so it is
			// an explicit user edit for the new account and must survive repaint.
			state.inheritedRemote = ""
		}
		state.accountKey = currentAccountKey
	}

	// Only non-busy states can represent a completed listing. A failed local
	// refresh leaves localCurrent at the previously verified base, while a
	// successful one commits the canonical LocalList result before this point.
	if !u.busy {
		state.verifiedLocal = u.localCurrent
		if u.connected {
			state.verifiedRemote = u.remoteCurrent
		}
	}
}
