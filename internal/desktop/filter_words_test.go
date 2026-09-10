package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestFileFilterWordsCoverEverySupportedLanguage(t *testing.T) {
	languages := i18n.Languages()
	if len(fileFilterTranslations) != len(languages) {
		t.Fatalf("filter translations=%d languages=%d", len(fileFilterTranslations), len(languages))
	}
	for _, language := range languages {
		words, ok := fileFilterTranslations[language.Code]
		if !ok {
			t.Fatalf("missing file filter translation for %s", language.Code)
		}
		if strings.TrimSpace(words.Cue) == "" {
			t.Fatalf("empty file filter cue for %s", language.Code)
		}
	}
}

func TestFileFilterWordsNormalizeAliasesAndFallback(t *testing.T) {
	if got := fileFilterWordsForLanguage("HR_hr").Cue; got != fileFilterTranslations["hr"].Cue {
		t.Fatalf("Croatian alias normalized to %q", got)
	}
	if got := fileFilterWordsForLanguage("unsupported").Cue; got != fileFilterTranslations["en"].Cue {
		t.Fatalf("fallback=%q", got)
	}
}
