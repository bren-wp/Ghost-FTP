//go:build windows

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestNativeWindowsPromptsCoverCanonicalLanguages(t *testing.T) {
	languages := i18n.Languages()
	if len(languages) != 24 {
		t.Fatalf("canonical language count = %d, want 24", len(languages))
	}

	englishClose := closeQuestion("en")
	englishDirectory := directoryDialogTitle("en")
	for _, language := range languages {
		code := language.Code
		values := []struct {
			name  string
			value string
		}{
			{"ok", okLabel(code)},
			{"close question", closeQuestion(code)},
			{"close body", closeBody(code)},
			{"private-key title", privateKeyDialogTitle(code)},
			{"private-key filter", privateKeyFilterLabel(code)},
			{"all-files filter", allFilesFilterLabel(code)},
			{"directory title", directoryDialogTitle(code)},
		}
		for _, item := range values {
			if item.value == "" {
				t.Fatalf("%s prompt is empty for %s", item.name, code)
			}
		}
		if code != "en" {
			if closeQuestion(code) == englishClose {
				t.Fatalf("close question silently fell back to English for %s", code)
			}
			if directoryDialogTitle(code) == englishDirectory {
				t.Fatalf("directory title silently fell back to English for %s", code)
			}
		}
	}
}

func TestNativeWindowsPromptsUseEnglishFallbackForUnknownLocale(t *testing.T) {
	const unknown = "zz-ZZ"
	checks := []struct {
		name    string
		got     string
		english string
	}{
		{"ok", okLabel(unknown), okLabel("en")},
		{"close question", closeQuestion(unknown), closeQuestion("en")},
		{"close body", closeBody(unknown), closeBody("en")},
		{"private-key title", privateKeyDialogTitle(unknown), privateKeyDialogTitle("en")},
		{"private-key filter", privateKeyFilterLabel(unknown), privateKeyFilterLabel("en")},
		{"all-files filter", allFilesFilterLabel(unknown), allFilesFilterLabel("en")},
		{"directory title", directoryDialogTitle(unknown), directoryDialogTitle("en")},
	}
	for _, check := range checks {
		if check.got != check.english {
			t.Fatalf("%s fallback = %q, want English %q", check.name, check.got, check.english)
		}
	}
}
