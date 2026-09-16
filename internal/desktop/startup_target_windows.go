//go:build windows

package desktop

import "strconv"

func (a *app) applyStartupTarget() bool {
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
