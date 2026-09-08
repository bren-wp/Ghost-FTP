package uipalette

// RGB is one local sRGB color in the Ghost FTP desktop palette.
type RGB struct {
	R byte
	G byte
	B byte
}

// Theme is the canonical cross-platform Ghost FTP desktop palette. Keeping
// these values in a dependency-neutral package prevents the desktop workspace
// and application-owned platform dialogs from drifting into subtly different
// Light/Dark products.
type Theme struct {
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

// Dark uses a restrained navy/charcoal hierarchy rather than near-black blocks.
var Dark = Theme{
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
var Light = Theme{
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
