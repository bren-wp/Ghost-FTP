//go:build linux

package desktop

// The native Linux workspace uses Classic Light as its canonical 1.1 desktop
// appearance. Keeping this choice compile-time/local avoids mutable global theme
// state in the X11 render loop and therefore avoids introducing redraw races or
// additional appearance controls that do not yet have a complete Linux runtime
// switching contract.
func init() {
	premiumTheme = lightTheme
}
