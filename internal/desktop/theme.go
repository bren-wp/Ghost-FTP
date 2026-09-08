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

// Dark uses a restrained navy/charcoal surface rather than near-black blocks.
// This preserves contrast while reducing eye strain and keeps list/panel layers
// visually separable without relying on heavy borders.
var darkTheme = PremiumTheme{
	Window:       RGB{0x0B, 0x0F, 0x17},
	Panel:        RGB{0x12, 0x18, 0x24},
	List:         RGB{0x16, 0x1D, 0x2A},
	Border:       RGB{0x2C, 0x36, 0x48},
	Text:         RGB{0xF2, 0xF5, 0xFA},
	Muted:        RGB{0x97, 0xA3, 0xB8},
	Accent:       RGB{0x5B, 0x7C, 0xFA},
	AccentStrong: RGB{0x7A, 0x98, 0xFF},
	Success:      RGB{0x4A, 0xD7, 0x9B},
	Warn:         RGB{0xF2, 0xBA, 0x55},
	Danger:       RGB{0xFF, 0x68, 0x78},
	Selection:    RGB{0x20, 0x2F, 0x50},
}

// Light deliberately avoids pure white as the dominant application surface.
// The slightly cool neutral hierarchy keeps long file-management sessions less
// glaring while preserving native-control readability and clear selection.
var lightTheme = PremiumTheme{
	Window:       RGB{0xEE, 0xF1, 0xF5},
	Panel:        RGB{0xF6, 0xF8, 0xFB},
	List:         RGB{0xFA, 0xFB, 0xFD},
	Border:       RGB{0xC7, 0xCE, 0xD8},
	Text:         RGB{0x20, 0x25, 0x2B},
	Muted:        RGB{0x65, 0x70, 0x83},
	Accent:       RGB{0x3F, 0x63, 0xDD},
	AccentStrong: RGB{0x25, 0x4B, 0xC7},
	Success:      RGB{0x1B, 0x7F, 0x4B},
	Warn:         RGB{0x9A, 0x67, 0x00},
	Danger:       RGB{0xC6, 0x28, 0x28},
	Selection:    RGB{0xDC, 0xE8, 0xFF},
}

// Classic Light is the product-default desktop surface. Windows startup then
// applies the user's persisted appearance before controls are created, so an
// explicit Dark preference remains stable without a light-to-dark flash.
var premiumTheme = lightTheme

func themeForAppearance(appearance string) PremiumTheme {
	if appearance == model.AppearanceDark {
		return darkTheme
	}
	return lightTheme
}

func setActiveTheme(appearance string) {
	premiumTheme = themeForAppearance(appearance)
}

func isDarkAppearance(appearance string) bool {
	return appearance == model.AppearanceDark
}

func activeThemeIsDark() bool {
	return premiumTheme == darkTheme
}
