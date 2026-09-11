//go:build linux

package desktop

// Classic Light remains the safe fallback before the engine-backed settings are
// available. newLinuxDesktop applies the persisted appearance before the first
// frame is rendered, and the Linux settings overlay can switch the same shared
// Light/Dark palette at runtime. The X11 event/render loop owns those mutations,
// so there is no concurrent theme writer.
func init() {
	premiumTheme = lightTheme
}
