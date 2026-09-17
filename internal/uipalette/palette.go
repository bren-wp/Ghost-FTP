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
	OnAccent     RGB
	Success      RGB
	Warn         RGB
	Danger       RGB
	Selection    RGB
}

// Dark is the primary Ghost FTP product theme. The restrained blue-black
// surfaces mirror the native desktop references while Ghost Gold provides one
// recognizable action/focus color without turning the workspace into a neon UI.
var Dark = Theme{
	Window:       RGB{0x0B, 0x0F, 0x17},
	Panel:        RGB{0x12, 0x18, 0x24},
	List:         RGB{0x16, 0x1D, 0x2A},
	Border:       RGB{0x2C, 0x36, 0x48},
	Text:         RGB{0xF2, 0xF5, 0xFA},
	Muted:        RGB{0x97, 0xA3, 0xB8},
	Accent:       RGB{0xF6, 0xC4, 0x45},
	AccentStrong: RGB{0xFF, 0xD7, 0x68},
	OnAccent:     RGB{0x16, 0x13, 0x0B},
	Success:      RGB{0x4A, 0xD7, 0x9B},
	Warn:         RGB{0xF2, 0xBA, 0x55},
	Danger:       RGB{0xFF, 0x68, 0x78},
	Selection:    RGB{0x2B, 0x25, 0x15},
}

// Light is intentionally off-white rather than sterile white. It keeps the same
// Ghost Gold identity while using a darker amber tone so filled controls retain
// sufficient contrast with their light foreground.
var Light = Theme{
	Window:       RGB{0xEE, 0xF1, 0xF5},
	Panel:        RGB{0xF6, 0xF8, 0xFB},
	List:         RGB{0xFA, 0xFB, 0xFD},
	Border:       RGB{0xD6, 0xDC, 0xE5},
	Text:         RGB{0x17, 0x20, 0x33},
	Muted:        RGB{0x66, 0x70, 0x85},
	Accent:       RGB{0xA6, 0x65, 0x00},
	AccentStrong: RGB{0x87, 0x51, 0x00},
	OnAccent:     RGB{0xFA, 0xFB, 0xFD},
	Success:      RGB{0x1B, 0x7F, 0x4B},
	Warn:         RGB{0x9A, 0x67, 0x00},
	Danger:       RGB{0xB4, 0x23, 0x18},
	Selection:    RGB{0xF5, 0xE7, 0xC7},
}
