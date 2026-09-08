//go:build windows

package desktop

import (
	"strings"
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
			{"yes", yesLabel(code)},
			{"no", noLabel(code)},
			{"close question", closeQuestion(code)},
			{"close body", closeBody(code)},
			{"private-key title", privateKeyDialogTitle(code)},
			{"private-key filter", privateKeyFilterLabel(code)},
			{"all-files filter", allFilesFilterLabel(code)},
			{"directory title", directoryDialogTitle(code)},
			{"profile busy heading", profileBusyHeading(code)},
			{"profile busy body", profileBusyBody(code)},
			{"profile name label", profileNameLabel(code)},
			{"profile name required heading", profileNameRequiredHeading(code)},
			{"profile name required body", profileNameRequiredBody(code)},
			{"profile privacy title", profilePrivacyTitle(code)},
			{"profile retain heading", profileRetainHeading(code)},
			{"profile retain body", profileRetainBody(code)},
			{"profile auto-remove prefix", profileAutoRemovedPrefix(code)},
			{"profile old-credential heading", profileOldCredentialsHeading(code)},
			{"profile old-credential body", profileOldCredentialsBody(code)},
		}
		for _, item := range values {
			if strings.TrimSpace(item.value) == "" {
				t.Fatalf("%s prompt is empty for %s", item.name, code)
			}
		}
		if !strings.Contains(profileRetainBody(code), yesLabel(code)) || !strings.Contains(profileRetainBody(code), noLabel(code)) {
			t.Fatalf("profile retain prompt does not match decision labels for %s", code)
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
		{"yes", yesLabel(unknown), yesLabel("en")},
		{"no", noLabel(unknown), noLabel("en")},
		{"close question", closeQuestion(unknown), closeQuestion("en")},
		{"close body", closeBody(unknown), closeBody("en")},
		{"private-key title", privateKeyDialogTitle(unknown), privateKeyDialogTitle("en")},
		{"private-key filter", privateKeyFilterLabel(unknown), privateKeyFilterLabel("en")},
		{"all-files filter", allFilesFilterLabel(unknown), allFilesFilterLabel("en")},
		{"directory title", directoryDialogTitle(unknown), directoryDialogTitle("en")},
		{"profile busy heading", profileBusyHeading(unknown), profileBusyHeading("en")},
		{"profile busy body", profileBusyBody(unknown), profileBusyBody("en")},
		{"profile name label", profileNameLabel(unknown), profileNameLabel("en")},
		{"profile name required heading", profileNameRequiredHeading(unknown), profileNameRequiredHeading("en")},
		{"profile name required body", profileNameRequiredBody(unknown), profileNameRequiredBody("en")},
		{"profile privacy title", profilePrivacyTitle(unknown), profilePrivacyTitle("en")},
		{"profile retain heading", profileRetainHeading(unknown), profileRetainHeading("en")},
		{"profile retain body", profileRetainBody(unknown), profileRetainBody("en")},
		{"profile auto-remove prefix", profileAutoRemovedPrefix(unknown), profileAutoRemovedPrefix("en")},
		{"profile old-credential heading", profileOldCredentialsHeading(unknown), profileOldCredentialsHeading("en")},
		{"profile old-credential body", profileOldCredentialsBody(unknown), profileOldCredentialsBody("en")},
	}
	for _, check := range checks {
		if check.got != check.english {
			t.Fatalf("%s fallback = %q, want English %q", check.name, check.got, check.english)
		}
	}
}
