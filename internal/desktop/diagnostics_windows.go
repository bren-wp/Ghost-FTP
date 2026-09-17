//go:build windows

package desktop

import (
	"fmt"
	"strings"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func (a *app) showDiagnostics() {
	if a == nil {
		return
	}
	words, ok := diagnosticsWords[a.languageCode()]
	if !ok {
		words = diagnosticsWords["en"]
	}
	state := words.RemoteOffline
	if a.connected {
		state = a.tr("badge.connected")
	}
	title := navigationLabelsForLanguage(a.languageCode()).Diagnostics
	body := fmt.Sprintf(
		"Ghost FTP %s\n\n%s · %s\n\n%s",
		a.version,
		strings.ToUpper(a.protocolValue()),
		state,
		words.PrivacyBody,
	)
	platform.CompactInfoDialog(
		"Ghost FTP — "+title,
		title,
		body,
		okLabel(a.languageCode()),
	)
}
