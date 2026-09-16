//go:build !windows

package desktop

// StartStartupTargetReceiver is a no-op outside Windows because 0.0.6 registers
// the ghostftp: desktop protocol handler only in the installed Windows client.
func StartStartupTargetReceiver() func() {
	return func() {}
}

// ForwardStartupTarget is currently implemented only for the installed Windows
// desktop client, which owns the ghostftp: protocol registration in 0.0.6.
func ForwardStartupTarget(raw string) bool {
	return false
}
