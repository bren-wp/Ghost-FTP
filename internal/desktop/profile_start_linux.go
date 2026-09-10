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

// enforceLinuxProfileStartDirectories runs on the UI goroutine before the
// bookmark header is painted. cycleProfile historically copied start paths
// directly into the editable path fields. This guard turns those copies into
// drafts: the previous verified local base is restored immediately and the
// configured local start is committed only by refreshLocal after LocalList
// succeeds. Remote starts remain usable as the post-connect listing target only
// while protocol/host/port/username still match the selected profile.
func (u *linuxDesktop) enforceLinuxProfileStartDirectories() {
	if u == nil {
		return
	}
	state := linuxProfileStartStateFor(u)
	if !state.initialized {
		state.initialized = true
		state.selectedProfileID = u.selectedProfileID
		state.verifiedLocal = u.localCurrent
		state.verifiedRemote = u.remoteCurrent
		return
	}

	profile, hasProfile := u.selectedLinuxProfile()
	if u.selectedProfileID != state.selectedProfileID {
		previousLocal := state.verifiedLocal
		previousRemote := state.verifiedRemote
		state.selectedProfileID = u.selectedProfileID

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

		remoteTarget := strings.TrimSpace(profile.RemotePath)
		if remoteTarget != "" && u.remoteCurrent == profile.RemotePath {
			u.remoteCurrent = previousRemote
		}
	}

	if hasProfile && !u.connected {
		if u.linuxProfileMatchesConnectionFields(profile) {
			if strings.TrimSpace(profile.RemotePath) != "" {
				u.remoteCurrent = profile.RemotePath
			} else {
				u.remoteCurrent = linuxProtocolRemoteDefault(u.protocol)
			}
		} else {
			// A selected profile can be edited before Connect. Once its account
			// identity no longer matches, never carry the selected account's server
			// directory into that different login.
			u.remoteCurrent = linuxProtocolRemoteDefault(u.protocol)
		}
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
