package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

var profileSnapshot []model.PublicProfile

func refreshProfilesLocked() error {
	if bridgeState.engine == nil {
		return errors.New("engine is not initialized")
	}
	items, err := bridgeState.engine.Profiles()
	if err != nil {
		return err
	}
	profileSnapshot = append(profileSnapshot[:0], items...)
	return nil
}

func profileAt(index C.int) (model.PublicProfile, bool) {
	i := int(index)
	if i < 0 || i >= len(profileSnapshot) {
		return model.PublicProfile{}, false
	}
	return profileSnapshot[i], true
}

func profileByID(id string) (model.PublicProfile, bool) {
	id = strings.TrimSpace(id)
	for _, p := range profileSnapshot {
		if p.ID == id {
			return p, true
		}
	}
	return model.PublicProfile{}, false
}

//export GhostFTPRefreshProfiles
func GhostFTPRefreshProfiles() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if err := refreshProfilesLocked(); err != nil {
		setBridgeError(err, "Ghost FTP could not load saved sites.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPProfileCount
func GhostFTPProfileCount() C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	return C.int(len(profileSnapshot))
}

//export GhostFTPProfileID
func GhostFTPProfileID(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.ID)
}

//export GhostFTPProfileName
func GhostFTPProfileName(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.Name)
}

//export GhostFTPProfileProtocol
func GhostFTPProfileProtocol(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.Protocol)
}

//export GhostFTPProfileHost
func GhostFTPProfileHost(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.Host)
}

//export GhostFTPProfilePort
func GhostFTPProfilePort(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return 0 }
	return C.int(p.Port)
}

//export GhostFTPProfileUsername
func GhostFTPProfileUsername(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.Username)
}

//export GhostFTPProfileHasPassword
func GhostFTPProfileHasPassword(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if ok && p.HasPassword { return 1 }
	return 0
}

//export GhostFTPProfilePrivateKeyPath
func GhostFTPProfilePrivateKeyPath(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.PrivateKeyPath)
}

//export GhostFTPProfileHasPassphrase
func GhostFTPProfileHasPassphrase(index C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if ok && p.HasPassphrase { return 1 }
	return 0
}

//export GhostFTPProfileFingerprint
func GhostFTPProfileFingerprint(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.Fingerprint)
}

//export GhostFTPProfileRemotePath
func GhostFTPProfileRemotePath(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.RemotePath)
}

//export GhostFTPProfileLocalPath
func GhostFTPProfileLocalPath(index C.int) *C.char {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	p, ok := profileAt(index)
	if !ok { return C.CString("") }
	return C.CString(p.LocalPath)
}

// GhostFTPSaveProfile never accepts or returns protected secret blobs. Plaintext
// credential fields exist only for this typed call and are either persisted via
// the platform profile protector or discarded according to saveCredentials.
//export GhostFTPSaveProfile
func GhostFTPSaveProfile(idValue, nameValue, protocolValue, hostValue *C.char, portValue C.int, usernameValue, passwordValue *C.char, saveCredentials C.int, privateKeyValue, passphraseValue, fingerprintValue, remotePathValue, localPathValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	in := model.ProfileInput{
		ID:             strings.TrimSpace(goString(idValue)),
		Name:           strings.TrimSpace(goString(nameValue)),
		Protocol:       strings.ToLower(strings.TrimSpace(goString(protocolValue))),
		Host:           strings.TrimSpace(goString(hostValue)),
		Port:           int(portValue),
		Username:       goString(usernameValue),
		PrivateKeyPath: strings.TrimSpace(goString(privateKeyValue)),
		Fingerprint:    strings.TrimSpace(goString(fingerprintValue)),
		RemotePath:     strings.TrimSpace(goString(remotePathValue)),
		LocalPath:      strings.TrimSpace(goString(localPathValue)),
	}
	if saveCredentials != 0 {
		in.Password = goString(passwordValue)
		in.Passphrase = goString(passphraseValue)
	} else {
		in.ClearPassword = true
		in.ClearPassphrase = true
	}
	if _, err := bridgeState.engine.SaveProfile(in); err != nil {
		setBridgeError(err, "Ghost FTP could not save this site profile.")
		return 0
	}
	if err := refreshProfilesLocked(); err != nil {
		setBridgeError(err, "The profile was saved but the site list could not be refreshed.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

//export GhostFTPRemoveProfile
func GhostFTPRemoveProfile(idValue *C.char) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	id := strings.TrimSpace(goString(idValue))
	if id == "" {
		setBridgeError(errors.New("profile id is empty"), "Select a saved site first.")
		return 0
	}
	if err := bridgeState.engine.RemoveProfile(id); err != nil {
		setBridgeError(err, "Ghost FTP could not remove this site profile.")
		return 0
	}
	if err := refreshProfilesLocked(); err != nil {
		setBridgeError(err, "The profile was removed but the site list could not be refreshed.")
		return 0
	}
	bridgeState.lastError = ""
	return 1
}

// GhostFTPConnectProfile returns the same state codes as GhostFTPConnect:
// 1 connected, 2 SFTP host-key trust required, 0 failure.
//export GhostFTPConnectProfile
func GhostFTPConnectProfile(idValue, passwordValue, passphraseValue, trustFingerprint *C.char, rememberFingerprint C.int) C.int {
	bridgeState.mu.Lock()
	defer bridgeState.mu.Unlock()
	if bridgeState.engine == nil {
		setBridgeError(errors.New("engine is not initialized"), "Ghost FTP is not ready.")
		return 0
	}
	id := strings.TrimSpace(goString(idValue))
	if id == "" {
		setBridgeError(errors.New("profile id is empty"), "Select a saved site first.")
		return 0
	}
	if len(profileSnapshot) == 0 {
		if err := refreshProfilesLocked(); err != nil {
			setBridgeError(err, "Ghost FTP could not load saved sites.")
			return 0
		}
	}
	p, ok := profileByID(id)
	if !ok {
		setBridgeError(errors.New("profile is no longer available"), "Refresh Site Manager and try again.")
		return 0
	}
	cfg := model.ConnectionConfig{
		Protocol:       p.Protocol,
		Host:           p.Host,
		Port:           p.Port,
		Username:       p.Username,
		Password:       goString(passwordValue),
		PrivateKeyPath: p.PrivateKeyPath,
		Passphrase:     goString(passphraseValue),
		Fingerprint:    p.Fingerprint,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	result, err := bridgeState.engine.Connect(ctx, p.ID, cfg, strings.TrimSpace(goString(trustFingerprint)), rememberFingerprint != 0)
	if err != nil {
		bridgeState.pendingFingerprint = ""
		setBridgeError(err, "Connection failed. Check the saved site and try again.")
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
