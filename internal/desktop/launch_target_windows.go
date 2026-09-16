//go:build windows

package desktop

func (a *app) applyInitialLaunchTarget() bool {
	target, ok := takeInitialLaunchTarget()
	if !ok {
		return false
	}

	sendMessageW.Call(a.protocol, cbSetCurSel, protocolIndex(target.Protocol), 0)
	setText(a.host, target.Host)
	setText(a.port, target.Port)
	setText(a.user, target.Username)
	setText(a.pass, "")
	setText(a.passphrase, "")
	setText(a.remotePath, target.Path)
	a.remoteCurrent = target.Path
	a.updateProtocolControls()
	a.updateActionControls()
	return true
}
