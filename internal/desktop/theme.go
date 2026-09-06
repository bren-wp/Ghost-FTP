package desktop

// PremiumTheme is the canonical cross-platform Ghost FTP desktop palette.
// The values intentionally mirror the maintained Ghost FTP Web design tokens
// so Windows, Linux and the web companion present one coherent product brand.
//
// Web source of truth (GhostFTP WEB/assets/css/app.css):
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
	Window:       RGB{R: 8, G: 10, B: 15},     // #080A0F -- --bg
	Panel:        RGB{R: 15, G: 19, B: 28},    // #0F131C -- --surface
	List:         RGB{R: 21, G: 26, B: 37},    // #151A25 -- --surface-2
	Border:       RGB{R: 37, G: 45, B: 60},    // #252D3C -- --line
	Text:         RGB{R: 244, G: 247, B: 255}, // #F4F7FF -- --text
	Muted:        RGB{R: 142, G: 153, B: 173}, // #8E99AD -- --muted
	Accent:       RGB{R: 82, G: 119, B: 245},  // #5277F5 -- --accent
	AccentStrong: RGB{R: 114, G: 147, B: 255}, // #7293FF -- --accent-2
	Success:      RGB{R: 74, G: 215, B: 155},  // #4AD79B -- --success
	Warn:         RGB{R: 242, G: 186, B: 85},  // #F2BA55 -- --warning
	Danger:       RGB{R: 255, G: 100, B: 118}, // #FF6476 -- --danger
	Selection:    RGB{R: 27, G: 35, B: 58},    // accent at ~10% over surface-2
}

// Shared desktop layout metrics are expressed in logical 96-DPI pixels. The
// Windows layer scales them for DPI; Linux X11/XWayland uses actual window
// pixels and keeps the same proportions. Radii follow the Web token contract.
const (
	premiumMinWidth    = 1080
	premiumMinHeight   = 700
	premiumStartWidth  = 1200
	premiumStartHeight = 780
	premiumOuterGap    = 18
	premiumPanelGap    = 12
	premiumRadius      = 16
)
