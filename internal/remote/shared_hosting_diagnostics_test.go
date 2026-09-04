package remote

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestDiagnoseConnectionFindsPreferredWebRoot(t *testing.T) {
	items := []model.Item{
		{Name: "www", IsDirectory: true},
		{Name: "public_html", IsDirectory: true},
		{Name: "index.php", IsDirectory: false},
	}

	got := diagnoseConnection("ftps", items)
	if !got.Secure {
		t.Fatal("FTPS diagnostics must report a secure transport")
	}
	if got.RootMode != "account" {
		t.Fatalf("unexpected root mode: %q", got.RootMode)
	}
	if !got.WebRootDetected || got.WebRoot != "public_html" {
		t.Fatalf("expected public_html to win web-root priority, got %#v", got)
	}
	if got.RootEntryCount != len(items) {
		t.Fatalf("unexpected root entry count: %d", got.RootEntryCount)
	}
}

func TestDiagnoseConnectionDoesNotTreatFilesOrSymlinksAsWebRoot(t *testing.T) {
	items := []model.Item{
		{Name: "public_html", IsDirectory: false},
		{Name: "www", IsDirectory: true, IsSymlink: true},
		{Name: " htdocs ", IsDirectory: true},
		{Name: "web", IsDirectory: true},
	}

	got := diagnoseConnection("ftp", items)
	if got.Secure {
		t.Fatal("plain FTP diagnostics must remain visibly insecure")
	}
	if !got.WebRootDetected || got.WebRoot != "web" {
		t.Fatalf("expected exact real directory web without trimming unsafe candidates, got %#v", got)
	}
}

func TestDiagnoseConnectionUsesSFTPHomeRootWithoutInventingWebRoot(t *testing.T) {
	got := diagnoseConnection("sftp", []model.Item{{Name: "backups", IsDirectory: true}})
	if !got.Secure {
		t.Fatal("SFTP diagnostics must report a secure transport")
	}
	if got.RootMode != "home" {
		t.Fatalf("unexpected SFTP root mode: %q", got.RootMode)
	}
	if got.WebRootDetected || got.WebRoot != "" {
		t.Fatalf("diagnostics invented a web root: %#v", got)
	}
}
