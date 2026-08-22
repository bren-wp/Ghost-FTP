package usererror

import (
	"context"
	"errors"
	"testing"

	"brendigo.com/byftp/internal/i18n"
)

func TestMessageDefaultsToEnglishAndHidesToolDetails(t *testing.T) {
	lowLevel := `curl: (67) Login denied; server said: 530 Login incorrect`
	got := Message(errors.New(lowLevel), "Connection failed.")
	want := i18n.T("en", "error.auth")
	if got != want {
		t.Fatalf("unexpected message: %q", got)
	}
	if got == lowLevel {
		t.Fatal("low-level details leaked")
	}
}

func TestMessageForLocalizesKnownErrors(t *testing.T) {
	err := errors.New("421 Too many connections from this IP")
	for _, language := range i18n.Languages() {
		got := MessageFor(language.Code, err, "fallback")
		want := i18n.T(language.Code, "error.ftp_limit")
		if got != want {
			t.Fatalf("%s: got %q want %q", language.Code, got, want)
		}
	}
}

func TestMessageDeadlineIsLocalized(t *testing.T) {
	if got, want := MessageFor("de", context.DeadlineExceeded, "x"), i18n.T("de", "error.timeout"); got != want {
		t.Fatalf("unexpected German deadline message: %q", got)
	}
}

func TestMessageMissingSFTPComponent(t *testing.T) {
	got := Message(errors.New("SFTP podrška nije dostupna u sustavu Windows"), "x")
	if want := i18n.T("en", "error.sftp_unavailable"); got != want {
		t.Fatalf("unexpected SFTP component message: %q", got)
	}
}

func TestMessageDisconnectLifecycleWinsJoinedDeadline(t *testing.T) {
	err := errors.Join(context.DeadlineExceeded, errors.New("sigurno zatvaranje veze još traje"))
	got := MessageFor("hr", err, "x")
	if want := i18n.T("hr", "error.disconnect_closing"); got != want {
		t.Fatalf("unexpected joined-error message: %q", got)
	}
}

func TestMessageFallback(t *testing.T) {
	got := MessageFor("ja", errors.New("opaque-internal-tool-error"), "Custom fallback")
	if got != "Custom fallback" {
		t.Fatalf("unexpected fallback: %q", got)
	}
	got = MessageFor("ja", errors.New("opaque-internal-tool-error"), "")
	if want := i18n.T("ja", "error.generic"); got != want {
		t.Fatalf("unexpected localized default fallback: %q", got)
	}
}
