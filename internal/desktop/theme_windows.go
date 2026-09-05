//go:build windows

package desktop

// Ghost FTP uses a restrained graphite/navy palette so the application remains
// comfortable during long transfer sessions while keeping status and primary
// actions visually distinct. Colors intentionally avoid pure black/white to
// reduce glare and preserve legibility on standard and high-DPI displays.
const (
	colorWindowR, colorWindowG, colorWindowB    = byte(9), byte(13), byte(19)
	colorListR, colorListG, colorListB          = byte(17), byte(24), byte(33)
	colorTextR, colorTextG, colorTextB          = byte(241), byte(245), byte(249)
	colorMutedR, colorMutedG, colorMutedB       = byte(148), byte(163), byte(184)
	colorAccentR, colorAccentG, colorAccentB    = byte(56), byte(189), byte(248)
	colorSuccessR, colorSuccessG, colorSuccessB = byte(74), byte(222), byte(128)
	colorWarnR, colorWarnG, colorWarnB          = byte(251), byte(191), byte(36)
)

func windowColor() uintptr  { return rgb(colorWindowR, colorWindowG, colorWindowB) }
func listColor() uintptr    { return rgb(colorListR, colorListG, colorListB) }
func textColor() uintptr    { return rgb(colorTextR, colorTextG, colorTextB) }
func mutedColor() uintptr   { return rgb(colorMutedR, colorMutedG, colorMutedB) }
func accentColor() uintptr  { return rgb(colorAccentR, colorAccentG, colorAccentB) }
func successColor() uintptr { return rgb(colorSuccessR, colorSuccessG, colorSuccessB) }
func warnColor() uintptr    { return rgb(colorWarnR, colorWarnG, colorWarnB) }
