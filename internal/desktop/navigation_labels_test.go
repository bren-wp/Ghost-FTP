package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestApplicationNavigationLabelsCoverLanguageRegistry(t *testing.T) {
	languages := i18n.Languages()
	if len(languages) != 24 {
		t.Fatalf("language count = %d, want 24", len(languages))
	}
	if len(applicationNavigation) != len(languages) {
		t.Fatalf("navigation label count = %d, language count = %d", len(applicationNavigation), len(languages))
	}
	for _, language := range languages {
		labels, ok := applicationNavigation[language.Code]
		if !ok {
			t.Fatalf("%s is missing application navigation labels", language.Code)
		}
		if labels.Files == "" || labels.Connections == "" || labels.TransferQueue == "" || labels.SiteManager == "" || labels.Diagnostics == "" {
			t.Fatalf("%s has empty application navigation labels: %#v", language.Code, labels)
		}
	}
}

func TestApplicationNavigationLabelsMatchMasterRail(t *testing.T) {
	en := navigationLabelsForLanguage("en")
	if en.Files != "Files" || en.Connections != "Connections" || en.TransferQueue != "Transfer Queue" {
		t.Fatalf("English master navigation = %#v", en)
	}
	hr := navigationLabelsForLanguage("hr")
	if hr.Files != "Datoteke" || hr.Connections != "Veze" || hr.TransferQueue != "Red prijenosa" {
		t.Fatalf("Croatian master navigation = %#v", hr)
	}
}

func TestApplicationNavigationLabelsFallbackToEnglish(t *testing.T) {
	got := navigationLabelsForLanguage("unsupported")
	want := navigationLabelsForLanguage(i18n.DefaultLanguage)
	if got != want {
		t.Fatalf("fallback = %#v, want %#v", got, want)
	}
}
