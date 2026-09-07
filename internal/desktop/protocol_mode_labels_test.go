package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestProtocolModeLabelWordsCoverLanguageRegistry(t *testing.T) {
	languages := i18n.Languages()
	if len(languages) != 24 {
		t.Fatalf("language count = %d, want 24", len(languages))
	}
	for _, language := range languages {
		pair := protocolModeLabelWords(language.Code)
		if pair[0] == "" || pair[1] == "" {
			t.Fatalf("%s has an empty FTPS mode label: %#v", language.Code, pair)
		}
		if pair[0] == pair[1] {
			t.Fatalf("%s explicit and implicit FTPS labels must differ: %#v", language.Code, pair)
		}
	}
}

func TestProtocolModeLabelWordsFallsBackToEnglish(t *testing.T) {
	got := protocolModeLabelWords("unsupported")
	want := protocolModeLabelWords(i18n.DefaultLanguage)
	if got != want {
		t.Fatalf("fallback = %#v, want %#v", got, want)
	}
}
