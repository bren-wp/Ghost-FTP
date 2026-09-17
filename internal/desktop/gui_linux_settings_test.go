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

func TestLinuxSettingsPanelUsesRoomyWidthWithoutBreakingCompactWindows(t *testing.T) {
	if got := linuxSettingsPanelWidth(1280); got != 860 {
		t.Fatalf("1280 px panel width = %d, want 860", got)
	}
	if got := linuxSettingsPanelWidth(760); got != 660 {
		t.Fatalf("760 px panel width = %d, want 660", got)
	}
	if got := linuxSettingsPanelWidth(80); got != 80 {
		t.Fatalf("80 px defensive panel width = %d, want 80", got)
	}
}

func TestLinuxSettingsLabelAndChoiceSizingExpandsOnWidePanels(t *testing.T) {
	if got := linuxSettingsLabelLimit(860); got != 64 {
		t.Fatalf("wide label limit = %d, want 64", got)
	}
	if got := linuxSettingsLabelLimit(700); got != 48 {
		t.Fatalf("compact label limit = %d, want 48", got)
	}
	if got := linuxSettingsChoiceWidth(860); got != 280 {
		t.Fatalf("wide choice width = %d, want 280", got)
	}
	if got := linuxSettingsChoiceWidth(700); got != 230 {
		t.Fatalf("compact choice width = %d, want 230", got)
	}
}
