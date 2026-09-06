//go:build linux

package platform

// HandleIntegratedUninstall is intentionally a no-op on Linux. Linux package
// managers own package removal; the Windows --uninstall mode is never exposed.
func HandleIntegratedUninstall(args []string) (handled bool, exitCode int) {
	return false, 0
}
