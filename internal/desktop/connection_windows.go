//go:build windows

package desktop

import (
	"context"
	"strconv"
	"strings"
	"time"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
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

// beginConnectionTransition invalidates asynchronous callbacks created for an
// older session. The UI-thread-owned counter is checked again when background
// work dispatches results back to the window.
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
	for _, hwnd := range []uintptr{
		a.connect, a.disconnect, a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass,
		a.keyPath, a.chooseKey, a.passphrase, a.saveProfile, a.removeProfile, a.settingsBtn,
	} {
		setControlEnabled(hwnd, false)
	}
	a.setRemoteControls(false)
	a.updateActionControls()
}

func (a *app) setConnectionUI(connected bool) {
	a.connected = connected
	a.connectionBusy = false
	setControlEnabled(a.connect, !connected)
	setControlEnabled(a.disconnect, connected)
	for _, hwnd := range []uintptr{a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass} {
		setControlEnabled(hwnd, !connected)
	}
	if connected {
		for _, hwnd := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
			setControlEnabled(hwnd, false)
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
	host := strings.TrimSpace(getText(a.host))
	user := strings.TrimSpace(getText(a.user))
	password := getText(a.pass)
	port, err := strconv.Atoi(strings.TrimSpace(getText(a.port)))
	if err != nil || port < 1 || port > 65535 {
		platform.ErrorDialog("ByFTP", a.tr("connection.invalid_port"), a.tr("connection.invalid_port_body"))
		return
	}
	protocol := a.protocolValue()
	if err := security.ValidateConnection(protocol, host, user, port); err != nil {
		platform.ErrorDialog("ByFTP — "+a.tr("common.connect"), a.tr("connection.invalid_data"), a.userMessage(err, "connection.invalid_data_body"))
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
	// Secrets remain in disabled edit controls until the connection succeeds so
	// a transient network/port failure does not force the user to retype them.
	a.setConnectionBusy(true)
	a.setStatus(a.tr("connection.connecting", host))
	profileID := a.selectedProfileID
	ctx, cancel := a.connectionContext(75 * time.Second)
	a.goSafe(func() {
		defer cancel()
		result, err := a.engine.Connect(ctx, profileID, cfg, "", false)
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
				platform.ErrorDialog("ByFTP — "+a.tr("common.connect"), a.tr("connection.failed"), a.userMessage(err, "connection.failed_body"))
				return
			}
			if result.RequiresTrust {
				if !platform.ConfirmDialog(a.tr("sftp.security"), a.tr("sftp.new_key"), a.tr("sftp.trust_body", result.Fingerprint)) {
					a.engine.CancelPendingTrust()
					a.setConnectionUI(false)
					a.setStatus(a.tr("sftp.cancelled"))
					return
				}
				a.connectTrusted(profileID, cfg, result.Fingerprint, generation)
				return
			}
			a.onConnected(host)
		})
	})
}

func (a *app) connectTrusted(profileID string, cfg model.ConnectionConfig, fingerprint string, generation uint64) {
	if generation != a.connectionGeneration {
		return
	}
	// Quick-connect SFTP cannot depend only on short-lived pending-trust state.
	// Controls are locked during the attempt, so the entered secret can be read
	// again safely; saved profiles still resolve their bound saved credentials.
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
		_, err := a.engine.Connect(ctx, profileID, cfg, fingerprint, profileID != "")
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
				platform.ErrorDialog("ByFTP — SFTP", a.tr("connection.failed"), a.userMessage(err, "sftp.failed_body"))
				return
			}
			a.onConnected(cfg.Host)
		})
	})
}

func (a *app) onConnected(host string) {
	setText(a.pass, "")
	setText(a.passphrase, "")
	a.queuePaused = false
	a.setConnectionUI(true)
	a.setStatus(a.tr("connection.connected", host))
	remoteStart := "/"
	if a.protocolValue() == "sftp" {
		remoteStart = "."
	}
	localStart := ""
	if profile, ok := a.currentProfile(); ok && a.currentEndpointMatchesProfile(profile) {
		if strings.TrimSpace(profile.RemotePath) != "" {
			remoteStart = profile.RemotePath
		}
		localStart = profile.LocalPath
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
	path, err := a.engine.ChoosePrivateKey()
	if err != nil {
		platform.ErrorDialog("ByFTP", a.tr("key.choose_failed"), a.userMessage(err, "key.choose_failed_body"))
		return
	}
	if path != "" {
		setText(a.keyPath, path)
	}
}
