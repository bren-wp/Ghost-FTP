//go:build linux

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestNextLinuxAppearanceTogglesCanonicalModes(t *testing.T) {
	if got := nextLinuxAppearance(model.AppearanceLight); got != model.AppearanceDark {
		t.Fatalf("light toggled to %q, want %q", got, model.AppearanceDark)
	}
	if got := nextLinuxAppearance(model.AppearanceDark); got != model.AppearanceLight {
		t.Fatalf("dark toggled to %q, want %q", got, model.AppearanceLight)
	}
	if got := nextLinuxAppearance("invalid"); got != model.AppearanceDark {
		t.Fatalf("invalid appearance toggled to %q, want conservative dark selection", got)
	}
}

func TestLinuxAppearanceUsesSharedDesktopThemes(t *testing.T) {
	previous := premiumTheme
	defer func() { premiumTheme = previous }()

	setActiveTheme(model.AppearanceDark)
	if premiumTheme != darkTheme || !activeThemeIsDark() {
		t.Fatal("Linux dark appearance did not activate the shared desktop dark theme")
	}

	setActiveTheme(model.AppearanceLight)
	if premiumTheme != lightTheme || activeThemeIsDark() {
		t.Fatal("Linux light appearance did not activate the shared desktop light theme")
	}
}
