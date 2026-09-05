package i18n

import "strings"

// Supplemental catalogs intentionally start from the canonical English text so
// a newly introduced string is always readable. CI separately enforces a real
// translated-core coverage floor so a locale cannot be advertised with only a
// token handful of translated labels.
var additionalLocaleOverrides = map[string]map[string]string{
	"it": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Gestione hosting rapida", "badge.connected": "● CONNESSO", "badge.disconnected": "● DISCONNESSO",
		"profile.quick": "Connessione rapida (senza profilo)", "profile.save": "Salva profilo", "profile.delete": "Elimina profilo",
		"common.settings": "Impostazioni", "common.about": "Informazioni", "common.connect": "Connetti", "common.disconnect": "Disconnetti", "common.up": "Su", "common.folder": "Cartella…", "common.refresh": "Aggiorna", "common.new_folder": "Nuova cartella", "common.rename": "Rinomina", "common.delete": "Elimina", "common.permissions": "Permessi", "common.cancel": "Annulla",
		"section.local": "COMPUTER LOCALE", "section.remote": "SERVER", "section.transfers": "TRASFERIMENTI",
		"transfer.upload": "Carica", "transfer.download": "Scarica", "transfer.pause": "Pausa", "transfer.resume": "Riprendi", "transfer.retry": "Riprova", "transfer.clear": "Cancella completati",
		"status.ready": "Pronto. Seleziona un profilo salvato o inserisci i dati del server.", "status.queued": "In coda", "status.running": "In corso", "status.done": "Completato", "status.failed": "Non riuscito", "status.cancelled": "Annullato", "status.skipped": "Ignorato",
		"cue.host": "Server FTP/SFTP, ad es. ftp.example.com", "cue.user": "Nome utente, può essere user@example.com", "cue.password": "Password FTP / SFTP",
		"column.name": "Nome", "column.type": "Tipo", "column.size": "Dimensione", "column.modified": "Modificato", "column.status": "Stato", "column.progress": "Avanzamento", "type.file": "file", "type.folder": "cartella",
		"settings.title": "Ghost FTP — Impostazioni", "settings.invalid_value": "Valore non valido", "settings.parallel": "Trasferimenti paralleli (1–8):", "settings.timeout": "Timeout di connessione (5–60 secondi):", "settings.retries": "Riprova automaticamente un trasferimento non riuscito (0–3 volte):", "settings.retry_delay": "Pausa tra i tentativi automatici (1–30 secondi):",
		"about.title": "Informazioni", "connection.failed": "Connessione non riuscita", "error.generic": "Operazione non riuscita. Controlla la connessione e riprova.",
	},
	"pl": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Szybkie zarządzanie hostingiem", "badge.connected": "● POŁĄCZONO", "badge.disconnected": "● ROZŁĄCZONO",
		"profile.quick": "Szybkie połączenie (bez profilu)", "profile.save": "Zapisz profil", "profile.delete": "Usuń profil",
		"common.settings": "Ustawienia", "common.about": "O programie", "common.connect": "Połącz", "common.disconnect": "Rozłącz", "common.up": "W górę", "common.folder": "Folder…", "common.refresh": "Odśwież", "common.new_folder": "Nowy folder", "common.rename": "Zmień nazwę", "common.delete": "Usuń", "common.permissions": "Uprawnienia", "common.cancel": "Anuluj",
		"section.local": "KOMPUTER LOKALNY", "section.remote": "SERWER", "section.transfers": "TRANSFERy",
		"transfer.upload": "Wyślij", "transfer.download": "Pobierz", "transfer.pause": "Wstrzymaj", "transfer.resume": "Wznów", "transfer.retry": "Ponów", "transfer.clear": "Wyczyść zakończone",
		"status.ready": "Gotowe. Wybierz zapisany profil lub wprowadź dane serwera.", "status.queued": "W kolejce", "status.running": "W toku", "status.done": "Zakończono", "status.failed": "Niepowodzenie", "status.cancelled": "Anulowano", "status.skipped": "Pominięto",
		"cue.host": "Serwer FTP/SFTP, np. ftp.example.com", "cue.user": "Nazwa użytkownika, może być user@example.com", "cue.password": "Hasło FTP / SFTP",
		"column.name": "Nazwa", "column.type": "Typ", "column.size": "Rozmiar", "column.modified": "Zmodyfikowano", "column.status": "Stan", "column.progress": "Postęp", "type.file": "plik", "type.folder": "folder",
		"settings.title": "Ghost FTP — Ustawienia", "settings.invalid_value": "Nieprawidłowa wartość", "settings.parallel": "Równoległe transfery (1–8):", "settings.timeout": "Limit czasu połączenia (5–60 sekund):", "settings.retries": "Automatycznie ponów nieudany transfer (0–3 razy):", "settings.retry_delay": "Opóźnienie między automatycznymi próbami (1–30 sekund):",
		"about.title": "O programie", "connection.failed": "Połączenie nie powiodło się", "error.generic": "Operacja nie powiodła się. Sprawdź połączenie i spróbuj ponownie.",
	},
	"nl": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Snel hostingbeheer", "badge.connected": "● VERBONDEN", "badge.disconnected": "● NIET VERBONDEN",
		"profile.quick": "Snel verbinden (zonder profiel)", "profile.save": "Profiel opslaan", "profile.delete": "Profiel verwijderen",
		"common.settings": "Instellingen", "common.about": "Over", "common.connect": "Verbinden", "common.disconnect": "Verbreken", "common.up": "Omhoog", "common.folder": "Map…", "common.refresh": "Vernieuwen", "common.new_folder": "Nieuwe map", "common.rename": "Hernoemen", "common.delete": "Verwijderen", "common.permissions": "Machtigingen", "common.cancel": "Annuleren",
		"section.local": "LOKALE COMPUTER", "section.remote": "SERVER", "section.transfers": "OVERDRACHTEN",
		"transfer.upload": "Uploaden", "transfer.download": "Downloaden", "transfer.pause": "Pauzeren", "transfer.resume": "Hervatten", "transfer.retry": "Opnieuw", "transfer.clear": "Voltooide wissen",
		"status.ready": "Gereed. Kies een opgeslagen profiel of voer servergegevens in.", "status.queued": "In wachtrij", "status.running": "Bezig", "status.done": "Voltooid", "status.failed": "Mislukt", "status.cancelled": "Geannuleerd", "status.skipped": "Overgeslagen",
		"cue.host": "FTP/SFTP-server, bv. ftp.example.com", "cue.user": "Gebruikersnaam, kan user@example.com zijn", "cue.password": "FTP / SFTP-wachtwoord",
		"column.name": "Naam", "column.type": "Type", "column.size": "Grootte", "column.modified": "Gewijzigd", "column.status": "Status", "column.progress": "Voortgang", "type.file": "bestand", "type.folder": "map",
		"settings.title": "Ghost FTP — Instellingen", "settings.invalid_value": "Ongeldige waarde", "settings.parallel": "Parallelle overdrachten (1–8):", "settings.timeout": "Verbindingstime-out (5–60 seconden):", "settings.retries": "Mislukte overdracht automatisch opnieuw proberen (0–3 keer):", "settings.retry_delay": "Vertraging tussen automatische pogingen (1–30 seconden):",
		"about.title": "Over", "connection.failed": "Verbinding mislukt", "error.generic": "De bewerking is mislukt. Controleer de verbinding en probeer opnieuw.",
	},
	"cs": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Rychlá správa hostingu", "badge.connected": "● PŘIPOJENO", "badge.disconnected": "● ODPOJENO",
		"profile.quick": "Rychlé připojení (bez profilu)", "profile.save": "Uložit profil", "profile.delete": "Smazat profil",
		"common.settings": "Nastavení", "common.about": "O aplikaci", "common.connect": "Připojit", "common.disconnect": "Odpojit", "common.up": "Nahoru", "common.folder": "Složka…", "common.refresh": "Obnovit", "common.new_folder": "Nová složka", "common.rename": "Přejmenovat", "common.delete": "Smazat", "common.permissions": "Oprávnění", "common.cancel": "Zrušit",
		"section.local": "MÍSTNÍ POČÍTAČ", "section.remote": "SERVER", "section.transfers": "PŘENOSY",
		"transfer.upload": "Nahrát", "transfer.download": "Stáhnout", "transfer.pause": "Pozastavit", "transfer.resume": "Pokračovat", "transfer.retry": "Opakovat", "transfer.clear": "Vymazat dokončené",
		"status.ready": "Připraveno. Vyberte uložený profil nebo zadejte údaje serveru.", "status.queued": "Ve frontě", "status.running": "Probíhá", "status.done": "Dokončeno", "status.failed": "Nezdařilo se", "status.cancelled": "Zrušeno", "status.skipped": "Přeskočeno",
		"cue.host": "FTP/SFTP server, např. ftp.example.com", "cue.user": "Uživatelské jméno, může být user@example.com", "cue.password": "Heslo FTP / SFTP",
		"column.name": "Název", "column.type": "Typ", "column.size": "Velikost", "column.modified": "Změněno", "column.status": "Stav", "column.progress": "Průběh", "type.file": "soubor", "type.folder": "složka",
		"settings.title": "Ghost FTP — Nastavení", "settings.invalid_value": "Neplatná hodnota", "settings.parallel": "Souběžné přenosy (1–8):", "settings.timeout": "Časový limit připojení (5–60 sekund):", "settings.retries": "Automaticky opakovat neúspěšný přenos (0–3krát):", "settings.retry_delay": "Prodleva mezi automatickými pokusy (1–30 sekund):",
		"about.title": "O aplikaci", "connection.failed": "Připojení se nezdařilo", "error.generic": "Operace se nezdařila. Zkontrolujte připojení a zkuste to znovu.",
	},
	"uk": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Швидке керування хостингом", "badge.connected": "● ПІДКЛЮЧЕНО", "badge.disconnected": "● ВІДКЛЮЧЕНО",
		"profile.quick": "Швидке підключення (без профілю)", "profile.save": "Зберегти профіль", "profile.delete": "Видалити профіль",
		"common.settings": "Налаштування", "common.about": "Про програму", "common.connect": "Підключити", "common.disconnect": "Відключити", "common.up": "Вгору", "common.folder": "Папка…", "common.refresh": "Оновити", "common.new_folder": "Нова папка", "common.rename": "Перейменувати", "common.delete": "Видалити", "common.permissions": "Дозволи", "common.cancel": "Скасувати",
		"section.local": "ЛОКАЛЬНИЙ КОМП'ЮТЕР", "section.remote": "СЕРВЕР", "section.transfers": "ПЕРЕДАВАННЯ",
		"transfer.upload": "Вивантажити", "transfer.download": "Завантажити", "transfer.pause": "Призупинити", "transfer.resume": "Продовжити", "transfer.retry": "Повторити", "transfer.clear": "Очистити завершені",
		"status.ready": "Готово. Виберіть збережений профіль або введіть дані сервера.", "status.queued": "У черзі", "status.running": "Виконується", "status.done": "Завершено", "status.failed": "Помилка", "status.cancelled": "Скасовано", "status.skipped": "Пропущено",
		"cue.host": "FTP/SFTP сервер, напр. ftp.example.com", "cue.user": "Ім'я користувача, може бути user@example.com", "cue.password": "Пароль FTP / SFTP",
		"column.name": "Назва", "column.type": "Тип", "column.size": "Розмір", "column.modified": "Змінено", "column.status": "Стан", "column.progress": "Прогрес", "type.file": "файл", "type.folder": "папка",
		"settings.title": "Ghost FTP — Налаштування", "settings.invalid_value": "Недійсне значення", "settings.parallel": "Паралельні передачі (1–8):", "settings.timeout": "Тайм-аут підключення (5–60 секунд):", "settings.retries": "Автоматично повторити невдалу передачу (0–3 рази):", "settings.retry_delay": "Затримка між автоматичними спробами (1–30 секунд):",
		"about.title": "Про програму", "connection.failed": "Не вдалося підключитися", "error.generic": "Операція не вдалася. Перевірте підключення та спробуйте ще раз.",
	},
	"sv": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Snabb hostinghantering", "badge.connected": "● ANSLUTEN", "badge.disconnected": "● FRÅNKOPPLAD",
		"profile.quick": "Snabbanslutning (utan profil)", "profile.save": "Spara profil", "profile.delete": "Ta bort profil",
		"common.settings": "Inställningar", "common.about": "Om", "common.connect": "Anslut", "common.disconnect": "Koppla från", "common.up": "Upp", "common.folder": "Mapp…", "common.refresh": "Uppdatera", "common.new_folder": "Ny mapp", "common.rename": "Byt namn", "common.delete": "Ta bort", "common.permissions": "Behörigheter", "common.cancel": "Avbryt",
		"section.local": "LOKAL DATOR", "section.remote": "SERVER", "section.transfers": "ÖVERFÖRINGAR",
		"transfer.upload": "Ladda upp", "transfer.download": "Ladda ner", "transfer.pause": "Pausa", "transfer.resume": "Fortsätt", "transfer.retry": "Försök igen", "transfer.clear": "Rensa slutförda",
		"status.ready": "Klar. Välj en sparad profil eller ange serveruppgifter.", "status.queued": "I kö", "status.running": "Pågår", "status.done": "Klar", "status.failed": "Misslyckades", "status.cancelled": "Avbruten", "status.skipped": "Hoppades över",
		"cue.host": "FTP/SFTP-server, t.ex. ftp.example.com", "cue.user": "Användarnamn, kan vara user@example.com", "cue.password": "FTP / SFTP-lösenord",
		"column.name": "Namn", "column.type": "Typ", "column.size": "Storlek", "column.modified": "Ändrad", "column.status": "Status", "column.progress": "Förlopp", "type.file": "fil", "type.folder": "mapp",
		"settings.title": "Ghost FTP — Inställningar", "settings.invalid_value": "Ogiltigt värde", "settings.parallel": "Parallella överföringar (1–8):", "settings.timeout": "Anslutningstimeout (5–60 sekunder):", "settings.retries": "Försök automatiskt igen vid misslyckad överföring (0–3 gånger):", "settings.retry_delay": "Fördröjning mellan automatiska försök (1–30 sekunder):",
		"about.title": "Om", "connection.failed": "Anslutningen misslyckades", "error.generic": "Åtgärden misslyckades. Kontrollera anslutningen och försök igen.",
	},
	"ro": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Administrare rapidă a găzduirii", "badge.connected": "● CONECTAT", "badge.disconnected": "● DECONECTAT",
		"profile.quick": "Conectare rapidă (fără profil)", "profile.save": "Salvează profilul", "profile.delete": "Șterge profilul",
		"common.settings": "Setări", "common.about": "Despre", "common.connect": "Conectează", "common.disconnect": "Deconectează", "common.up": "Sus", "common.folder": "Dosar…", "common.refresh": "Reîmprospătează", "common.new_folder": "Dosar nou", "common.rename": "Redenumește", "common.delete": "Șterge", "common.permissions": "Permisiuni", "common.cancel": "Anulează",
		"section.local": "CALCULATOR LOCAL", "section.remote": "SERVER", "section.transfers": "TRANSFERURI",
		"transfer.upload": "Încarcă", "transfer.download": "Descarcă", "transfer.pause": "Pauză", "transfer.resume": "Continuă", "transfer.retry": "Reîncearcă", "transfer.clear": "Șterge finalizate",
		"status.ready": "Gata. Selectează un profil salvat sau introdu datele serverului.", "status.queued": "În așteptare", "status.running": "În curs", "status.done": "Finalizat", "status.failed": "Eșuat", "status.cancelled": "Anulat", "status.skipped": "Omis",
		"cue.host": "Server FTP/SFTP, de ex. ftp.example.com", "cue.user": "Nume utilizator, poate fi user@example.com", "cue.password": "Parolă FTP / SFTP",
		"column.name": "Nume", "column.type": "Tip", "column.size": "Dimensiune", "column.modified": "Modificat", "column.status": "Stare", "column.progress": "Progres", "type.file": "fișier", "type.folder": "dosar",
		"settings.title": "Ghost FTP — Setări", "settings.invalid_value": "Valoare invalidă", "settings.parallel": "Transferuri paralele (1–8):", "settings.timeout": "Timp limită conexiune (5–60 secunde):", "settings.retries": "Reîncearcă automat un transfer eșuat (0–3 ori):", "settings.retry_delay": "Întârziere între reîncercări (1–30 secunde):",
		"about.title": "Despre", "connection.failed": "Conexiunea a eșuat", "error.generic": "Operația a eșuat. Verifică conexiunea și încearcă din nou.",
	},
	"hu": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Gyors tárhelykezelés", "badge.connected": "● KAPCSOLÓDVA", "badge.disconnected": "● NINCS KAPCSOLAT",
		"profile.quick": "Gyors kapcsolódás (profil nélkül)", "profile.save": "Profil mentése", "profile.delete": "Profil törlése",
		"common.settings": "Beállítások", "common.about": "Névjegy", "common.connect": "Kapcsolódás", "common.disconnect": "Leválasztás", "common.up": "Fel", "common.folder": "Mappa…", "common.refresh": "Frissítés", "common.new_folder": "Új mappa", "common.rename": "Átnevezés", "common.delete": "Törlés", "common.permissions": "Jogosultságok", "common.cancel": "Mégse",
		"section.local": "HELYI SZÁMÍTÓGÉP", "section.remote": "SZERVER", "section.transfers": "ÁTVITELEK",
		"transfer.upload": "Feltöltés", "transfer.download": "Letöltés", "transfer.pause": "Szünet", "transfer.resume": "Folytatás", "transfer.retry": "Újra", "transfer.clear": "Befejezettek törlése",
		"status.ready": "Kész. Válassz mentett profilt vagy add meg a szerver adatait.", "status.queued": "Sorban", "status.running": "Folyamatban", "status.done": "Kész", "status.failed": "Sikertelen", "status.cancelled": "Megszakítva", "status.skipped": "Kihagyva",
		"cue.host": "FTP/SFTP szerver, pl. ftp.example.com", "cue.user": "Felhasználónév, lehet user@example.com", "cue.password": "FTP / SFTP jelszó",
		"column.name": "Név", "column.type": "Típus", "column.size": "Méret", "column.modified": "Módosítva", "column.status": "Állapot", "column.progress": "Folyamat", "type.file": "fájl", "type.folder": "mappa",
		"settings.title": "Ghost FTP — Beállítások", "settings.invalid_value": "Érvénytelen érték", "settings.parallel": "Párhuzamos átvitelek (1–8):", "settings.timeout": "Kapcsolati időkorlát (5–60 másodperc):", "settings.retries": "Sikertelen átvitel automatikus újrapróbálása (0–3 alkalom):", "settings.retry_delay": "Késleltetés az automatikus próbák között (1–30 másodperc):",
		"about.title": "Névjegy", "connection.failed": "A kapcsolódás sikertelen", "error.generic": "A művelet sikertelen. Ellenőrizd a kapcsolatot, majd próbáld újra.",
	},
	"da": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Hurtig hostingstyring", "badge.connected": "● FORBUNDET", "badge.disconnected": "● AFBRUDT",
		"profile.quick": "Hurtig forbindelse (uden profil)", "profile.save": "Gem profil", "profile.delete": "Slet profil",
		"common.settings": "Indstillinger", "common.about": "Om", "common.connect": "Forbind", "common.disconnect": "Afbryd", "common.up": "Op", "common.folder": "Mappe…", "common.refresh": "Opdater", "common.new_folder": "Ny mappe", "common.rename": "Omdøb", "common.delete": "Slet", "common.permissions": "Tilladelser", "common.cancel": "Annuller",
		"section.local": "LOKAL COMPUTER", "section.remote": "SERVER", "section.transfers": "OVERFØRSLER",
		"transfer.upload": "Upload", "transfer.download": "Download", "transfer.pause": "Pause", "transfer.resume": "Fortsæt", "transfer.retry": "Prøv igen", "transfer.clear": "Ryd færdige",
		"status.ready": "Klar. Vælg en gemt profil eller indtast serveroplysninger.", "status.queued": "I kø", "status.running": "Kører", "status.done": "Færdig", "status.failed": "Mislykkedes", "status.cancelled": "Annulleret", "status.skipped": "Sprunget over",
		"cue.host": "FTP/SFTP-server, f.eks. ftp.example.com", "cue.user": "Brugernavn, kan være user@example.com", "cue.password": "FTP / SFTP-adgangskode",
		"column.name": "Navn", "column.type": "Type", "column.size": "Størrelse", "column.modified": "Ændret", "column.status": "Status", "column.progress": "Fremdrift", "type.file": "fil", "type.folder": "mappe",
		"settings.title": "Ghost FTP — Indstillinger", "settings.invalid_value": "Ugyldig værdi", "settings.parallel": "Parallelle overførsler (1–8):", "settings.timeout": "Forbindelsestimeout (5–60 sekunder):", "settings.retries": "Prøv automatisk en mislykket overførsel igen (0–3 gange):", "settings.retry_delay": "Forsinkelse mellem automatiske forsøg (1–30 sekunder):",
		"about.title": "Om", "connection.failed": "Forbindelsen mislykkedes", "error.generic": "Handlingen mislykkedes. Kontrollér forbindelsen, og prøv igen.",
	},
	"fi": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Nopea palvelinhallinta", "badge.connected": "● YHDISTETTY", "badge.disconnected": "● EI YHTEYTTÄ",
		"profile.quick": "Pikayhteys (ei profiilia)", "profile.save": "Tallenna profiili", "profile.delete": "Poista profiili",
		"common.settings": "Asetukset", "common.about": "Tietoja", "common.connect": "Yhdistä", "common.disconnect": "Katkaise yhteys", "common.up": "Ylös", "common.folder": "Kansio…", "common.refresh": "Päivitä", "common.new_folder": "Uusi kansio", "common.rename": "Nimeä uudelleen", "common.delete": "Poista", "common.permissions": "Oikeudet", "common.cancel": "Peruuta",
		"section.local": "PAIKALLINEN TIETOKONE", "section.remote": "PALVELIN", "section.transfers": "SIIRROT",
		"transfer.upload": "Lähetä", "transfer.download": "Lataa", "transfer.pause": "Tauko", "transfer.resume": "Jatka", "transfer.retry": "Yritä uudelleen", "transfer.clear": "Tyhjennä valmiit",
		"status.ready": "Valmis. Valitse tallennettu profiili tai anna palvelimen tiedot.", "status.queued": "Jonossa", "status.running": "Käynnissä", "status.done": "Valmis", "status.failed": "Epäonnistui", "status.cancelled": "Peruutettu", "status.skipped": "Ohitettu",
		"cue.host": "FTP/SFTP-palvelin, esim. ftp.example.com", "cue.user": "Käyttäjänimi, voi olla user@example.com", "cue.password": "FTP / SFTP-salasana",
		"column.name": "Nimi", "column.type": "Tyyppi", "column.size": "Koko", "column.modified": "Muokattu", "column.status": "Tila", "column.progress": "Edistyminen", "type.file": "tiedosto", "type.folder": "kansio",
		"settings.title": "Ghost FTP — Asetukset", "settings.invalid_value": "Virheellinen arvo", "settings.parallel": "Rinnakkaiset siirrot (1–8):", "settings.timeout": "Yhteyden aikakatkaisu (5–60 sekuntia):", "settings.retries": "Yritä epäonnistunutta siirtoa automaattisesti uudelleen (0–3 kertaa):", "settings.retry_delay": "Viive automaattisten yritysten välillä (1–30 sekuntia):",
		"about.title": "Tietoja", "connection.failed": "Yhteys epäonnistui", "error.generic": "Toiminto epäonnistui. Tarkista yhteys ja yritä uudelleen.",
	},
	"no": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Rask hostingadministrasjon", "badge.connected": "● TILKOBLET", "badge.disconnected": "● FRAKOBLET",
		"profile.quick": "Hurtigtilkobling (uten profil)", "profile.save": "Lagre profil", "profile.delete": "Slett profil",
		"common.settings": "Innstillinger", "common.about": "Om", "common.connect": "Koble til", "common.disconnect": "Koble fra", "common.up": "Opp", "common.folder": "Mappe…", "common.refresh": "Oppdater", "common.new_folder": "Ny mappe", "common.rename": "Gi nytt navn", "common.delete": "Slett", "common.permissions": "Tillatelser", "common.cancel": "Avbryt",
		"section.local": "LOKAL DATAMASKIN", "section.remote": "SERVER", "section.transfers": "OVERFØRINGER",
		"transfer.upload": "Last opp", "transfer.download": "Last ned", "transfer.pause": "Pause", "transfer.resume": "Fortsett", "transfer.retry": "Prøv igjen", "transfer.clear": "Tøm fullførte",
		"status.ready": "Klar. Velg en lagret profil eller skriv inn serverdetaljer.", "status.queued": "I kø", "status.running": "Kjører", "status.done": "Fullført", "status.failed": "Mislyktes", "status.cancelled": "Avbrutt", "status.skipped": "Hoppet over",
		"cue.host": "FTP/SFTP-server, f.eks. ftp.example.com", "cue.user": "Brukernavn, kan være user@example.com", "cue.password": "FTP / SFTP-passord",
		"column.name": "Navn", "column.type": "Type", "column.size": "Størrelse", "column.modified": "Endret", "column.status": "Status", "column.progress": "Fremdrift", "type.file": "fil", "type.folder": "mappe",
		"settings.title": "Ghost FTP — Innstillinger", "settings.invalid_value": "Ugyldig verdi", "settings.parallel": "Parallelle overføringer (1–8):", "settings.timeout": "Tidsavbrudd for tilkobling (5–60 sekunder):", "settings.retries": "Prøv automatisk en mislykket overføring igjen (0–3 ganger):", "settings.retry_delay": "Forsinkelse mellom automatiske forsøk (1–30 sekunder):",
		"about.title": "Om", "connection.failed": "Tilkoblingen mislyktes", "error.generic": "Operasjonen mislyktes. Kontroller tilkoblingen og prøv igjen.",
	},
	"ko": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  빠른 호스팅 관리", "badge.connected": "● 연결됨", "badge.disconnected": "● 연결 끊김",
		"profile.quick": "빠른 연결 (프로필 없음)", "profile.save": "프로필 저장", "profile.delete": "프로필 삭제",
		"common.settings": "설정", "common.about": "정보", "common.connect": "연결", "common.disconnect": "연결 해제", "common.up": "위로", "common.folder": "폴더…", "common.refresh": "새로 고침", "common.new_folder": "새 폴더", "common.rename": "이름 바꾸기", "common.delete": "삭제", "common.permissions": "권한", "common.cancel": "취소",
		"section.local": "로컬 컴퓨터", "section.remote": "서버", "section.transfers": "전송",
		"transfer.upload": "업로드", "transfer.download": "다운로드", "transfer.pause": "일시 중지", "transfer.resume": "계속", "transfer.retry": "다시 시도", "transfer.clear": "완료 항목 지우기",
		"status.ready": "준비됨. 저장된 프로필을 선택하거나 서버 정보를 입력하세요.", "status.queued": "대기 중", "status.running": "진행 중", "status.done": "완료", "status.failed": "실패", "status.cancelled": "취소됨", "status.skipped": "건너뜀",
		"cue.host": "FTP/SFTP 서버, 예: ftp.example.com", "cue.user": "사용자 이름, user@example.com 형식 가능", "cue.password": "FTP / SFTP 비밀번호",
		"column.name": "이름", "column.type": "유형", "column.size": "크기", "column.modified": "수정됨", "column.status": "상태", "column.progress": "진행률", "type.file": "파일", "type.folder": "폴더",
		"settings.title": "Ghost FTP — 설정", "settings.invalid_value": "잘못된 값", "settings.parallel": "동시 전송 (1–8):", "settings.timeout": "연결 제한 시간 (5–60초):", "settings.retries": "실패한 전송 자동 재시도 (0–3회):", "settings.retry_delay": "자동 재시도 간 지연 (1–30초):",
		"about.title": "정보", "connection.failed": "연결 실패", "error.generic": "작업에 실패했습니다. 연결을 확인하고 다시 시도하세요.",
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

	codes := make([]string, 0, len(supportedLanguages))
	for _, language := range supportedLanguages {
		codes = append(codes, language.Code)
	}
	usage := "Usage: language <" + strings.Join(codes, "|") + ">"
	for _, catalog := range catalogs {
		if _, ok := catalog["terminal.language_usage"]; ok {
			catalog["terminal.language_usage"] = usage
		}
	}
}
