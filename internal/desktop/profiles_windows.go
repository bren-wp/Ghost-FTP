//go:build windows

package desktop

import (
	"strconv"
	"strings"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/profilebinding"
	"brendigo.com/byftp/internal/security"
)

func (a *app) currentEndpointMatchesProfile(profile model.PublicProfile) bool {
	port, err := strconv.Atoi(strings.TrimSpace(getText(a.port)))
	if err != nil {
		return false
	}
	return profilebinding.EndpointMatches(
		profile.Protocol, profile.Host, profile.Port,
		a.protocolValue(), strings.TrimSpace(getText(a.host)), port,
	)
}

func (a *app) resetProfileCredentialCues() {
	cue(a.pass, a.tr("cue.password"))
	cue(a.passphrase, a.tr("cue.passphrase"))
}

func (a *app) setProfileCredentialCues(profile model.PublicProfile) {
	if profile.HasPassword {
		cue(a.pass, a.tr("cue.saved_password"))
	} else {
		cue(a.pass, a.tr("cue.password"))
	}
	if profile.HasPassphrase && profile.PrivateKeyPath != "" {
		cue(a.passphrase, a.tr("cue.saved_passphrase"))
	} else {
		cue(a.passphrase, a.tr("cue.passphrase"))
	}
}

func (a *app) loadProfiles() {
	a.goSafe(func() {
		profiles, err := a.engine.Profiles()
		a.dispatch(func() {
			a.applyProfiles(profiles, err)
		})
	})
}

func (a *app) applyProfiles(profiles []model.PublicProfile, loadErr error) {
	if loadErr != nil {
		a.setStatus(a.userMessage(loadErr, "profile.load_failed"))
		return
	}
	a.profiles = profiles
	var selectedProfile model.PublicProfile
	found := false
	if a.selectedProfileID != "" {
		for _, profile := range profiles {
			if profile.ID == a.selectedProfileID {
				selectedProfile = profile
				found = true
				break
			}
		}
	}
	if !found {
		a.selectedProfileID = ""
		a.resetProfileCredentialCues()
	} else {
		a.setProfileCredentialCues(selectedProfile)
	}
	a.reloadProfileLabels()
	a.updateActionControls()
}

func (a *app) currentProfile() (model.PublicProfile, bool) {
	if a.selectedProfileID == "" {
		return model.PublicProfile{}, false
	}
	for _, profile := range a.profiles {
		if profile.ID == a.selectedProfileID {
			return profile, true
		}
	}
	return model.PublicProfile{}, false
}

func (a *app) selectProfile() {
	if a.connectionBusy || a.connected {
		return
	}
	idx := selectedComboIndex(a.profilesCombo)
	if idx <= 0 || idx > len(a.profiles) {
		a.selectedProfileID = ""
		setText(a.pass, "")
		setText(a.passphrase, "")
		a.resetProfileCredentialCues()
		a.updateActionControls()
		return
	}
	profile := a.profiles[idx-1]
	a.selectedProfileID = profile.ID
	a.setProtocolValue(profile.Protocol)
	setText(a.host, profile.Host)
	setText(a.port, strconv.Itoa(profile.Port))
	setText(a.user, profile.Username)
	setText(a.pass, "")
	setText(a.keyPath, profile.PrivateKeyPath)
	setText(a.passphrase, "")
	if profile.LocalPath != "" {
		a.refreshLocal(profile.LocalPath)
	}
	if profile.RemotePath != "" {
		setText(a.remotePath, profile.RemotePath)
	}
	a.setProfileCredentialCues(profile)
	a.updateActionControls()
}

