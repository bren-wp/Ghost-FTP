//go:build windows

package desktop

import (
	"context"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/profilebinding"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
	"strconv"
	"strings"
	"time"
	"unsafe"
)

func (a *app) cancelConnectionAttempt() {
	if a.connectionCancel != nil {
		a.connectionCancel()
		a.connectionCancel = nil
	}
}

func (a *app) cancelHealthCheck() {
	if a.healthCheckCancel != nil {
		a.healthCheckCancel()
		a.healthCheckCancel = nil
	}
	a.healthCheckRunning = false
}

// beginConnectionTransition invalidates every asynchronous callback that was
// created for an older session. The counter is UI-thread owned and is checked
// again when asynchronous work dispatches its result back to the window.
func (a *app) beginConnectionTransition() uint64 {
	a.connectionGeneration++
	a.cancelHealthCheck()
	return a.connectionGeneration
}

func (a *app) connectionContext(timeout time.Duration) (context.Context, context.CancelFunc) {
	a.cancelConnectionAttempt()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	a.connectionCancel = cancel
	return ctx, cancel
}

func (a *app) syncDefaultPort() {
	idx, _, _ := sendMessageW.Call(a.protocol, cbGetCurSel, 0, 0)
	setText(a.port, protocolAt(idx).Port)
}

func (a *app) protocolValue() string {
	idx, _, _ := sendMessageW.Call(a.protocol, cbGetCurSel, 0, 0)
	return protocolAt(idx).Value
}

func (a *app) setProtocolValue(protocol string) {
	sendMessageW.Call(a.protocol, cbSetCurSel, protocolIndex(protocol), 0)
	a.updateProtocolControls()
}

func (a *app) setConnectionBusy(busy bool) {
	a.connectionBusy = busy
	if !busy {
		a.setConnectionUI(a.connected)
		return
	}
	for _, h := range []uintptr{
		a.connect, a.disconnect, a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.chooseKey, a.passphrase, a.saveProfile, a.removeProfile, a.settingsBtn,
	} {
		setControlEnabled(h, false)
	}
	a.setRemoteControls(false)
	a.updateActionControls()
}

func (a *app) setConnectionUI(connected bool) {
	a.connected = connected
	a.connectionBusy = false
	setControlEnabled(a.connect, !connected)
	setControlEnabled(a.disconnect, connected)
	for _, h := range []uintptr{a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass} {
		setControlEnabled(h, !connected)
	}
	if connected {
		for _, h := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
			setControlEnabled(h, false)
		}
		setText(a.connectionBadge, a.tr("badge.connected"))
	} else {
		setText(a.connectionBadge, a.tr("badge.disconnected"))
		a.updateProtocolControls()
	}
	a.setRemoteControls(connected)
	a.updateActionControls()
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) connectNow() {
	if a.connectionBusy || a.connected {
		return
	}
	host := getText(a.host)
	user := getText(a.user)
	password := getText(a.pass)
	protocol := a.protocolValue()
	port, err := validateRawConnectionInput(protocol, host, getText(a.port), user)
	if err != nil {
		if err == errInvalidConnectionPort {
			platform.ErrorDialog("Ghost FTP", a.tr("connection.invalid_port"), a.tr("connection.invalid_port_body"))
		} else {
			platform.ErrorDialog("Ghost FTP", a.tr("connection.invalid_data"), a.userMessage(err, "connection.invalid_data_body"))
		}
		return
	}
	cfg := model.ConnectionConfig{
		Protocol:       protocol,
		Host:           host,
		Port:           port,
		Username:       user,
		Password:       password,
		PrivateKeyPath: strings.TrimSpace(getText(a.keyPath)),
		Passphrase:     getText(a.passphrase),
	}
	generation := a.beginConnectionTransition()
	// Entered secrets stay in the disabled edit controls until the connection
	// succeeds. This lets a user retry a network/port failure without retyping a
	// secret, while onConnected clears them immediately after confirmed login.
	a.setConnectionBusy(true)
	a.setStatus(a.tr("connection.connecting", host))
	profileID := a.selectedProfileID
	ctx, cancel := a.connectionContext(75 * time.Second)
	a.goSafe(func() {
		defer cancel()
		r, err := a.engine.Connect(ctx, profileID, cfg, "", false)
		cfg.Password = ""
		cfg.Passphrase = ""
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			a.connectionCancel = nil
			if err != nil {
				a.setConnectionUI(false)
				a.setStatus(a.userMessage(err, "connection.failed_status"))
				platform.ErrorDialog("Ghost FTP", a.tr("connection.failed"), a.userMessage(err, "connection.failed_body"))
				return
			}
			if r.RequiresTrust {
				if !platform.ConfirmDialog(a.tr("sftp.security"), a.tr("sftp.new_key"), a.tr("sftp.trust_body", r.Fingerprint)) {
					a.engine.CancelPendingTrust()
					a.setConnectionUI(false)
					a.setStatus(a.tr("sftp.cancelled"))
					return
				}
				a.connectTrusted(profileID, cfg, r.Fingerprint, generation)
				return
			}
			a.onConnected(host, r.Diagnostics)
		})
	})
}

