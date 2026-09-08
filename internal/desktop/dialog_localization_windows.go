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

func activeDialogLocale() (language, cancelLabel string) {
	language = "en"
	cancelLabel = "Cancel"
	apps.Range(func(_, value any) bool {
		a, ok := value.(*app)
		if !ok || a == nil {
			return true
		}
		language = a.languageCode()
		cancelLabel = a.tr("common.cancel")
		return false
	})
	return language, cancelLabel
}

// The platform layer stays independent from the desktop translation catalog.
// Both providers resolve labels when their native surface opens, so startup,
// runtime locale changes and asynchronous dialogs all use the same live locale.
func init() {
	platform.SetDialogLabelProvider(func() (string, string, string, string) {
		language, cancelLabel := activeDialogLocale()
		return okLabel(language), cancelLabel, yesLabel(language), noLabel(language)
	})
	platform.SetPickerLabelProvider(func() (string, string, string, string) {
		language, _ := activeDialogLocale()
		return privateKeyDialogTitle(language), privateKeyFilterLabel(language), allFilesFilterLabel(language), directoryDialogTitle(language)
	})
}
