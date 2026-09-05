//go:build windows

package desktop

// Reference-aligned Ghost FTP palette. The shell deliberately stays in deep
// navy rather than pure black, while controls use a slightly lighter surface
// and cool blue borders. Primary actions use the blue-violet accent visible in
// the canonical Ghost FTP desktop reference. All colors are local constants;
// no theme service, web asset or telemetry dependency is involved.
const (
	colorWindowR, colorWindowG, colorWindowB       = byte(5), byte(17), byte(29)
	colorPanelR, colorPanelG, colorPanelB          = byte(7), byte(25), byte(39)
	colorListR, colorListG, colorListB             = byte(8), byte(28), byte(43)
	colorBorderR, colorBorderG, colorBorderB       = byte(28), byte(62), byte(86)
	colorTextR, colorTextG, colorTextB             = byte(224), byte(237), byte(255)
	colorMutedR, colorMutedG, colorMutedB          = byte(126), byte(161), byte(201)
	colorAccentR, colorAccentG, colorAccentB       = byte(96), byte(126), byte(255)
	colorAccentStrongR, colorAccentStrongG, colorAccentStrongB = byte(110), byte(84), byte(255)
	colorSuccessR, colorSuccessG, colorSuccessB    = byte(57), byte(216), byte(166)
	colorWarnR, colorWarnG, colorWarnB             = byte(247), byte(190), byte(72)
)

func windowColor() uintptr       { return rgb(colorWindowR, colorWindowG, colorWindowB) }
func panelColor() uintptr        { return rgb(colorPanelR, colorPanelG, colorPanelB) }
func listColor() uintptr         { return rgb(colorListR, colorListG, colorListB) }
func borderColor() uintptr       { return rgb(colorBorderR, colorBorderG, colorBorderB) }
func textColor() uintptr         { return rgb(colorTextR, colorTextG, colorTextB) }
func mutedColor() uintptr        { return rgb(colorMutedR, colorMutedG, colorMutedB) }
func accentColor() uintptr       { return rgb(colorAccentR, colorAccentG, colorAccentB) }
func accentStrongColor() uintptr { return rgb(colorAccentStrongR, colorAccentStrongG, colorAccentStrongB) }
func successColor() uintptr      { return rgb(colorSuccessR, colorSuccessG, colorSuccessB) }
func warnColor() uintptr         { return rgb(colorWarnR, colorWarnG, colorWarnB) }
