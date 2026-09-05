//go:build windows

package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/remote"
)

func TestConnectionDiagnosticStatusUsesActiveEnglishLocale(t *testing.T) {
	a := &app{settings: model.Settings{Language: "en"}}
	host := "example.test"
	got := a.connectionDiagnosticStatus(host, remote.ConnectionDiagnostics{
		Secure: true, RootMode: "account", WebRoot: "public_html", WebRootDetected: true,
	})
	want := i18n.T("en", "connection.connected", host)
	if got != want {
		t.Fatalf("English connection status = %q, want %q", got, want)
	}
}

func TestConnectionDiagnosticStatusUsesActiveCroatianLocale(t *testing.T) {
	a := &app{settings: model.Settings{Language: "hr"}}
	host := "example.test"
	got := a.connectionDiagnosticStatus(host, remote.ConnectionDiagnostics{
		Secure: false, RootMode: "account",
	})
	want := i18n.T("hr", "connection.connected", host)
	if got != want {
		t.Fatalf("Croatian connection status = %q, want %q", got, want)
	}
}

func TestConnectionDiagnosticStatusDoesNotMixTransportDiagnosticsIntoConciseStatus(t *testing.T) {
	a := &app{settings: model.Settings{Language: "en"}}
	host := "example.test"

	statuses := map[string]string{
		"secure web root": a.connectionDiagnosticStatus(host, remote.ConnectionDiagnostics{
			Secure: true, RootMode: "account", WebRoot: "public_html", WebRootDetected: true,
		}),
		"plain account root": a.connectionDiagnosticStatus(host, remote.ConnectionDiagnostics{
			Secure: false, RootMode: "account",
		}),
		"sftp home": a.connectionDiagnosticStatus(host, remote.ConnectionDiagnostics{
			Secure: true, RootMode: "home",
		}),
	}

	want := i18n.T("en", "connection.connected", host)
	for name, got := range statuses {
		if got != want {
			t.Fatalf("%s status = %q, want concise localized status %q", name, got, want)
		}
	}
}