func (a *app) connectTrusted(profileID string, cfg model.ConnectionConfig, fingerprint string, generation uint64) {
	if generation != a.connectionGeneration {
		return
	}
	// A quick SFTP connection cannot rely only on the short-lived pending-trust
	// cache. Controls are locked during the attempt, so the just-entered secret
	// can safely be read again. For a saved profile an empty field still lets
	// Resolve use the correctly bound stored credential.
	if cfg.Password == "" {
		cfg.Password = getText(a.pass)
	}
	if cfg.Passphrase == "" {
		cfg.Passphrase = getText(a.passphrase)
	}
	a.setStatus(a.tr("sftp.verifying"))
	ctx, cancel := a.connectionContext(75 * time.Second)
	a.goSafe(func() {
		defer cancel()
		r, err := a.engine.Connect(ctx, profileID, cfg, fingerprint, profileID != "")
		cfg.Password = ""
		cfg.Passphrase = ""
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			a.connectionCancel = nil
			if err != nil {
				a.setConnectionUI(false)
				a.setStatus(a.userMessage(err, "sftp.failed_status"))
				platform.ErrorDialog("Ghost FTP — SFTP", a.tr("connection.failed"), a.userMessage(err, "sftp.failed_body"))
				return
			}
			a.onConnected(cfg.Host, r.Diagnostics)
		})
	})
}

func (a *app) currentEndpointMatchesProfile(p model.PublicProfile) bool {
	port, err := strconv.Atoi(getText(a.port))
	if err != nil {
		return false
	}
	return profilebinding.EndpointMatches(
		p.Protocol, p.Host, p.Port,
		a.protocolValue(), getText(a.host), port,
	)
}

func (a *app) connectionDiagnosticStatus(host string, diagnostics remote.ConnectionDiagnostics) string {
	// Diagnostics remain available to the connection engine, but the concise
	// workspace status uses the current UI catalog instead of mixing one locale
	// into another. Detailed transport diagnostics belong in explicit error or
	// security flows, not an unlocalized status sentence.
	_ = diagnostics
	return a.tr("connection.connected", host)
}

func (a *app) onConnected(host string, diagnostics remote.ConnectionDiagnostics) {
	setText(a.pass, "")
	setText(a.passphrase, "")
	a.queuePaused = false
	a.setConnectionUI(true)
	a.setStatus(a.connectionDiagnosticStatus(host, diagnostics))
	remoteStart := "/"
	if a.protocolValue() == "sftp" {
		remoteStart = "."
	}
	localStart := ""
	if p, ok := a.currentProfile(); ok && a.currentEndpointMatchesProfile(p) {
		if strings.TrimSpace(p.RemotePath) != "" {
			remoteStart = p.RemotePath
		}
		localStart = p.LocalPath
	}
	if localStart != "" {
		a.refreshLocal(localStart)
	}
	a.refreshRemote(remoteStart)
	a.refreshTransfers()
}

