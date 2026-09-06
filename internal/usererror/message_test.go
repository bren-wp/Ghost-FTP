package usererror

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

type classifiedTestError struct {
	kind string
	raw  string
}

func (e classifiedTestError) Error() string         { return e.raw }
func (e classifiedTestError) UserErrorKind() string { return e.kind }

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

func TestMessageUsesStructuredTransportClassificationWithoutLeakingRawDetail(t *testing.T) {
	raw := `opaque child-process output C:\Users\Person\.ssh\id_ed25519 secret-token-should-not-leak`
	err := classifiedTestError{kind: "sftp_settings", raw: raw}
	for _, language := range i18n.Languages() {
		got := MessageFor(language.Code, err, "fallback")
		want := i18n.T(language.Code, "sftp.failed_body")
		if got != want {
			t.Fatalf("%s: got %q want %q", language.Code, got, want)
		}
		if strings.Contains(got, "secret-token-should-not-leak") || strings.Contains(got, "id_ed25519") {
			t.Fatalf("%s: structured diagnostic leaked raw detail: %q", language.Code, got)
		}
	}
}

func TestMessageStructuredKindWhitelist(t *testing.T) {
	err := classifiedTestError{kind: "error.arbitrary_key", raw: "opaque-internal-tool-error"}
	if got := MessageFor("en", err, "Safe fallback"); got != "Safe fallback" {
		t.Fatalf("unapproved kind selected a message: %q", got)
	}
}

func TestMessageContextDeadlineWinsStructuredClassification(t *testing.T) {
	err := errors.Join(context.DeadlineExceeded, classifiedTestError{kind: "auth", raw: "opaque auth failure"})
	if got, want := MessageFor("hr", err, "x"), i18n.T("hr", "error.timeout"); got != want {
		t.Fatalf("deadline should win structured kind: got %q want %q", got, want)
	}
}

func TestMessageStructuredKindsMapToExpectedCatalogKeys(t *testing.T) {
	tests := map[string]string{
		"hostkey_changed":      "error.hostkey_changed",
		"sftp_unavailable":     "error.sftp_unavailable",
		"sftp_hostkey_missing": "error.sftp_hostkey_missing",
		"sftp_settings":        "sftp.failed_body",
		"auth":                 "error.auth",
		"ftp_limit":            "error.ftp_limit",
		"ftp_data":             "error.ftp_data",
		"resolve":              "error.resolve",
		"refused":              "error.refused",
		"timeout":              "error.timeout",
		"connection_lost":      "error.connection_lost",
		"tls":                  "error.tls",
		"disk":                 "error.disk",
		"permission":           "error.permission",
		"not_found":            "error.not_found",
	}
	for kind, key := range tests {
		t.Run(kind, func(t *testing.T) {
			err := classifiedTestError{kind: kind, raw: "opaque"}
			if got, want := MessageFor("en", err, "fallback"), i18n.T("en", key); got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
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

func TestMessageOpenSSHFallbackMarkers(t *testing.T) {
	tests := []struct {
		name string
		err  string
		key  string
	}{
		{name: "changed host key", err: "REMOTE HOST IDENTIFICATION HAS CHANGED", key: "error.hostkey_changed"},
		{name: "hostname resolution", err: "Could not resolve hostname example.invalid", key: "error.resolve"},
		{name: "keyboard interactive auth", err: "Permission denied (keyboard-interactive)", key: "error.auth"},
		{name: "connection timeout", err: "Connection timed out", key: "error.timeout"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, want := MessageFor("en", errors.New(tc.err), "fallback"), i18n.T("en", tc.key); got != want {
				t.Fatalf("got %q want %q", got, want)
			}
		})
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
