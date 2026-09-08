//go:build windows

package desktop

import "github.com/bren-wp/Ghost-FTP/internal/platform"

func yesLabel(language string) string {
	return localizedPrompt(language, [24]string{
		"Yes", "Da", "Ja", "Oui", "Sí", "Evet", "Ναι", "Sim", "是", "Да", "हाँ", "はい",
		"Sì", "Tak", "Ja", "Ano", "Так", "Ja", "Da", "Igen", "Ja", "Kyllä", "Ja", "예",
	})
}

func noLabel(language string) string {
	return localizedPrompt(language, [24]string{
		"No", "Ne", "Nein", "Non", "No", "Hayır", "Όχι", "Não", "否", "Нет", "नहीं", "いいえ",
		"No", "Nie", "Nee", "Ne", "Ні", "Nej", "Nu", "Nem", "Nej", "Ei", "Nei", "아니요",
	})
}

// installDialogLabelProvider keeps the platform layer independent from the
// desktop translation catalog while resolving labels at the moment a dialog is
// opened. That makes startup, runtime locale switches and asynchronous security
// confirmations use the same current language without a second i18n state.
func init() {
	platform.SetDialogLabelProvider(func() (string, string, string, string) {
		language := "en"
		cancelLabel := "Cancel"
		apps.Range(func(_, value any) bool {
			a, ok := value.(*app)
			if !ok || a == nil {
				return true
			}
			language = a.languageCode()
			cancelLabel = a.tr("common.cancel")
			return false
		})
		return okLabel(language), cancelLabel, yesLabel(language), noLabel(language)
	})
}
