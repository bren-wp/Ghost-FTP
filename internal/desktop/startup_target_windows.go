//go:build windows

package desktop

import "strconv"

func (a *app) applyStartupTarget() bool {
	if a == nil || a.protocol == 0 || a.host == 0 || a.port == 0 || a.user == 0 || a.pass == 0 || a.remotePath == 0 {
		return false
	}
	if a.connected || a.connectionBusy {
		return false
	}
	target, ok := takeStartupTarget()
	if !ok {
		return false
	}

	sendMessageW.Call(a.protocol, cbSetCurSel, protocolIndex(target.Protocol), 0)
	setText(a.host, target.Host)
	setText(a.port, strconv.Itoa(startupTargetPort(target)))
	setText(a.user, target.Username)
	setText(a.pass, "")
	setText(a.passphrase, "")
	setText(a.remotePath, target.Path)
	a.remoteCurrent = target.Path
	a.updateProtocolControls()
	return true
}
