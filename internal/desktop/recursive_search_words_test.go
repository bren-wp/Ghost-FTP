package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestRecursiveSearchWordsCoverEverySupportedLanguage(t *testing.T) {
	languages := i18n.Languages()
	if len(recursiveSearchTranslations) != len(languages) {
		t.Fatalf("recursive search translations=%d languages=%d", len(recursiveSearchTranslations), len(languages))
	}
	for _, language := range languages {
		words, ok := recursiveSearchTranslations[language.Code]
		if !ok {
			t.Fatalf("missing recursive search translation for %s", language.Code)
		}
		for label, value := range map[string]string{
			"search": words.Search,
			"navigate": words.Navigate,
			"cancel": words.Cancel,
			"close": words.Close,
			"searching": words.Searching,
			"done": words.Done,
			"cancelled": words.Cancelled,
			"noResults": words.NoResults,
			"localDisclosure": words.LocalDisclosure,
			"remoteDisclosure": words.RemoteDisclosure,
		} {
			if strings.TrimSpace(value) == "" {
				t.Fatalf("empty recursive search %s for %s", label, language.Code)
			}
		}
	}
}

func TestRecursiveSearchWordsNormalizeAliasesAndFallback(t *testing.T) {
	if got := recursiveSearchWordsForLanguage("HR_hr").Search; got != recursiveSearchTranslations["hr"].Search {
		t.Fatalf("Croatian alias normalized to %q", got)
	}
	if got := recursiveSearchWordsForLanguage("unsupported").Search; got != recursiveSearchTranslations["en"].Search {
		t.Fatalf("fallback=%q", got)
	}
}
