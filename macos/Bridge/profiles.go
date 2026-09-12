package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

var profileBridgeState struct {
	mu       sync.RWMutex
	profiles []model.PublicProfile
}

func profileEngine() (*api.Engine, error) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		return nil, errors.New("engine is not initialized")
	}
	return bridgeState.engine, nil
}

func setProfileBridgeError(err error, fallback string) {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	setBridgeError(err, fallback)
}

func refreshProfilesSnapshot(engine *api.Engine) error {
	profiles, err := engine.Profiles()
	if err != nil {
		return err
	}
	profileBridgeState.mu.Lock()
	profileBridgeState.profiles = append(profileBridgeState.profiles[:0], profiles...)
	profileBridgeState.mu.Unlock()
	return nil
}

func profileAt(index C.int) (model.PublicProfile, bool) {
	profileBridgeState.mu.RLock()
	defer profileBridgeState.mu.RUnlock()
	i := int(index)
	if i < 0 || i >= len(profileBridgeState.profiles) {
		return model.PublicProfile{}, false
	}
	return profileBridgeState.profiles[i], true
}

//export GhostFTPRefreshProfiles
func GhostFTPRefreshProfiles() C.int {
	engine, err := profileEngine()
	if err != nil {
		setProfileBridgeError(err, "Ghost FTP could not read saved profiles.")
		return 0
	}
	if err := refreshProfilesSnapshot(engine); err != nil {
		setProfileBridgeError(err, "Ghost FTP could not unlock saved profiles.")
		return 0
	}
	setProfileBridgeError(nil, "")
	return 1
}

//export GhostFTPProfileCount
func GhostFTPProfileCount() C.int {
	profileBridgeState.mu.RLock()
	defer profileBridgeState.mu.RUnlock()
	return C.int(len(profileBridgeState.profiles))
}

//export GhostFTPProfileID
func GhostFTPProfileID(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.ID)
}

//export GhostFTPProfileName
func GhostFTPProfileName(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.Name)
}

//export GhostFTPProfileProtocol
func GhostFTPProfileProtocol(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.Protocol)
}

//export GhostFTPProfileHost
func GhostFTPProfileHost(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.Host)
}

//export GhostFTPProfilePort
func GhostFTPProfilePort(index C.int) C.int {
	profile, ok := profileAt(index)
	if !ok { return 0 }
	return C.int(profile.Port)
}

//export GhostFTPProfileUsername
func GhostFTPProfileUsername(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.Username)
}

//export GhostFTPProfileHasPassword
func GhostFTPProfileHasPassword(index C.int) C.int {
	profile, ok := profileAt(index)
	if ok && profile.HasPassword { return 1 }
	return 0
}

//export GhostFTPProfilePrivateKeyPath
func GhostFTPProfilePrivateKeyPath(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.PrivateKeyPath)
}

//export GhostFTPProfileHasPassphrase
func GhostFTPProfileHasPassphrase(index C.int) C.int {
	profile, ok := profileAt(index)
	if ok && profile.HasPassphrase { return 1 }
	return 0
}

//export GhostFTPProfileRemotePath
func GhostFTPProfileRemotePath(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.RemotePath)
}

//export GhostFTPProfileLocalPath
func GhostFTPProfileLocalPath(index C.int) *C.char {
	profile, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(profile.LocalPath)
}

