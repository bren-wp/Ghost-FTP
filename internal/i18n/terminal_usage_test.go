package i18n

import (
	"strings"
	"testing"
)

func TestTerminalLanguageUsageTracksSupportedRegistry(t *testing.T) {
	codes := make([]string, 0, len(Languages()))
	for _, language := range Languages() {
		codes = append(codes, language.Code)
	}
	want := "<" + strings.Join(codes, "|") + ">"
	if len(codes) != 24 {
		t.Fatalf("supported language registry contains %d languages, want 24", len(codes))
	}

	for _, language := range Languages() {
		usage := T(language.Code, "terminal.language_usage")
		if !strings.Contains(usage, want) {
			t.Fatalf("terminal language usage for %s = %q, want registry %q", language.Code, usage, want)
		}
	}
}

func TestExpandTerminalLanguageUsagePreservesLocalizedText(t *testing.T) {
	got := expandTerminalLanguageUsage("Upotreba: language <en|hr>")
	if !strings.HasPrefix(got, "Upotreba: language <") {
		t.Fatalf("localized prefix was not preserved: %q", got)
	}
	if !strings.Contains(got, "|ko>") {
		t.Fatalf("expanded usage does not include the last supported language: %q", got)
	}
}

func TestExpandTerminalLanguageUsageFailsSafeOnMalformedTemplate(t *testing.T) {
	got := expandTerminalLanguageUsage("broken template")
	if !strings.HasPrefix(got, "Usage: language <en|hr|") {
		t.Fatalf("malformed template fallback = %q", got)
	}
	if !strings.HasSuffix(got, "|ko>") {
		t.Fatalf("malformed template fallback is incomplete: %q", got)
	}
}
