package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/uipalette"
)

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

// Keep the historical desktop names as aliases so Windows/Linux rendering and
// existing tests continue to express the desktop contract while the actual RGB
// source of truth lives in one dependency-neutral package shared with modals.
type PremiumTheme = uipalette.Theme
type RGB = uipalette.RGB

var darkTheme = uipalette.Dark
var lightTheme = uipalette.Light

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
