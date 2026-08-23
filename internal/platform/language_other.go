//go:build !windows

package platform

// SelectLanguageDialog is a non-Windows build stub. The graphical installer is
// distributed for Windows; keeping this symbol available preserves cross-platform
// package builds and tests.
func SelectLanguageDialog(_ string, _ string, options []string, defaultIndex int) (int, bool) {
	if len(options) == 0 {
		return 0, false
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}
	return defaultIndex, true
}
