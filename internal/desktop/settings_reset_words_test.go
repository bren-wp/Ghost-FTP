package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestSettingsResetLabelsCoverAllLanguages(t *testing.T) {
	for _, language := range i18n.Languages() {
		if got := settingsResetLabel(language.Code); got == "" {
			t.Fatalf("empty restore-defaults label for %s", language.Code)
		}
	}
	if got := settingsResetLabel("unknown"); got != settingsResetLabels[i18n.DefaultLanguage] {
		t.Fatalf("fallback label = %q, want English", got)
	}
}