// GhostFTPSaveProfile accepts typed fields only. clearPassword/clearPassphrase
// explicitly model credential consent; an empty secret with a false clear flag
// retains an existing saved credential after shared identity checks pass.
//export GhostFTPSaveProfile
func GhostFTPSaveProfile(id, name, protocol, host *C.char, port C.int, username, password *C.char, clearPassword C.int, privateKeyPath, passphrase *C.char, clearPassphrase C.int, remotePath, localPath *C.char) C.int {
	engine, err := profileEngine()
	if err != nil {
		setProfileBridgeError(err, "Ghost FTP is not ready.")
		return 0
	}
	input := model.ProfileInput{
		ID:              strings.TrimSpace(goString(id)),
		Name:            strings.TrimSpace(goString(name)),
		Protocol:        strings.ToLower(strings.TrimSpace(goString(protocol))),
		Host:            strings.TrimSpace(goString(host)),
		Port:            int(port),
		Username:        goString(username),
		Password:        goString(password),
		ClearPassword:   clearPassword != 0,
		PrivateKeyPath:  strings.TrimSpace(goString(privateKeyPath)),
		Passphrase:      goString(passphrase),
		ClearPassphrase: clearPassphrase != 0,
		RemotePath:      strings.TrimSpace(goString(remotePath)),
		LocalPath:       strings.TrimSpace(goString(localPath)),
	}
	if _, err := engine.SaveProfile(input); err != nil {
		setProfileBridgeError(err, "Ghost FTP could not save this profile.")
		return 0
	}
	if err := refreshProfilesSnapshot(engine); err != nil {
		setProfileBridgeError(err, "The profile was saved but the profile list could not be refreshed.")
		return 0
	}
	setProfileBridgeError(nil, "")
	return 1
}

//export GhostFTPRemoveProfile
func GhostFTPRemoveProfile(id *C.char) C.int {
	engine, err := profileEngine()
	if err != nil {
		setProfileBridgeError(err, "Ghost FTP is not ready.")
		return 0
	}
	profileID := strings.TrimSpace(goString(id))
	if profileID == "" {
		setProfileBridgeError(errors.New("profile id is empty"), "Select a saved profile first.")
		return 0
	}
	if err := engine.RemoveProfile(profileID); err != nil {
		setProfileBridgeError(err, "Ghost FTP could not remove this profile.")
		return 0
	}
	if err := refreshProfilesSnapshot(engine); err != nil {
		setProfileBridgeError(err, "The profile was removed but the profile list could not be refreshed.")
		return 0
	}
	setProfileBridgeError(nil, "")
	return 1
}

func publicProfileByID(engine *api.Engine, id string) (model.PublicProfile, error) {
	profiles, err := engine.Profiles()
	if err != nil { return model.PublicProfile{}, err }
	for _, profile := range profiles {
		if profile.ID == id { return profile, nil }
	}
	return model.PublicProfile{}, errors.New("saved profile was not found")
}

// GhostFTPConnectProfile returns the same 0/1/2 state contract as Quick
// Connect. Saved password/passphrase values are resolved inside Engine.Connect;
// only PublicProfile metadata crosses this bridge.
//export GhostFTPConnectProfile
func GhostFTPConnectProfile(profileID, trustFingerprint *C.char, rememberFingerprint C.int) C.int {
	engine, err := profileEngine()
	if err != nil {
		setProfileBridgeError(err, "Ghost FTP is not ready.")
		return 0
	}
	id := strings.TrimSpace(goString(profileID))
	profile, err := publicProfileByID(engine, id)
	if err != nil {
		setProfileBridgeError(err, "The selected saved profile is no longer available.")
		return 0
	}
	cfg := model.ConnectionConfig{
		Protocol:       profile.Protocol,
		Host:           profile.Host,
		Port:           profile.Port,
		Username:       profile.Username,
		PrivateKeyPath: profile.PrivateKeyPath,
		Fingerprint:    profile.Fingerprint,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := engine.Connect(ctx, id, cfg, strings.TrimSpace(goString(trustFingerprint)), rememberFingerprint != 0)
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if err != nil {
		bridgeState.pendingFingerprint = ""
		setBridgeError(err, "Connection failed. Check the saved profile and try again.")
		return 0
	}
	bridgeState.lastError = ""
	if result.RequiresTrust {
		bridgeState.pendingFingerprint = result.Fingerprint
		return 2
	}
	bridgeState.pendingFingerprint = ""
	bridgeState.remotePath = ""
	bridgeState.remoteItems = nil
	if result.Connected { return 1 }
	setBridgeError(errors.New("connection was not established"), "Connection failed. Please try again.")
	return 0
}

//export GhostFTPClearProfilesSnapshot
func GhostFTPClearProfilesSnapshot() {
	profileBridgeState.mu.Lock()
	profileBridgeState.profiles = nil
	profileBridgeState.mu.Unlock()
}
