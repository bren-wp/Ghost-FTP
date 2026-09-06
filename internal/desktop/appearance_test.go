package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestAppearanceLabelsCoverEverySupportedLanguage(t *testing.T) {
	for _, language := range i18n.Languages() {
		words := appearanceText(language.Code)
		if strings.TrimSpace(words.Title) == "" || strings.TrimSpace(words.Dark) == "" || strings.TrimSpace(words.Light) == "" || strings.TrimSpace(words.Hint) == "" {
			t.Fatalf("appearance labels are incomplete for %s: %#v", language.Code, words)
		}
	}
}

func TestApplyAppearanceSelectionUsesOnlyCanonicalValues(t *testing.T) {
	settings := model.Settings{}
	applyAppearanceSelection(&settings, 1)
	if settings.Appearance != model.AppearanceLight {
		t.Fatalf("light selection=%q", settings.Appearance)
	}
	applyAppearanceSelection(&settings, 0)
	if settings.Appearance != model.AppearanceDark {
		t.Fatalf("dark selection=%q", settings.Appearance)
	}
	applyAppearanceSelection(&settings, 99)
	if settings.Appearance != model.AppearanceLight {
		t.Fatalf("unknown selection did not fall back to primary light: %q", settings.Appearance)
	}
}

func TestAppearanceIndexFallsBackToLight(t *testing.T) {
	if appearanceIndex(model.AppearanceLight) != 1 {
		t.Fatal("light appearance index is incorrect")
	}
	if appearanceIndex(model.AppearanceDark) != 0 {
		t.Fatal("dark appearance index is incorrect")
	}
	if appearanceIndex("unexpected") != 1 {
		t.Fatal("default appearance index is not light")
	}
}
