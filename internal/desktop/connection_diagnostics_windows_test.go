//go:build windows

package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/by-ftp/internal/remote"
)

func TestConnectionDiagnosticStatusShowsSecureWebRoot(t *testing.T) {
	got := connectionDiagnosticStatus("example.test", remote.ConnectionDiagnostics{
		Secure: true, RootMode: "account", WebRoot: "public_html", WebRootDetected: true,
	})
	for _, want := range []string{"Povezano: example.test", "siguran prijenos", "web root: public_html"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status %q is missing %q", got, want)
		}
	}
}

func TestConnectionDiagnosticStatusShowsPlainFTPAccountRoot(t *testing.T) {
	got := connectionDiagnosticStatus("example.test", remote.ConnectionDiagnostics{
		Secure: false, RootMode: "account",
	})
	for _, want := range []string{"FTP bez enkripcije", "account root spreman"} {
		if !strings.Contains(got, want) {
			t.Fatalf("status %q is missing %q", got, want)
		}
	}
}

func TestConnectionDiagnosticStatusShowsSFTPHome(t *testing.T) {
	got := connectionDiagnosticStatus("example.test", remote.ConnectionDiagnostics{
		Secure: true, RootMode: "home",
	})
	if !strings.Contains(got, "SFTP home spreman") {
		t.Fatalf("unexpected SFTP status: %q", got)
	}
}
