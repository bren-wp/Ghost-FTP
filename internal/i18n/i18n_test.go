package i18n

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var formatVerbRE = regexp.MustCompile(`%[sdv]`)

func TestCatalogsAreCompleteAndFormatCompatible(t *testing.T) {
	if err := ValidateCatalogs(); err != nil {
		t.Fatal(err)
	}
	english := catalogs[DefaultLanguage]
	supplemental := map[string]bool{
		"it": true, "pl": true, "nl": true, "cs": true, "uk": true, "sv": true,
		"ro": true, "hu": true, "da": true, "fi": true, "no": true, "ko": true,
	}
	for _, language := range Languages() {
		catalog := catalogs[language.Code]
		different := 0
		for key, englishValue := range english {
			if catalog[key] != englishValue {
				different++
			}
			if got, want := formatVerbRE.FindAllString(catalog[key], -1), formatVerbRE.FindAllString(englishValue, -1); !reflect.DeepEqual(got, want) {
				t.Fatalf("format verbs differ for %s/%s: got %v want %v", language.Code, key, got, want)
			}
		}
		if language.Code == DefaultLanguage {
			continue
		}
		translated, total := TranslationCoverage(language.Code)
		if translated != different || total != len(english) {
			t.Fatalf("coverage accounting drift for %s: translated=%d/%d different=%d/%d", language.Code, translated, total, different, len(english))
		}
		if supplemental[language.Code] {
			if different < 30 {
				t.Fatalf("supplemental catalog %s needs at least 30 genuinely localized strings, got %d", language.Code, different)
			}
			continue
		}
		if different*100 < len(english)*70 {
			t.Fatalf("catalog %s appears to fall back to English too often: %d/%d localized", language.Code, different, len(english))
		}
	}
}

func TestLanguagesAndNormalization(t *testing.T) {
	languages := Languages()
	if got := len(languages); got != 24 {
		t.Fatalf("expected 24 supported languages, got %d", got)
	}
	if languages[0].Code != DefaultLanguage || DefaultLanguage != "en" {
		t.Fatalf("English must remain the primary/default language: first=%q default=%q", languages[0].Code, DefaultLanguage)
	}
	cases := map[string]string{
		"": "en", "EN": "en", "pt-BR": "pt", "zh_CN": "zh", "zh-Hans": "zh", "de-DE": "de", "it-IT": "it", "uk-UA": "uk",
		"ro-RO": "ro", "hu-HU": "hu", "da-DK": "da", "fi-FI": "fi", "nb-NO": "no", "nn-NO": "no", "no-NO": "no", "ko-KR": "ko", "unknown": "en",
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", input, got, want)
		}
	}
	if IsSupported("unknown") {
		t.Fatal("unknown language must not be reported as supported")
	}
	for _, code := range []string{"sv-SE", "ro-RO", "hu-HU", "da-DK", "fi-FI", "nb-NO", "no-NO", "ko-KR"} {
		if !IsSupported(code) {
			t.Fatalf("regional form of supported language should be supported: %s", code)
		}
	}
}

func TestPublicTranslationsNeverExposeTechnicalGhostFTPBrand(t *testing.T) {
	for _, language := range Languages() {
		values := map[string]string{
			"settings.title":   T(language.Code, "settings.title"),
			"about.body":       T(language.Code, "about.body", "product.example", "contact.example"),
			"sftp.security":    T(language.Code, "sftp.security"),
			"disconnect.title": T(language.Code, "disconnect.title"),
			"terminal.title":   T(language.Code, "terminal.title", "1.1.5", "test"),
		}
		for key, value := range values {
			if strings.Contains(value, "GhostFTP") {
				t.Fatalf("public translation %s/%s leaked technical brand: %q", language.Code, key, value)
			}
		}
	}
	if got := T("en", "settings.title"); got != "Ghost FTP — Settings" {
		t.Fatalf("settings title = %q", got)
	}
	if got := T("hr", "settings.title"); got != "Ghost FTP — Postavke" {
		t.Fatalf("Croatian settings title = %q", got)
	}
}

func TestAffirmativeAnswers(t *testing.T) {
	cases := []struct{ language, answer string }{
		{"en", "yes"}, {"hr", "da"}, {"de", "ja"}, {"fr", "oui"}, {"es", "sí"}, {"tr", "evet"}, {"el", "ναι"}, {"pt", "sim"}, {"zh", "是"}, {"ru", "да"}, {"hi", "हाँ"}, {"ja", "はい"},
		{"it", "sì"}, {"pl", "tak"}, {"nl", "ja"}, {"cs", "ano"}, {"uk", "так"}, {"sv", "ja"}, {"ro", "da"}, {"hu", "igen"}, {"da", "ja"}, {"fi", "kyllä"}, {"no", "ja"}, {"ko", "예"}, {"nb-NO", "ja"},
	}
	for _, tc := range cases {
		if !IsAffirmative(tc.language, tc.answer) {
			t.Fatalf("expected affirmative answer for %s: %q", tc.language, tc.answer)
		}
	}
	if IsAffirmative("en", "no") {
		t.Fatal("negative answer must not be accepted")
	}
}
