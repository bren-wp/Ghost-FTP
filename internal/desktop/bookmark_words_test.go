package desktop

import "testing"

func TestBookmarkWordCatalogCoversSupportedLanguages(t *testing.T) {
	languages := []string{"en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja", "it", "pl", "nl", "cs", "uk", "sv", "ro", "hu", "da", "fi", "no", "ko"}
	for _, language := range languages {
		words := bookmarkWordsForLanguage(language)
		if words.Title == "" || words.AddLocal == "" || words.AddRemote == "" || words.Open == "" || words.Delete == "" || words.Close == "" || words.Name == "" {
			t.Fatalf("bookmark copy incomplete for %q: %#v", language, words)
		}
	}
}

func TestBookmarkWordsFallbackToEnglish(t *testing.T) {
	got := bookmarkWordsForLanguage("xx")
	want := bookmarkWordsForLanguage("en")
	if got != want {
		t.Fatalf("fallback = %#v, want %#v", got, want)
	}
}
