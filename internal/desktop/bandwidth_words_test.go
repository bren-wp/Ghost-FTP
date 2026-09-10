package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestBandwidthTranslationsCoverEverySupportedLanguage(t *testing.T) {
	languages := i18n.Languages()
	if len(languages) != 24 {
		t.Fatalf("expected canonical 24-language catalog, got %d", len(languages))
	}
	for _, language := range languages {
		words, ok := bandwidthTranslations[language.Code]
		if !ok {
			t.Fatalf("missing bandwidth translation for %s", language.Code)
		}
		if strings.TrimSpace(words.UploadLabel) == "" || strings.TrimSpace(words.DownloadLabel) == "" {
			t.Fatalf("empty bandwidth label for %s: %+v", language.Code, words)
		}
		if !strings.Contains(words.UploadLabel, "KiB/s") || !strings.Contains(words.DownloadLabel, "KiB/s") {
			t.Fatalf("bandwidth units must remain explicit for %s: %+v", language.Code, words)
		}
	}
}

func TestBandwidthTranslationFallbackUsesEnglish(t *testing.T) {
	if got, want := bandwidthWordsForLanguage("xx-invalid"), bandwidthTranslations[i18n.DefaultLanguage]; got != want {
		t.Fatalf("invalid language fallback = %+v, want %+v", got, want)
	}
}
