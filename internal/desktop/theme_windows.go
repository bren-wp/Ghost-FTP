//go:build windows

package desktop

const (
	colorWindowR, colorWindowG, colorWindowB    = byte(14), byte(17), byte(22)
	colorListR, colorListG, colorListB          = byte(27), byte(33), byte(41)
	colorTextR, colorTextG, colorTextB          = byte(236), byte(241), byte(247)
	colorMutedR, colorMutedG, colorMutedB       = byte(154), byte(166), byte(180)
	colorAccentR, colorAccentG, colorAccentB    = byte(41), byte(182), byte(246)
	colorSuccessR, colorSuccessG, colorSuccessB = byte(76), byte(217), byte(140)
	colorWarnR, colorWarnG, colorWarnB          = byte(255), byte(183), byte(47)
)

func windowColor() uintptr  { return rgb(colorWindowR, colorWindowG, colorWindowB) }
func listColor() uintptr    { return rgb(colorListR, colorListG, colorListB) }
func textColor() uintptr    { return rgb(colorTextR, colorTextG, colorTextB) }
func mutedColor() uintptr   { return rgb(colorMutedR, colorMutedG, colorMutedB) }
func accentColor() uintptr  { return rgb(colorAccentR, colorAccentG, colorAccentB) }
func successColor() uintptr { return rgb(colorSuccessR, colorSuccessG, colorSuccessB) }
func warnColor() uintptr    { return rgb(colorWarnR, colorWarnG, colorWarnB) }
