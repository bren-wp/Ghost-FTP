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
// Version 1.0.1 uses a restrained light workspace inspired by the clarity of
// traditional two-pane FTP clients: neutral window chrome, white file panes,
// subtle separators, dark readable text and a focused blue selection/accent.
// The visual language remains Ghost FTP's own and does not copy third-party
// artwork, icons, fonts or branded assets.
//
// Canonical palette:
//
//	bg       #F3F4F6   surface   #FFFFFF   surface-2 #FAFBFC
//	line     #D7DCE3   text      #1F2937   muted     #667085
//	accent   #356AC3   accent-2  #2457A6   success   #177245
//	warning  #9A6700   danger    #B42318   selection #DCEBFF
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
	Window:       RGB{0xF3, 0xF4, 0xF6},
	Panel:        RGB{0xFF, 0xFF, 0xFF},
	List:         RGB{0xFA, 0xFB, 0xFC},
	Border:       RGB{0xD7, 0xDC, 0xE3},
	Text:         RGB{0x1F, 0x29, 0x37},
	Muted:        RGB{0x66, 0x70, 0x85},
	Accent:       RGB{0x35, 0x6A, 0xC3},
	AccentStrong: RGB{0x24, 0x57, 0xA6},
	Success:      RGB{0x17, 0x72, 0x45},
	Warn:         RGB{0x9A, 0x67, 0x00},
	Danger:       RGB{0xB4, 0x23, 0x18},
	Selection:    RGB{0xDC, 0xEB, 0xFF},
}
