package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestDirectoryCompareWordsCoverEverySupportedLanguage(t *testing.T) {
	languages := i18n.Languages()
	if len(directoryCompareTranslations) != len(languages) {
		t.Fatalf("comparison translations=%d languages=%d", len(directoryCompareTranslations), len(languages))
	}
	for _, language := range languages {
		words, ok := directoryCompareTranslations[language.Code]
		if !ok {
			t.Fatalf("missing directory comparison translation for %s", language.Code)
		}
		for label, value := range map[string]string{
			"compare":     words.Compare,
			"close":       words.Close,
			"openBoth":    words.OpenBoth,
			"same":        words.Same,
			"localOnly":   words.LocalOnly,
			"remoteOnly":  words.RemoteOnly,
			"newerLocal":  words.NewerLocal,
			"newerRemote": words.NewerRemote,
			"conflict":    words.Conflict,
			"unknown":     words.Unknown,
			"ready":       words.Ready,
			"unavailable": words.Unavailable,
			"disclosure":  words.Disclosure,
		} {
			if strings.TrimSpace(value) == "" {
				t.Fatalf("empty comparison %s for %s", label, language.Code)
			}
		}
	}
}

func TestDirectoryCompareWordsNormalizeAliasesAndFallback(t *testing.T) {
	if got := directoryCompareWordsForLanguage("HR_hr").Compare; got != directoryCompareTranslations["hr"].Compare {
		t.Fatalf("Croatian alias normalized to %q", got)
	}
	if got := directoryCompareWordsForLanguage("unsupported").Compare; got != directoryCompareTranslations["en"].Compare {
		t.Fatalf("fallback=%q", got)
	}
}
