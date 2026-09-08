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

// syncPlatformDialogLocalization updates every application-owned native action
// label as one atomic UI-language decision. It is called at startup, whenever
// the runtime language changes, and before command flows that may complete
// asynchronously and show a later security confirmation.
func (a *app) syncPlatformDialogLocalization() {
	language := a.languageCode()
	platform.SetDialogActionLabels(okLabel(language), a.tr("common.cancel"))
	platform.SetDialogDecisionLabels(yesLabel(language), noLabel(language))
}
