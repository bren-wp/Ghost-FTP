package desktop

import (
	"strings"
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestSiteManagerNavigationRowsAreCompactAndPrivacySafe(t *testing.T) {
	profiles := []model.PublicProfile{
		{
			Name:           "Production",
			Protocol:       "ftps",
			Host:           "ftp.example.com",
			Username:       "private-user@example.com",
			PrivateKeyPath:  "/home/user/.ssh/id_ed25519",
			Fingerprint:     "SHA256:private-fingerprint",
			RemotePath:      "/private/path",
			LocalPath:       "/home/user/customer-files",
			HasPassword:     true,
			HasPassphrase:   true,
		},
	}

	rows := siteManagerNavigationRows(" Quick connect (no profile) ", profiles)
	if len(rows) != 2 {
		t.Fatalf("row count = %d, want 2", len(rows))
	}
	if !rows[0].Quick || rows[0].Primary != "Quick connect (no profile)" || rows[0].Secondary != "" {
		t.Fatalf("unexpected quick-connect row: %#v", rows[0])
	}
	if rows[1].Quick || rows[1].Primary != "Production" || rows[1].Secondary != "FTPS · ftp.example.com" {
		t.Fatalf("unexpected saved-profile row: %#v", rows[1])
	}

	combined := rows[1].Primary + " " + rows[1].Secondary
	for _, forbidden := range []string{
		"private-user@example.com",
		"id_ed25519",
		"private-fingerprint",
		"/private/path",
		"customer-files",
	} {
		if strings.Contains(combined, forbidden) {
			t.Fatalf("left navigation leaked detail %q in %q", forbidden, combined)
		}
	}
}

func TestSiteManagerNavigationRowsDisambiguateDuplicateNames(t *testing.T) {
	rows := siteManagerNavigationRows("Quick", []model.PublicProfile{
		{Name: "Hosting", Protocol: "sftp", Host: "sftp-a.example.com"},
		{Name: "Hosting", Protocol: "ftps", Host: "ftp-b.example.com"},
	})
	if rows[1].Primary != rows[2].Primary {
		t.Fatal("test requires duplicate profile names")
	}
	if rows[1].Secondary == rows[2].Secondary {
		t.Fatalf("duplicate profile names are not disambiguated: %#v %#v", rows[1], rows[2])
	}
}

func TestSiteManagerNavigationThemeClassMatchesAppearance(t *testing.T) {
	if got := siteManagerNavigationThemeClass(false); got != "" {
		t.Fatalf("Classic Light theme class = %q, want empty native theme", got)
	}
	if got := siteManagerNavigationThemeClass(true); got != "DarkMode_Explorer" {
		t.Fatalf("Dark theme class = %q", got)
	}
}
