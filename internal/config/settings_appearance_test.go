package config

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestDefaultSettingsUseLightAppearance(t *testing.T) {
	if got := DefaultSettings().Appearance; got != model.AppearanceLight {
		t.Fatalf("default appearance=%q, want %q", got, model.AppearanceLight)
	}
}

func TestNormalizeSettingsMigratesMissingAppearanceToLight(t *testing.T) {
	settings := DefaultSettings()
	settings.Appearance = ""
	if got := normalizeSettings(settings).Appearance; got != model.AppearanceLight {
		t.Fatalf("normalized missing appearance=%q, want %q", got, model.AppearanceLight)
	}
}

func TestNormalizeSettingsRepairsUnknownPersistedAppearanceToLight(t *testing.T) {
	settings := DefaultSettings()
	settings.Appearance = "unexpected-theme"
	if got := normalizeSettings(settings).Appearance; got != model.AppearanceLight {
		t.Fatalf("normalized unknown appearance=%q, want %q", got, model.AppearanceLight)
	}
}

func TestNormalizeSettingsPreservesExplicitDarkAppearance(t *testing.T) {
	settings := DefaultSettings()
	settings.Appearance = model.AppearanceDark
	if got := normalizeSettings(settings).Appearance; got != model.AppearanceDark {
		t.Fatalf("normalized explicit dark appearance=%q, want %q", got, model.AppearanceDark)
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
