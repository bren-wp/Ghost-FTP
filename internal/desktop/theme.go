package desktop

// Canonical desktop geometry primitives. Linux uses these values directly and
// Windows applies its own DPI scaling around the same minimum workspace model.
// Keeping them outside a platform build tag prevents Win/Linux UI drift and
// ensures the Linux GUI can be compiled independently in CI.
const (
	premiumStartWidth  = 1280
	premiumStartHeight = 820
	premiumMinWidth    = 940
	premiumMinHeight   = 680
	premiumOuterGap    = 14
	premiumPanelGap    = 12
)

// PremiumTheme is the canonical cross-platform Ghost FTP desktop palette.
// Windows and Linux use the same restrained dark surfaces, high-contrast text,
// accessible state colors and selection treatment. The palette is owned by the
// desktop product and does not depend on a browser runtime, remote stylesheet,
// external font, theme service or third-party GUI framework.
//
// Canonical palette:
//
//	bg       #080A0F   surface   #0F131C   surface-2 #151A25
//	line     #252D3C   text      #F4F7FF   muted     #8E99AD
//	accent   #5277F5   accent-2  #7293FF   success   #4AD79B
//	warning  #F2BA55   danger    #FF6476
//
// The theme is data-only. It never loads fonts, images, telemetry, network
// resources or third-party theme engines at runtime.
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
	Window:       RGB{0x08, 0x0A, 0x0F},
	Panel:        RGB{0x0F, 0x13, 0x1C},
	List:         RGB{0x15, 0x1A, 0x25},
	Border:       RGB{0x25, 0x2D, 0x3C},
	Text:         RGB{0xF4, 0xF7, 0xFF},
	Muted:        RGB{0x8E, 0x99, 0xAD},
	Accent:       RGB{0x52, 0x77, 0xF5},
	AccentStrong: RGB{0x72, 0x93, 0xFF},
	Success:      RGB{0x4A, 0xD7, 0x9B},
	Warn:         RGB{0xF2, 0xBA, 0x55},
	Danger:       RGB{0xFF, 0x64, 0x76},
	Selection:    RGB{0x1D, 0x2A, 0x4A},
}
