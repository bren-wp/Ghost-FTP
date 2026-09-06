package desktop

import "github.com/bren-wp/Ghost-FTP/internal/model"

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
// Both supported appearances are local data-only palettes. They never load
// fonts, images, telemetry, network resources or third-party theme engines at
// runtime.
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

var darkTheme = PremiumTheme{
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

// lightTheme intentionally follows the restrained, information-dense visual
// language of traditional desktop file-transfer clients while retaining Ghost
// FTP's own identity. It does not copy third-party branding or assets.
var lightTheme = PremiumTheme{
	Window:       RGB{0xF3, 0xF4, 0xF6},
	Panel:        RGB{0xFF, 0xFF, 0xFF},
	List:         RGB{0xFF, 0xFF, 0xFF},
	Border:       RGB{0xC8, 0xCD, 0xD6},
	Text:         RGB{0x1F, 0x23, 0x28},
	Muted:        RGB{0x66, 0x70, 0x85},
	Accent:       RGB{0x3F, 0x63, 0xDD},
	AccentStrong: RGB{0x25, 0x4B, 0xC7},
	Success:      RGB{0x1B, 0x7F, 0x4B},
	Warn:         RGB{0x9A, 0x67, 0x00},
	Danger:       RGB{0xC6, 0x28, 0x28},
	Selection:    RGB{0xDC, 0xE8, 0xFF},
}

var premiumTheme = darkTheme

func themeForAppearance(appearance string) PremiumTheme {
	if appearance == model.AppearanceLight {
		return lightTheme
	}
	return darkTheme
}

func setActiveTheme(appearance string) {
	premiumTheme = themeForAppearance(appearance)
}

func isDarkAppearance(appearance string) bool {
	return appearance != model.AppearanceLight
}
