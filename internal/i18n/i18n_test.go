package i18n

import (
	"reflect"
	"regexp"
	"testing"
)

var formatVerbRE = regexp.MustCompile(`%[sdv]`)

func TestCatalogsAreCompleteAndFormatCompatible(t *testing.T) {
	if err := ValidateCatalogs(); err != nil {
		t.Fatal(err)
	}
	english := catalogs[DefaultLanguage]
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
		if language.Code != DefaultLanguage && different*100 < len(english)*70 {
			t.Fatalf("catalog %s appears to fall back to English too often: %d/%d localized", language.Code, different, len(english))
		}
	}
}

func TestLanguagesAndNormalization(t *testing.T) {
	if got := len(Languages()); got != 12 {
		t.Fatalf("expected 12 supported languages, got %d", got)
	}
	cases := map[string]string{
		"": "en",
		"EN": "en",
		"pt-BR": "pt",
		"zh_CN": "zh",
		"de-DE": "de",
		"unknown": "en",
	}
	for input, want := range cases {
		if got := Normalize(input); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", input, got, want)
		}
	}
	if IsSupported("unknown") {
		t.Fatal("unknown language must not be reported as supported")
	}
	if !IsSupported("ja-JP") {
		t.Fatal("regional form of supported language should be supported")
	}
}

func TestAffirmativeAnswers(t *testing.T) {
	cases := []struct{ language, answer string }{
		{"en", "yes"}, {"hr", "da"}, {"de", "ja"}, {"fr", "oui"}, {"es", "sí"}, {"tr", "evet"},
		{"el", "ναι"}, {"pt", "sim"}, {"zh", "是"}, {"ru", "да"}, {"hi", "हाँ"}, {"ja", "はい"},
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
