package desktop

// PremiumTheme is the canonical cross-platform Ghost FTP desktop palette.
// Windows and Linux presentation layers consume these exact RGB values so the
// maintained product identity does not drift between operating systems.
//
// The theme is intentionally data-only. It does not load fonts, images,
// telemetry, network resources or third-party theme engines at runtime.
type PremiumTheme struct {
	Window       RGB
	Panel        RGB
	List         RGB
	Border       RGB
	Text         RGB
	Muted        RGB
	Accent       RGB
	AccentStrong RGB
	Success      RGB
	Warn         RGB
	Danger       RGB
	Selection    RGB
}

type RGB struct {
	R byte
	G byte
	B byte
}

var premiumTheme = PremiumTheme{
	Window:       RGB{R: 5, G: 17, B: 29},
	Panel:        RGB{R: 7, G: 25, B: 39},
	List:         RGB{R: 8, G: 28, B: 43},
	Border:       RGB{R: 28, G: 62, B: 86},
	Text:         RGB{R: 224, G: 237, B: 255},
	Muted:        RGB{R: 126, G: 161, B: 201},
	Accent:       RGB{R: 96, G: 126, B: 255},
	AccentStrong: RGB{R: 110, G: 84, B: 255},
	Success:      RGB{R: 57, G: 216, B: 166},
	Warn:         RGB{R: 247, G: 190, B: 72},
	Danger:       RGB{R: 244, G: 103, B: 122},
	Selection:    RGB{R: 18, G: 49, B: 76},
}

// Shared desktop layout metrics are expressed in logical 96-DPI pixels. The
// Windows layer scales them for DPI; Linux X11/XWayland uses the window's
// actual pixel dimensions and the same proportions.
const (
	premiumMinWidth   = 1080
	premiumMinHeight  = 700
	premiumStartWidth = 1200
	premiumStartHeight = 780
	premiumOuterGap   = 18
	premiumPanelGap   = 12
	premiumRadius     = 8
)
