package i18n

var additionalLocaleOverrides = map[string]map[string]string{
	"it": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Gestione hosting rapida", "section.local": "COMPUTER LOCALE", "section.remote": "SERVER", "section.transfers": "TRASFERIMENTI",
		"common.settings": "Impostazioni", "common.connect": "Connetti", "common.disconnect": "Disconnetti", "common.refresh": "Aggiorna", "common.cancel": "Annulla",
		"transfer.upload": "Carica", "transfer.download": "Scarica", "transfer.pause": "Pausa", "transfer.resume": "Riprendi", "transfer.retry": "Riprova", "transfer.clear": "Cancella completati",
		"status.ready": "Pronto. Seleziona un profilo salvato o inserisci i dati del server.", "settings.title": "ByFTP — Impostazioni", "profile.save": "Salva profilo", "profile.delete": "Elimina profilo",
	},
	"pl": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Szybkie zarządzanie hostingiem", "section.local": "KOMPUTER LOKALNY", "section.remote": "SERWER", "section.transfers": "TRANSFER",
		"common.settings": "Ustawienia", "common.connect": "Połącz", "common.disconnect": "Rozłącz", "common.refresh": "Odśwież", "common.cancel": "Anuluj",
		"transfer.upload": "Wyślij", "transfer.download": "Pobierz", "transfer.pause": "Wstrzymaj", "transfer.resume": "Wznów", "transfer.retry": "Ponów", "transfer.clear": "Wyczyść zakończone",
		"status.ready": "Gotowe. Wybierz zapisany profil lub wprowadź dane serwera.", "settings.title": "ByFTP — Ustawienia", "profile.save": "Zapisz profil", "profile.delete": "Usuń profil",
	},
	"nl": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Snel hostingbeheer", "section.local": "LOKALE COMPUTER", "section.remote": "SERVER", "section.transfers": "OVERDRACHTEN",
		"common.settings": "Instellingen", "common.connect": "Verbinden", "common.disconnect": "Verbreken", "common.refresh": "Vernieuwen", "common.cancel": "Annuleren",
		"transfer.upload": "Uploaden", "transfer.download": "Downloaden", "transfer.pause": "Pauzeren", "transfer.resume": "Hervatten", "transfer.retry": "Opnieuw", "transfer.clear": "Voltooide wissen",
		"status.ready": "Gereed. Kies een opgeslagen profiel of voer servergegevens in.", "settings.title": "ByFTP — Instellingen", "profile.save": "Profiel opslaan", "profile.delete": "Profiel verwijderen",
	},
	"cs": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Rychlá správa hostingu", "section.local": "MÍSTNÍ POČÍTAČ", "section.remote": "SERVER", "section.transfers": "PŘENOSY",
		"common.settings": "Nastavení", "common.connect": "Připojit", "common.disconnect": "Odpojit", "common.refresh": "Obnovit", "common.cancel": "Zrušit",
		"transfer.upload": "Nahrát", "transfer.download": "Stáhnout", "transfer.pause": "Pozastavit", "transfer.resume": "Pokračovat", "transfer.retry": "Opakovat", "transfer.clear": "Vymazat dokončené",
		"status.ready": "Připraveno. Vyberte uložený profil nebo zadejte údaje serveru.", "settings.title": "ByFTP — Nastavení", "profile.save": "Uložit profil", "profile.delete": "Smazat profil",
	},
	"uk": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Швидке керування хостингом", "section.local": "ЛОКАЛЬНИЙ КОМП'ЮТЕР", "section.remote": "СЕРВЕР", "section.transfers": "ПЕРЕДАВАННЯ",
		"common.settings": "Налаштування", "common.connect": "Підключити", "common.disconnect": "Відключити", "common.refresh": "Оновити", "common.cancel": "Скасувати",
		"transfer.upload": "Вивантажити", "transfer.download": "Завантажити", "transfer.pause": "Призупинити", "transfer.resume": "Продовжити", "transfer.retry": "Повторити", "transfer.clear": "Очистити завершені",
		"status.ready": "Готово. Виберіть збережений профіль або введіть дані сервера.", "settings.title": "ByFTP — Налаштування", "profile.save": "Зберегти профіль", "profile.delete": "Видалити профіль",
	},
	"sv": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Snabb hostinghantering", "section.local": "LOKAL DATOR", "section.remote": "SERVER", "section.transfers": "ÖVERFÖRINGAR",
		"common.settings": "Inställningar", "common.connect": "Anslut", "common.disconnect": "Koppla från", "common.refresh": "Uppdatera", "common.cancel": "Avbryt",
		"transfer.upload": "Ladda upp", "transfer.download": "Ladda ner", "transfer.pause": "Pausa", "transfer.resume": "Fortsätt", "transfer.retry": "Försök igen", "transfer.clear": "Rensa slutförda",
		"status.ready": "Klar. Välj en sparad profil eller ange serveruppgifter.", "settings.title": "ByFTP — Inställningar", "profile.save": "Spara profil", "profile.delete": "Ta bort profil",
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
	}
}
