//go:build linux

package platform

// SelectLanguageDialog is the Linux build stub. Ghost FTP Setup is a Windows
// application, but keeping this symbol on Linux preserves shared package tests
// without advertising support for any additional application platform.
func SelectLanguageDialog(_ string, _ string, options []string, defaultIndex int) (int, bool) {
	if len(options) == 0 {
		return 0, false
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}
	return defaultIndex, true
}
