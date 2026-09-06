package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestCredentialConsentCoversEverySupportedLanguage(t *testing.T) {
	for _, language := range i18n.Languages() {
		words := credentialConsentText(language.Code)
		if strings.TrimSpace(words.Title) == "" || strings.TrimSpace(words.Question) == "" || strings.TrimSpace(words.Body) == "" {
			t.Fatalf("credential consent text is incomplete for %s: %#v", language.Code, words)
		}
	}
}

func TestCredentialConsentUnknownLanguageFallsBackToEnglish(t *testing.T) {
	if got, want := credentialConsentText("unsupported"), credentialConsentText(i18n.DefaultLanguage); got != want {
		t.Fatalf("unexpected consent fallback: got %#v want %#v", got, want)
	}
}