func (a *app) saveCurrentProfile() {
	if a.connected || a.connectionBusy {
		platform.InfoDialog("ByFTP", a.tr("profile.edit_blocked_title"), a.tr("profile.edit_blocked_body"))
		return
	}
	existing, editing := a.currentProfile()
	defaultName := strings.TrimSpace(getText(a.host))
	if editing {
		defaultName = existing.Name
	}
	name, ok := a.promptText(a.tr("profile.dialog_title"), a.tr("profile.name_prompt"), defaultName)
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		platform.ErrorDialog(a.tr("profile.dialog_title"), a.tr("profile.name_missing_title"), a.tr("profile.name_missing_body"))
		return
	}
	protocol := a.protocolValue()
	host := strings.TrimSpace(getText(a.host))
	username := strings.TrimSpace(getText(a.user))
	port, err := strconv.Atoi(strings.TrimSpace(getText(a.port)))
	if err != nil || port < 1 || port > 65535 {
		platform.ErrorDialog(a.tr("profile.dialog_title"), a.tr("connection.invalid_port"), a.tr("connection.invalid_port_body"))
		return
	}
	if err := security.ValidateConnection(protocol, host, username, port); err != nil {
		platform.ErrorDialog(a.tr("profile.dialog_title"), a.tr("connection.invalid_data"), a.userMessage(err, "connection.invalid_data_body"))
		return
	}
	keyPath := strings.TrimSpace(getText(a.keyPath))
	password := getText(a.pass)
	passphrase := getText(a.passphrase)
	clearPassword := false
	clearPassphrase := false

	if editing {
		sameAccount := profilebinding.AccountMatches(
			existing.Protocol, existing.Host, existing.Port, existing.Username,
			protocol, host, port, username,
		)
		samePrivateKey := profilebinding.PrivateKeyMatches(
			existing.Protocol, existing.Host, existing.Port, existing.Username, existing.PrivateKeyPath,
			protocol, host, port, username, keyPath,
		)
		clearPassword = existing.HasPassword && !sameAccount
		clearPassphrase = existing.HasPassphrase && !samePrivateKey
	}

	if password != "" || passphrase != "" {
		if !platform.ConfirmDialog(a.tr("privacy.title"), a.tr("profile.store_credentials_title"), a.tr("profile.store_credentials_body")) {
			password = ""
			passphrase = ""
			clearPassword = true
			clearPassphrase = true
		} else {
			if password != "" {
				clearPassword = false
			}
			if passphrase != "" {
				clearPassphrase = false
			}
		}
	} else if editing {
		retainPassword := existing.HasPassword && !clearPassword
		retainPassphrase := existing.HasPassphrase && !clearPassphrase
		autoRemoved := clearPassword || clearPassphrase
		if retainPassword || retainPassphrase {
			message := a.tr("profile.retain_credentials_body")
			if autoRemoved {
				message = a.tr("profile.credentials_auto_removed_intro") + message
			}
			if !platform.ConfirmDialog(a.tr("privacy.title"), a.tr("profile.retain_credentials_title"), message) {
				if retainPassword {
					clearPassword = true
				}
				if retainPassphrase {
					clearPassphrase = true
				}
			}
		} else if autoRemoved {
			platform.InfoDialog(a.tr("profile.dialog_title"), a.tr("profile.old_credentials_title"), a.tr("profile.old_credentials_body"))
		}
	}

	payload := model.ProfileInput{
		ID:              a.selectedProfileID,
		Name:            name,
		Protocol:        protocol,
		Host:            host,
		Port:            port,
		Username:        username,
		Password:        password,
		ClearPassword:   clearPassword,
		PrivateKeyPath:  keyPath,
		Passphrase:      passphrase,
		ClearPassphrase: clearPassphrase,
		LocalPath:       a.localCurrent,
		RemotePath:      optionalRemotePath(getText(a.remotePath)),
	}
	setText(a.pass, "")
	setText(a.passphrase, "")
	password = ""
	passphrase = ""
	a.goSafe(func() {
		saved, err := a.engine.SaveProfile(payload)
		payload.Password = ""
		payload.Passphrase = ""
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog(a.tr("profile.dialog_title"), a.tr("profile.save_failed_title"), a.userMessage(err, "profile.save_failed_body"))
				return
			}
			a.selectedProfileID = saved.ID
			setText(a.pass, "")
			setText(a.passphrase, "")
			a.setProfileCredentialCues(saved)
			a.setStatus(a.tr("profile.saved", saved.Name))
			a.loadProfiles()
		})
	})
}

func (a *app) removeCurrentProfile() {
	if a.connectionBusy {
		return
	}
	profile, ok := a.currentProfile()
	if !ok {
		platform.InfoDialog("ByFTP", a.tr("profile.none_selected_title"), a.tr("profile.none_selected_body"))
		return
	}
	if a.connected {
		platform.InfoDialog("ByFTP", a.tr("profile.delete_connected_title"), a.tr("profile.delete_connected_body"))
		return
	}
	if !platform.ConfirmDialog(a.tr("profile.delete_title"), a.tr("profile.delete_question"), a.tr("profile.delete_body", profile.Name)) {
		return
	}
	id := profile.ID
	a.goSafe(func() {
		err := a.engine.RemoveProfile(id)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog(a.tr("profile.delete_title"), a.tr("profile.delete_failed_title"), a.userMessage(err, "profile.delete_failed_body"))
				return
			}
			a.selectedProfileID = ""
			a.resetProfileCredentialCues()
			a.loadProfiles()
			a.setStatus(a.tr("profile.deleted"))
			a.updateActionControls()
		})
	})
}
