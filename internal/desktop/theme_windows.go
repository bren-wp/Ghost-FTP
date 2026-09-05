//go:build windows

package desktop

// Windows keeps native Win32 controls and converts the canonical shared RGB
// palette into COLORREF values. Keeping the palette in theme.go makes visual
// drift between Windows and Linux auditable rather than relying on duplicated
// platform constants.
func themeColor(c RGB) uintptr { return rgb(c.R, c.G, c.B) }

func windowColor() uintptr       { return themeColor(premiumTheme.Window) }
func panelColor() uintptr        { return themeColor(premiumTheme.Panel) }
func listColor() uintptr         { return themeColor(premiumTheme.List) }
func borderColor() uintptr       { return themeColor(premiumTheme.Border) }
func textColor() uintptr         { return themeColor(premiumTheme.Text) }
func mutedColor() uintptr        { return themeColor(premiumTheme.Muted) }
func accentColor() uintptr       { return themeColor(premiumTheme.Accent) }
func accentStrongColor() uintptr { return themeColor(premiumTheme.AccentStrong) }
func successColor() uintptr      { return themeColor(premiumTheme.Success) }
func warnColor() uintptr         { return themeColor(premiumTheme.Warn) }
func dangerColor() uintptr       { return themeColor(premiumTheme.Danger) }
func selectionColor() uintptr    { return themeColor(premiumTheme.Selection) }
