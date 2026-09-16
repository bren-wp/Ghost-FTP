//go:build windows

package desktop

import "strconv"

func (a *app) applyStartupTarget() bool {
	if a == nil || a.profilesCombo == 0 || a.protocol == 0 || a.host == 0 || a.port == 0 || a.user == 0 || a.pass == 0 || a.keyPath == 0 || a.passphrase == 0 || a.remotePath == 0 {
		return false
	}
	if a.connected || a.connectionBusy {
		return false
	}
	target, ok := takeStartupTarget()
	if !ok {
		return false
	}

	// Browser handoff is always an independent quick connection. Clear every
	// saved-profile credential source before applying endpoint data so a later
	// manual Connect cannot silently reload a password/passphrase or inherit a
	// private-key path that was never part of the handoff.
	a.selectedProfileID = ""
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, 0, 0)
	a.resetProfileCredentialCues()
	setText(a.pass, "")
	setText(a.keyPath, "")
	setText(a.passphrase, "")

	sendMessageW.Call(a.protocol, cbSetCurSel, protocolIndex(target.Protocol), 0)
	setText(a.host, target.Host)
	setText(a.port, strconv.Itoa(startupTargetPort(target)))
	setText(a.user, target.Username)
	setText(a.remotePath, target.Path)
	a.remoteCurrent = target.Path
	a.updateProtocolControls()
	return true
}