func (a *app) finishDisconnected(status string) {
	if a.remoteNavCancel != nil {
		a.remoteNavCancel()
		a.remoteNavCancel = nil
	}
	a.remoteNavSeq++
	a.queuePaused = true
	a.setConnectionUI(false)
	clearList(a.remoteList)
	a.remoteItems = nil
	a.updateActionControls()
	if strings.TrimSpace(status) != "" {
		a.setStatus(status)
	}
}

func (a *app) disconnectNow() {
	if a.connectionBusy || !a.connected {
		return
	}
	if a.hasActiveTransfers() {
		if !platform.ConfirmDialog(a.tr("disconnect.title"), a.tr("disconnect.question"), a.tr("disconnect.body")) {
			return
		}
	}
	generation := a.beginConnectionTransition()
	a.setConnectionBusy(true)
	a.setStatus(a.tr("disconnect.progress"))
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := a.engine.Disconnect(ctx)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil {
				a.finishDisconnected(a.userMessage(err, "disconnect.warning"))
			} else {
				a.finishDisconnected(a.tr("disconnect.done"))
			}
		})
	})
}

func (a *app) choosePrivateKey() {
	p, err := a.engine.ChoosePrivateKey()
	if err != nil {
		platform.ErrorDialog("Ghost FTP", a.tr("key.choose_failed"), a.userMessage(err, "key.choose_failed_body"))
		return
	}
	if p != "" {
		setText(a.keyPath, p)
	}
}

func (a *app) resetProfileCredentialCues() {
	cue(a.pass, a.tr("cue.password"))
	cue(a.passphrase, a.tr("cue.passphrase"))
}

func (a *app) setProfileCredentialCues(p model.PublicProfile) {
	if p.HasPassword {
		cue(a.pass, a.tr("cue.saved_password"))
	} else {
		cue(a.pass, a.tr("cue.password"))
	}
	if p.HasPassphrase && p.PrivateKeyPath != "" {
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
	const cbResetContent = 0x014B
	sendMessageW.Call(a.profilesCombo, cbResetContent, 0, 0)
	sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(a.tr("profile.quick")))))
	for _, p := range profiles {
		label := p.Name + " — " + p.Host
		sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
	}
	selected := 0
	var selectedProfile model.PublicProfile
	if a.selectedProfileID != "" {
		for i, p := range profiles {
			if p.ID == a.selectedProfileID {
				selected = i + 1
				selectedProfile = p
				break
			}
		}
	}
	if selected == 0 {
		a.selectedProfileID = ""
		a.resetProfileCredentialCues()
	} else {
		a.setProfileCredentialCues(selectedProfile)
	}
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(selected), 0)
	a.updateActionControls()
}

func (a *app) currentProfile() (model.PublicProfile, bool) {
	if a.selectedProfileID == "" {
		return model.PublicProfile{}, false
	}
	for _, p := range a.profiles {
		if p.ID == a.selectedProfileID {
			return p, true
		}
	}
	return model.PublicProfile{}, false
}

func (a *app) selectProfile() {
	if a.connectionBusy || a.connected {
		return
	}
	idx, _, _ := sendMessageW.Call(a.profilesCombo, cbGetCurSel, 0, 0)
	if idx == 0 || int(idx) > len(a.profiles) {
		a.selectedProfileID = ""
		setText(a.pass, "")
		setText(a.passphrase, "")
		a.resetProfileCredentialCues()
		a.updateActionControls()
		return
	}
	p := a.profiles[int(idx)-1]
	a.selectedProfileID = p.ID
	a.setProtocolValue(p.Protocol)
	setText(a.host, p.Host)
	setText(a.port, strconv.Itoa(p.Port))
	setText(a.user, p.Username)
	setText(a.pass, "")
	setText(a.keyPath, p.PrivateKeyPath)
	setText(a.passphrase, "")
	if p.LocalPath != "" {
		a.refreshLocal(p.LocalPath)
	}
	if p.RemotePath != "" {
		setText(a.remotePath, p.RemotePath)
	}
	a.setProfileCredentialCues(p)
	a.updateActionControls()
}

