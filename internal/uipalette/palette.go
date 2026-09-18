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

// Dark is the primary Ghost FTP product theme. It follows the approved
// reference UI: deep blue-black surfaces, cyan primary actions, electric-blue
// interaction states and a subtle violet-tinted selection layer.
var Dark = Theme{
	Window:       RGB{0x08, 0x0E, 0x14},
	Panel:        RGB{0x0D, 0x17, 0x24},
	List:         RGB{0x0A, 0x14, 0x21},
	Border:       RGB{0x1F, 0x3A, 0x53},
	Text:         RGB{0xE5, 0xE7, 0xEB},
	Muted:        RGB{0x8E, 0xA2, 0xB7},
	Accent:       RGB{0x00, 0xE5, 0xFF},
	AccentStrong: RGB{0x3B, 0x82, 0xF6},
	OnAccent:     RGB{0x05, 0x10, 0x18},
	Success:      RGB{0x22, 0xE6, 0xA5},
	Warn:         RGB{0xF2, 0xBA, 0x55},
	Danger:       RGB{0xFF, 0x5D, 0x7A},
	Selection:    RGB{0x16, 0x2A, 0x4A},
}

// Light keeps the same blue/cyan product identity while remaining comfortable
// for users who explicitly prefer a bright workspace.
var Light = Theme{
	Window:       RGB{0xEC, 0xF2, 0xF8},
	Panel:        RGB{0xF6, 0xF9, 0xFC},
	List:         RGB{0xFA, 0xFC, 0xFE},
	Border:       RGB{0xC7, 0xD4, 0xE2},
	Text:         RGB{0x12, 0x20, 0x33},
	Muted:        RGB{0x5F, 0x70, 0x86},
	Accent:       RGB{0x00, 0x82, 0xB8},
	AccentStrong: RGB{0x25, 0x63, 0xEB},
	OnAccent:     RGB{0xFA, 0xFC, 0xFE},
	Success:      RGB{0x0F, 0x8A, 0x62},
	Warn:         RGB{0x9A, 0x67, 0x00},
	Danger:       RGB{0xB4, 0x23, 0x18},
	Selection:    RGB{0xD8, 0xE8, 0xFF},
}
