package config

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestDefaultSettingsUseDarkAppearance(t *testing.T) {
	if got := DefaultSettings().Appearance; got != model.AppearanceDark {
		t.Fatalf("default appearance=%q, want %q", got, model.AppearanceDark)
	}
}

func TestNormalizeSettingsMigratesMissingAppearance(t *testing.T) {
	settings := DefaultSettings()
	settings.Appearance = ""
	if got := normalizeSettings(settings).Appearance; got != model.AppearanceDark {
		t.Fatalf("normalized missing appearance=%q, want %q", got, model.AppearanceDark)
	}
}

func TestNormalizeSettingsRepairsUnknownPersistedAppearance(t *testing.T) {
	settings := DefaultSettings()
	settings.Appearance = "unexpected-theme"
	if got := normalizeSettings(settings).Appearance; got != model.AppearanceDark {
		t.Fatalf("normalized unknown appearance=%q, want %q", got, model.AppearanceDark)
	}
}

func TestValidateSettingsAcceptsOnlyCanonicalAppearances(t *testing.T) {
	for _, appearance := range []string{model.AppearanceDark, model.AppearanceLight} {
		settings := DefaultSettings()
		settings.Appearance = appearance
		if err := validateSettings(settings); err != nil {
			t.Fatalf("appearance %q rejected: %v", appearance, err)
		}
	}

	settings := DefaultSettings()
	settings.Appearance = "white"
	if err := validateSettings(settings); err == nil {
		t.Fatal("non-canonical appearance was accepted")
	}
}
