//go:build windows

package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

// openSiteManager provides one clear saved-site entry point from the native
// menu. The selection is deliberately applied back to the canonical profile
// controls before connecting so there is still only one connection/profile
// implementation and one credential-binding path.
func (a *app) openSiteManager() {
	if a == nil || a.connectionBusy || a.connected {
		return
	}
	words := nativeMenuWords(a.languageCode())
	options := make([]string, 0, len(a.profiles)+1)
	options = append(options, a.tr("profile.quick"))
	selected := 0
	for i, profile := range a.profiles {
		label := profile.Name
		if profile.Host != "" {
			label += "  ·  " + profile.Host
		}
		options = append(options, label)
		if profile.ID == a.selectedProfileID {
			selected = i + 1
		}
	}
	index, ok := platform.SelectOptionDialog(
		words[5],
		a.tr("status.ready"),
		"Ghost FTP · FTP / FTPS / SFTP",
		a.tr("common.connect"),
		a.tr("common.cancel"),
		options,
		selected,
	)
	if !ok || index < 0 || index >= len(options) {
		return
	}
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(index), 0)
	a.selectProfile()
	// Quick connect remains an editable connection form. A saved site can be
	// connected immediately because selectProfile has already populated the
	// endpoint and credential cues from the protected profile store.
	if index > 0 {
		a.connectNow()
	}
}