func (a *app) saveCurrentProfile() {
	if a.connected || a.connectionBusy {
		platform.InfoDialog("Ghost FTP", "Profile changes are unavailable during a connection", "Wait for the connection attempt to finish or disconnect before saving profile changes.")
		return
	}
	existing, editing := a.currentProfile()
	defaultName := strings.TrimSpace(getText(a.host))
	if editing {
		defaultName = existing.Name
	}
	name, ok := platform.PromptDialog("Ghost FTP — profile", "Profile name:", defaultName)
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	if name == "" {
		platform.ErrorDialog("Ghost FTP — profile", "Profile name is required", "Enter a name that identifies this connection.")
		return
	}
	protocol := a.protocolValue()
	host := getText(a.host)
	username := getText(a.user)
	port, err := validateRawConnectionInput(protocol, host, getText(a.port), username)
	if err != nil {
		if err == errInvalidConnectionPort {
			platform.ErrorDialog("Ghost FTP — profile", a.tr("connection.invalid_port"), a.tr("connection.invalid_port_body"))
		} else {
			platform.ErrorDialog("Ghost FTP — profile", a.tr("connection.invalid_data"), a.userMessage(err, "connection.invalid_data_body"))
		}
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
		if !platform.ConfirmDialog(
			"Ghost FTP — privacy",
			"Save credentials on this computer?",
			"Yes = store the entered credentials in the Windows-protected Ghost FTP profile store.\nNo = save the profile without a stored password or private-key passphrase.",
		) {
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
			message := "This profile already contains stored credentials that still belong to the same identity.\n\nYes = retain them.\nNo = remove them from the Ghost FTP profile store."
			if autoRemoved {
				message = "Credentials that no longer belong to the changed server, account or private key will be removed automatically.\n\n" + message
			}
			if !platform.ConfirmDialog("Ghost FTP — privacy", "Retain stored credentials?", message) {
				if retainPassword {
					clearPassword = true
				}
				if retainPassphrase {
					clearPassphrase = true
				}
			}
		} else if autoRemoved {
			platform.InfoDialog(
				"Ghost FTP — profile security",
				"Old credentials will not be transferred",
				"The server, port, username or private key changed. For safety, old stored credentials are removed from this profile. Enter them again if you want to store credentials for the new identity.",
			)
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
				platform.ErrorDialog("Ghost FTP — profile", a.tr("settings.save_failed"), a.userMessage(err, "settings.save_failed_body"))
				return
			}
			a.selectedProfileID = saved.ID
			setText(a.pass, "")
			setText(a.passphrase, "")
			a.setProfileCredentialCues(saved)
			a.setStatus(a.tr("profile.save") + ": " + saved.Name)
			a.loadProfiles()
		})
	})
}

func (a *app) removeCurrentProfile() {
	if a.connectionBusy {
		return
	}
	p, ok := a.currentProfile()
	if !ok {
		platform.InfoDialog("Ghost FTP", a.tr("profile.delete"), a.tr("profile.load_failed"))
		return
	}
	if a.connected {
		platform.InfoDialog("Ghost FTP", a.tr("profile.delete"), a.tr("disconnect.question"))
		return
	}
	if !platform.ConfirmDialog("Ghost FTP — profiles", a.tr("profile.delete"), p.Name) {
		return
	}
	id := p.ID
	a.goSafe(func() {
		err := a.engine.RemoveProfile(id)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("Ghost FTP — profiles", a.tr("profile.delete"), a.userMessage(err, "error.generic"))
				return
			}
			a.selectedProfileID = ""
			a.resetProfileCredentialCues()
			a.loadProfiles()
			a.setStatus(a.tr("profile.delete"))
			a.updateActionControls()
		})
	})
}
