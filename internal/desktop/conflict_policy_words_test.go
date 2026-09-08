package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestConflictPolicyWordsCoverEverySupportedLanguage(t *testing.T) {
	languages := i18n.Languages()
	if len(localizedConflictPolicyWords) != len(languages) {
		t.Fatalf("conflict-policy localization count=%d want %d", len(localizedConflictPolicyWords), len(languages))
	}
	for _, language := range languages {
		words, ok := localizedConflictPolicyWords[language.Code]
		if !ok {
			t.Fatalf("missing conflict-policy localization for %s", language.Code)
		}
		for label, value := range map[string]string{
			"title": words.Title, "skip": words.Skip, "replace": words.Replace, "replace_backup": words.ReplaceBackup,
		} {
			if strings.TrimSpace(value) == "" {
				t.Fatalf("empty conflict-policy %s for %s", label, language.Code)
			}
			if strings.Contains(value, "GhostFTP") {
				t.Fatalf("technical brand leaked into conflict-policy %s/%s: %q", language.Code, label, value)
			}
		}
	}
}

func TestConflictPolicyWordsAreUserFacingPoliciesNotLegacyBooleanQuestions(t *testing.T) {
	en := conflictPolicyText("en-US")
	if en.Title != "When a destination file already exists" {
		t.Fatalf("English conflict title=%q", en.Title)
	}
	if en.Skip != "Skip existing files" || en.Replace != "Replace existing files" || en.ReplaceBackup != "Replace and keep a backup" {
		t.Fatalf("English conflict choices drifted: %#v", en)
	}
	for _, value := range []string{en.Title, en.Skip, en.Replace, en.ReplaceBackup} {
		if strings.Contains(value, "?") || strings.Contains(strings.ToLower(value), "enabled") {
			t.Fatalf("legacy boolean wording leaked into conflict-policy UI: %q", value)
		}
	}

	hr := conflictPolicyText("hr-HR")
	if hr.Title != "Kada odredišna datoteka već postoji" || hr.ReplaceBackup != "Zamijeni i zadrži sigurnosnu kopiju" {
		t.Fatalf("Croatian conflict-policy wording drifted: %#v", hr)
	}
}
