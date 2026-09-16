//go:build !windows

package desktop

// ForwardStartupTarget is currently implemented only for the installed Windows
// desktop client, which owns the ghostftp: protocol registration in 0.0.6.
func ForwardStartupTarget(raw string) bool {
	return false
}
