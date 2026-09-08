package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestThemeForAppearanceUsesCanonicalPalettes(t *testing.T) {
	if got := themeForAppearance(model.AppearanceDark); got != darkTheme {
		t.Fatal("dark appearance did not resolve to the canonical dark palette")
	}
	if got := themeForAppearance(model.AppearanceLight); got != lightTheme {
		t.Fatal("light appearance did not resolve to the canonical light palette")
	}
	if got := themeForAppearance("unknown"); got != lightTheme {
		t.Fatal("unknown appearance did not fall back to the primary light palette")
	}
}

func TestInitialDesktopThemeIsClassicLight(t *testing.T) {
	if premiumTheme != lightTheme {
		t.Fatal("initial desktop palette is not Classic Light")
	}
}

func TestClassicLightThemeKeepsReadableContrastDirection(t *testing.T) {
	if lightTheme.Window == darkTheme.Window || lightTheme.Panel == darkTheme.Panel || lightTheme.Text == darkTheme.Text {
		t.Fatal("light and dark palettes are not visually distinct")
	}
	if lightTheme.Text.R >= lightTheme.Panel.R && lightTheme.Text.G >= lightTheme.Panel.G && lightTheme.Text.B >= lightTheme.Panel.B {
		t.Fatal("light theme text is not darker than its panel surface")
	}
	if lightTheme.Selection == lightTheme.List {
		t.Fatal("light theme selection is indistinguishable from list background")
	}
}

func TestClassicLightAvoidsPureWhitePrimarySurfaces(t *testing.T) {
	pureWhite := RGB{R: 0xFF, G: 0xFF, B: 0xFF}
	for name, surface := range map[string]RGB{
		"window": lightTheme.Window,
		"panel":  lightTheme.Panel,
		"list":   lightTheme.List,
	} {
		if surface == pureWhite {
			t.Fatalf("Classic Light %s surface regressed to pure white", name)
		}
	}
}

func TestDarkThemeKeepsLayerSeparation(t *testing.T) {
	if darkTheme.Window == darkTheme.Panel || darkTheme.Panel == darkTheme.List {
		t.Fatal("dark theme window/panel/list layers are not visually separated")
	}
	if darkTheme.Text == darkTheme.Muted {
		t.Fatal("dark theme primary and muted text are indistinguishable")
	}
}
