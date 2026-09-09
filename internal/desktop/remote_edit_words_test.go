package desktop

import (
	"testing"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
)

func TestRemoteEditWordsCoverSupportedLanguages(t *testing.T) {
	for _, language := range i18n.Languages() {
		words := remoteEditWords(language.Code)
		for name, value := range map[string]string{
			"edit": words.Edit, "title": words.Title, "save": words.Save,
			"reload": words.Reload, "close": words.Close, "hint": words.Hint,
			"conflict": words.Conflict, "unsupported": words.Unsupported,
			"discard close": words.DiscardClose, "discard reload": words.DiscardReload,
			"mixed newlines": words.MixedNewlines, "connection changed": words.ConnectionChanged,
		} {
			if value == "" {
				t.Fatalf("language %s has empty remote editor %s text", language.Code, name)
			}
		}
	}
}
