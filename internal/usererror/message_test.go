package usererror

import (
	"context"
	"errors"
	"testing"

	"github.com/bren-wp/by-ftp/internal/i18n"
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

func TestMessageSharedHostingDataChannelFailure(t *testing.T) {
	got := MessageFor("en", errors.New("425 Can't open data connection"), "x")
	if want := i18n.T("en", "error.ftp_data"); got != want {
		t.Fatalf("unexpected FTP data-channel message: %q", got)
	}
}

func TestMessageSharedHostingTLSFailure(t *testing.T) {
	got := MessageFor("en", errors.New("SSL certificate problem: certificate has expired"), "x")
	if want := i18n.T("en", "error.tls"); got != want {
		t.Fatalf("unexpected TLS message: %q", got)
	}
}

func TestMessageSharedHostingQuotaFailure(t *testing.T) {
	got := MessageFor("en", errors.New("552 Quota exceeded"), "x")
	if want := i18n.T("en", "error.disk"); got != want {
		t.Fatalf("unexpected quota message: %q", got)
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

func TestMessageSessionStillClosing(t *testing.T) {
	got := MessageFor("en", errors.New("previous connection is still closing safely"), "x")
	if want := i18n.T("en", "error.session_closing"); got != want {
		t.Fatalf("unexpected session-closing message: %q", got)
	}
}

func TestMessageDisconnectCleanupStillRunning(t *testing.T) {
	got := MessageFor("en", errors.New("connection is still closing safely"), "x")
	if want := i18n.T("en", "error.disconnect_closing"); got != want {
		t.Fatalf("unexpected disconnect-closing message: %q", got)
	}
}

func TestMessageDisconnectLifecycleWinsJoinedDeadline(t *testing.T) {
	err := errors.Join(context.DeadlineExceeded, errors.New("sigurno zatvaranje veze još traje"))
	got := MessageFor("hr", err, "x")
	if want := i18n.T("hr", "error.disconnect_closing"); got != want {
		t.Fatalf("unexpected joined-error message: %q", got)
	}
}

func TestMessageSFTPHostKeyScanFailure(t *testing.T) {
	got := MessageFor("en", errors.New("could not retrieve sftp host key"), "x")
	if want := i18n.T("en", "error.sftp_hostkey_missing"); got != want {
		t.Fatalf("unexpected SFTP host-key scan message: %q", got)
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
