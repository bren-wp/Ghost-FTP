package desktop

import (
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type siteManagerNavigationRow struct {
	Primary   string
	Secondary string
	Quick     bool
}

// siteManagerNavigationRows keeps the left Site Manager navigation compact and
// privacy-safe. The row intentionally exposes only profile name, protocol and
// host: usernames, local paths, fingerprints and credential state stay in the
// details pane where they belong.
func siteManagerNavigationRows(quickLabel string, profiles []model.PublicProfile) []siteManagerNavigationRow {
	rows := make([]siteManagerNavigationRow, 0, len(profiles)+1)
	rows = append(rows, siteManagerNavigationRow{
		Primary: strings.TrimSpace(quickLabel),
		Quick:   true,
	})
	for _, profile := range profiles {
		protocol := strings.ToUpper(strings.TrimSpace(profile.Protocol))
		host := strings.TrimSpace(profile.Host)
		secondary := protocol
		if host != "" {
			if secondary != "" {
				secondary += " · "
			}
			secondary += host
		}
		rows = append(rows, siteManagerNavigationRow{
			Primary:   strings.TrimSpace(profile.Name),
			Secondary: secondary,
		})
	}
	return rows
}

func siteManagerNavigationThemeClass(dark bool) string {
	if dark {
		return "DarkMode_Explorer"
	}
	return ""
}
