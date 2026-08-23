package i18n

import "strings"

var additionalLocaleOverrides = map[string]map[string]string{
	"it": {
		"app.title": "ByFTP", "app.subtitle": "Trasferimento file sicuro", "section.local": "LOCALE", "section.remote": "SERVER", "section.transfers": "TRASFERIMENTI",
		"button.connect": "Connetti", "button.disconnect": "Disconnetti", "button.refresh": "Aggiorna", "button.upload": "Carica", "button.download": "Scarica", "button.cancel": "Annulla",
		"status.ready": "Pronto", "status.connecting": "Connessione…", "status.connected": "Connesso", "settings.title": "Impostazioni", "settings.language": "Lingua",
	},
	"pl": {
		"app.title": "ByFTP", "app.subtitle": "Bezpieczny transfer plików", "section.local": "LOKALNE", "section.remote": "SERWER", "section.transfers": "TRANSFERY",
		"button.connect": "Połącz", "button.disconnect": "Rozłącz", "button.refresh": "Odśwież", "button.upload": "Wyślij", "button.download": "Pobierz", "button.cancel": "Anuluj",
		"status.ready": "Gotowe", "status.connecting": "Łączenie…", "status.connected": "Połączono", "settings.title": "Ustawienia", "settings.language": "Język",
	},
	"nl": {
		"app.title": "ByFTP", "app.subtitle": "Veilige bestandsoverdracht", "section.local": "LOKAAL", "section.remote": "SERVER", "section.transfers": "OVERDRACHTEN",
		"button.connect": "Verbinden", "button.disconnect": "Verbreken", "button.refresh": "Vernieuwen", "button.upload": "Uploaden", "button.download": "Downloaden", "button.cancel": "Annuleren",
		"status.ready": "Gereed", "status.connecting": "Verbinden…", "status.connected": "Verbonden", "settings.title": "Instellingen", "settings.language": "Taal",
	},
	"cs": {
		"app.title": "ByFTP", "app.subtitle": "Bezpečný přenos souborů", "section.local": "MÍSTNÍ", "section.remote": "SERVER", "section.transfers": "PŘENOSY",
		"button.connect": "Připojit", "button.disconnect": "Odpojit", "button.refresh": "Obnovit", "button.upload": "Nahrát", "button.download": "Stáhnout", "button.cancel": "Zrušit",
		"status.ready": "Připraveno", "status.connecting": "Připojování…", "status.connected": "Připojeno", "settings.title": "Nastavení", "settings.language": "Jazyk",
	},
	"uk": {
		"app.title": "ByFTP", "app.subtitle": "Безпечне передавання файлів", "section.local": "ЛОКАЛЬНО", "section.remote": "СЕРВЕР", "section.transfers": "ПЕРЕДАВАННЯ",
		"button.connect": "Підключити", "button.disconnect": "Відключити", "button.refresh": "Оновити", "button.upload": "Вивантажити", "button.download": "Завантажити", "button.cancel": "Скасувати",
		"status.ready": "Готово", "status.connecting": "Підключення…", "status.connected": "Підключено", "settings.title": "Налаштування", "settings.language": "Мова",
	},
	"sv": {
		"app.title": "ByFTP", "app.subtitle": "Säker filöverföring", "section.local": "LOKALT", "section.remote": "SERVER", "section.transfers": "ÖVERFÖRINGAR",
		"button.connect": "Anslut", "button.disconnect": "Koppla från", "button.refresh": "Uppdatera", "button.upload": "Ladda upp", "button.download": "Ladda ner", "button.cancel": "Avbryt",
		"status.ready": "Klar", "status.connecting": "Ansluter…", "status.connected": "Ansluten", "settings.title": "Inställningar", "settings.language": "Språk",
	},
}

func init() {
	english := catalogs[DefaultLanguage]
	for code, overrides := range additionalLocaleOverrides {
		catalog := make(map[string]string, len(english))
		for key, value := range english {
			catalog[key] = value
		}
		for key, value := range overrides {
			if _, ok := english[key]; ok {
				catalog[key] = value
			}
		}
		catalogs[code] = catalog
	}

	codes := "en|hr|de|fr|es|tr|el|pt|zh|ru|hi|ja|it|pl|nl|cs|uk|sv"
	for _, catalog := range catalogs {
		if _, ok := catalog["terminal.language_usage"]; ok {
			catalog["terminal.language_usage"] = "Usage: language <" + codes + ">"
		}
		for key, value := range catalog {
			value = strings.ReplaceAll(value, "Brendigo", "ByFTP")
			value = strings.ReplaceAll(value, "brendigo.com", "github.com/bren-wp/by-ftp")
			catalog[key] = value
		}
	}
}
