package i18n

var catalogs = map[string]map[string]string{
	"en": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Fast hosting management  ·  Brendigo",
		"badge.connected": "● CONNECTED", "badge.disconnected": "● DISCONNECTED",
		"profile.quick": "Quick connect (no profile)", "profile.save": "Save profile", "profile.delete": "Delete profile",
		"common.settings": "Settings", "common.about": "About", "common.connect": "Connect", "common.disconnect": "Disconnect",
		"common.up": "Up", "common.folder": "Folder…", "common.refresh": "Refresh", "common.new_folder": "New folder",
		"common.rename": "Rename", "common.delete": "Delete", "common.permissions": "Permissions", "common.cancel": "Cancel",
		"auth.private_key": "Private key…", "section.local": "LOCAL COMPUTER", "section.remote": "SERVER", "section.transfers": "TRANSFERS",
		"transfer.upload": "Upload", "transfer.download": "Download", "transfer.pause": "Pause", "transfer.resume": "Resume",
		"transfer.retry": "Retry", "transfer.clear": "Clear finished",
		"status.ready": "Ready. Select a saved profile or enter server details.",
		"status.refresh_all": "Refreshing local and remote views…",
		"cue.host": "FTP/SFTP server, e.g. ftp.example.com", "cue.user": "Username, may be user@example.com",
		"cue.password": "FTP / SFTP password", "cue.private_key": "Private key for SFTP (optional)", "cue.passphrase": "Private-key passphrase",
		"cue.saved_password": "Saved password — leave empty to use it", "cue.saved_passphrase": "Saved key passphrase — leave empty to use it",
		"column.name": "Name", "column.type": "Type", "column.size": "Size", "column.modified": "Modified", "column.direction": "Direction",
		"column.local": "Local", "column.remote": "Server", "column.status": "Status", "column.progress": "Progress",
		"type.file": "file", "type.folder": "folder", "type.link": "link",
		"direction.upload": "Upload", "direction.download": "Download",
		"status.queued": "Queued", "status.running": "Running", "status.done": "Done", "status.failed": "Failed", "status.cancelled": "Cancelled", "status.skipped": "Skipped",
		"transfer.summary": "%d active  •  %d queued  •  %d completed", "transfer.summary_skipped": "  •  %d skipped", "transfer.summary_failed": "  •  %d failed/cancelled",
		"settings.load_failed": "Settings could not be loaded; safe defaults are being used.", "settings.title": "ByFTP — Settings",
		"settings.invalid_value": "Invalid value", "settings.enter_range": "Enter a number from %d to %d.",
		"settings.parallel": "Parallel transfers (1–8):", "settings.timeout": "Connection timeout (5–60 seconds):",
		"settings.retries": "Automatically retry a failed transfer (0–3 times):", "settings.retry_delay": "Delay between automatic retries (1–30 seconds):",
		"settings.backup_title": "Backup before overwrite?", "settings.backup_body": "Yes = keep a backup of the existing file.\nNo = remove the temporary safety backup after a successful transfer.",
		"settings.skip_title": "Skip files that already exist?", "settings.skip_body": "Yes = existing destination files will not be overwritten.\nNo = ByFTP will safely replace them according to the backup setting.",
		"settings.confirm_delete_title": "Confirm before deleting?", "settings.confirm_delete_body": "Keeping this enabled is recommended for local and remote files.",
		"settings.save_failed": "Settings were not saved", "settings.save_failed_body": "Settings cannot be saved right now.",
		"settings.retry_none": "no automatic retries", "settings.retry_count": "automatic retries: %d",
		"settings.overwrite": "overwrite enabled", "settings.skip_existing": "existing files are skipped",
		"settings.saved": "Settings saved. Parallel transfers: %d • connection: %d s • %s • %s",
		"about.title": "About", "about.heading": "Secure and simple FTP, FTPS and SFTP file transfer.",
		"about.body": "ByFTP does not track users and does not send entered data to third parties.\nSaved profiles stay on this computer and are protected by Windows.\n\nBrendigo\n%s\n%s",
		"connection.invalid_port": "Invalid port", "connection.invalid_port_body": "Port must be a number between 1 and 65535.",
		"connection.invalid_data": "Invalid connection details", "connection.invalid_data_body": "Check the server, port and username.",
		"connection.connecting": "Connecting to %s…", "connection.failed": "Connection failed", "connection.failed_status": "Connection failed. Check the details and try again.",
		"connection.failed_body": "Check login details and the network connection.",
		"sftp.security": "ByFTP — SFTP security", "sftp.new_key": "New SFTP server key",
		"sftp.trust_body": "Verify the server key fingerprint before accepting it:\n\n%s\n\nDo you trust this server?", "sftp.cancelled": "SFTP connection cancelled.",
		"sftp.verifying": "Verifying SFTP key and connecting…", "sftp.failed_status": "SFTP connection failed.", "sftp.failed_body": "Check login details and SFTP settings.",
		"connection.connected": "Connected: %s", "disconnect.title": "ByFTP — Disconnect", "disconnect.question": "Disconnect and cancel active transfers?",
		"disconnect.body": "All queued or running transfers will be cancelled before disconnecting.", "disconnect.progress": "Disconnecting…",
		"disconnect.warning": "The connection closed with a warning.", "disconnect.done": "Disconnected.",
		"key.choose_failed": "Could not select private key", "key.choose_failed_body": "The private key cannot be selected right now.",
		"profile.load_failed": "Saved profiles cannot be loaded right now.",
		"error.session_closing": "The previous connection is still closing safely. Try again in a few moments.",
		"error.disconnect_closing": "The connection is still closing safely. Reconnecting will be available when the old session closes.",
		"error.cancelled": "The connection or operation was cancelled.", "error.timeout": "The server did not respond in time. Check the address, port and network connection, then try again.",
		"error.hostkey_changed": "The server security key changed. The connection was blocked for your protection.",
		"error.sftp_unavailable": "SFTP support is unavailable. Enable the OpenSSH Client Windows feature.",
		"error.sftp_hostkey_missing": "The SFTP server did not return a security host key. Check the address, port and whether SSH/SFTP is running.",
		"error.auth": "Login was rejected. Check the full username and password; on shared hosting the FTP username is often user@example.com.",
		"error.ftp_limit": "The FTP server is not accepting a new session or the connection limit was reached. Close other FTP sessions and try again.",
		"error.ftp_data": "The FTP data connection could not be opened or was interrupted. Check the firewall/network and passive FTP ports.",
		"error.resolve": "The server was not found. Check the server address.", "error.refused": "The server refused the connection on the selected port. Check the protocol and port.",
		"error.connection_lost": "The server connection was interrupted. Reconnect and repeat the operation.",
		"error.tls": "The secure connection could not be verified. Check the certificate and FTPS server settings.",
		"error.disk": "There is not enough space to complete the operation.", "error.permission": "You do not have permission for this file or folder.",
		"error.not_found": "The file or folder is no longer available. Refresh the view and try again.", "error.exists": "An item with that name already exists.",
		"error.not_connected": "You are not connected to the server.", "error.queue_full": "The transfer queue is full. Clear completed transfers and try again.",
		"error.structure_large": "The selected structure is too large for one safe operation.", "error.invalid_port": "Port must be a number between 1 and 65535.",
		"error.invalid_host": "The server address is invalid.", "error.invalid_user": "The username is invalid.", "error.invalid_name": "The file or folder name is not allowed.",
		"error.generic": "The operation failed. Check the connection and try again.",
		"terminal.secret_hide_failed": "Unable to safely hide credential input in the terminal", "terminal.protocol": "Protocol (ftp/ftps/ftpsi/sftp)",
		"terminal.server": "Server", "terminal.port": "Port", "terminal.username": "Username", "terminal.private_key": "Private key",
		"terminal.key_passphrase": "Private-key passphrase (Enter if none)", "terminal.password": "Password",
		"terminal.port_number": "port must be a number", "terminal.sftp_key_required": "Linux/macOS SFTP requires an explicit private key",
		"terminal.sftp_passphrase_unsupported": "Linux/macOS SFTP currently supports only private keys without a passphrase",
		"terminal.fingerprint": "SFTP host-key fingerprint:", "terminal.trust": "Do you trust this server? (yes/no)",
		"terminal.server_not_connected": "the server did not confirm an active connection", "terminal.title": "ByFTP %s — %s terminal client",
		"terminal.privacy": "Telemetry-free FTP/FTPS/SFTP client. Secrets are never printed to the terminal.",
		"terminal.connect_failed": "Connection failed. Check the details and try again.", "terminal.connected": "Connected: %s:%d",
		"terminal.commands": "Commands: ls, cd, mkdir, rename, delete, chmod, get, put, pwd, language, help, quit",
		"terminal.quote_paths": "Put paths and names containing spaces in single or double quotes.", "terminal.invalid_command": "Invalid command: %s",
		"terminal.unknown_command": "Unknown command. Type 'help'.", "terminal.transfer_done": "Transfer completed.",
		"terminal.transfer_skipped": "the transfer was skipped because the destination already exists", "terminal.transfer_missing": "the transfer is no longer available in the queue",
		"terminal.transfer_failed": "the transfer failed", "terminal.items": "%d items", "terminal.item_file": "file", "terminal.item_folder": "dir", "terminal.item_link": "link",
		"terminal.language_usage": "Usage: language <en|hr|de|fr|es|tr|el|pt|zh|ru|hi|ja>", "terminal.language_saved": "Language changed to %s.",
	},
	"hr": {
		"app.subtitle": "FTP • FTPS • SFTP  ·  Brzo upravljanje hostingom  ·  Brendigo",
		"badge.connected": "● POVEZANO", "badge.disconnected": "● NIJE POVEZANO", "profile.quick": "Brzi spoj (bez profila)", "profile.save": "Spremi profil", "profile.delete": "Obriši profil",
		"common.settings": "Postavke", "common.about": "O programu", "common.connect": "Poveži", "common.disconnect": "Prekini", "common.up": "Gore", "common.folder": "Mapa…", "common.refresh": "Osvježi", "common.new_folder": "Nova mapa", "common.rename": "Preimenuj", "common.delete": "Obriši", "common.permissions": "Dozvole", "common.cancel": "Otkaži",
		"auth.private_key": "Privatni ključ…", "section.local": "LOKALNO RAČUNALO", "section.remote": "POSLUŽITELJ", "section.transfers": "PRIJENOSI", "transfer.upload": "Pošalji", "transfer.download": "Preuzmi", "transfer.pause": "Pauziraj", "transfer.resume": "Nastavi", "transfer.retry": "Ponovi", "transfer.clear": "Očisti završene",
		"status.ready": "Spremno. Odaberite spremljeni profil ili unesite podatke poslužitelja.", "status.refresh_all": "Osvježavanje lokalnog i udaljenog prikaza…",
		"cue.host": "FTP/SFTP poslužitelj, npr. ftp.domena.hr", "cue.user": "Korisničko ime, može korisnik@domena", "cue.password": "FTP / SFTP lozinka", "cue.private_key": "Privatni ključ za SFTP (opcionalno)", "cue.passphrase": "Zaporka privatnog ključa", "cue.saved_password": "Spremljena lozinka — prazno koristi spremljenu", "cue.saved_passphrase": "Spremljena zaporka ključa — prazno koristi spremljenu",
		"column.name": "Naziv", "column.type": "Vrsta", "column.size": "Veličina", "column.modified": "Izmijenjeno", "column.direction": "Smjer", "column.local": "Lokalno", "column.remote": "Poslužitelj", "column.status": "Status", "column.progress": "Napredak", "type.file": "datoteka", "type.folder": "mapa", "type.link": "poveznica", "direction.upload": "Slanje", "direction.download": "Preuzimanje", "status.queued": "Na čekanju", "status.running": "U tijeku", "status.done": "Završeno", "status.failed": "Greška", "status.cancelled": "Otkazano", "status.skipped": "Preskočeno",
		"transfer.summary": "%d aktivnih  •  %d na čekanju  •  %d završeno", "transfer.summary_skipped": "  •  %d preskočeno", "transfer.summary_failed": "  •  %d greška/otkazano",
		"settings.load_failed": "Postavke nisu učitane; koriste se sigurne zadane vrijednosti.", "settings.title": "ByFTP — Postavke", "settings.invalid_value": "Neispravna vrijednost", "settings.enter_range": "Unesite broj od %d do %d.", "settings.parallel": "Broj paralelnih prijenosa (1–8):", "settings.timeout": "Vrijeme čekanja pri spajanju (5–60 sekundi):", "settings.retries": "Automatski ponoviti neuspjeli prijenos (0–3 puta):", "settings.retry_delay": "Pauza između automatskih pokušaja (1–30 sekundi):", "settings.backup_title": "Sigurnosna kopija prije prepisivanja?", "settings.backup_body": "Da = ByFTP zadržava sigurnosnu kopiju postojeće datoteke.\nNe = privremena zaštitna kopija uklanja se nakon uspješnog prijenosa.", "settings.skip_title": "Preskočiti datoteke koje već postoje?", "settings.skip_body": "Da = postojeće odredišne datoteke neće se prepisivati.\nNe = ByFTP će ih sigurno zamijeniti prema postavci sigurnosne kopije.", "settings.confirm_delete_title": "Tražiti potvrdu prije brisanja?", "settings.confirm_delete_body": "Preporučeno je ostaviti ovu opciju uključenu za lokalne i udaljene datoteke.", "settings.save_failed": "Postavke nisu spremljene", "settings.save_failed_body": "Postavke trenutačno nije moguće spremiti.", "settings.retry_none": "bez automatskog ponavljanja", "settings.retry_count": "automatska ponavljanja: %d", "settings.overwrite": "prepisivanje uključeno", "settings.skip_existing": "postojeće datoteke se preskaču", "settings.saved": "Postavke spremljene. Paralelni prijenosi: %d • spajanje: %d s • %s • %s",
		"about.title": "O programu", "about.heading": "Siguran i jednostavan prijenos datoteka putem FTP, FTPS i SFTP veze.", "about.body": "ByFTP ne prati korisnika i ne šalje unesene podatke trećim stranama.\nSpremljeni profili ostaju samo na ovom računalu i zaštićeni su sustavom Windows.\n\nBrendigo\n%s\n%s",
		"connection.invalid_port": "Neispravan port", "connection.invalid_port_body": "Port mora biti broj između 1 i 65535.", "connection.invalid_data": "Neispravni podaci veze", "connection.invalid_data_body": "Provjerite poslužitelj, port i korisničko ime.", "connection.connecting": "Povezivanje s %s…", "connection.failed": "Povezivanje nije uspjelo", "connection.failed_status": "Povezivanje nije uspjelo. Provjerite podatke i pokušajte ponovno.", "connection.failed_body": "Provjerite podatke za prijavu i mrežnu vezu.", "sftp.security": "ByFTP — SFTP sigurnost", "sftp.new_key": "Novi SFTP ključ poslužitelja", "sftp.trust_body": "Provjerite otisak sigurnosnog ključa prije prihvaćanja:\n\n%s\n\nVjerujete li ovom poslužitelju?", "sftp.cancelled": "SFTP povezivanje otkazano.", "sftp.verifying": "Provjera SFTP ključa i povezivanje…", "sftp.failed_status": "SFTP povezivanje nije uspjelo.", "sftp.failed_body": "Provjerite podatke za prijavu i SFTP postavke.", "connection.connected": "Povezano: %s", "disconnect.title": "ByFTP — Prekid veze", "disconnect.question": "Prekinuti vezu i aktivne prijenose?", "disconnect.body": "Svi prijenosi koji su na čekanju ili u tijeku bit će otkazani prije prekida veze.", "disconnect.progress": "Prekid veze…", "disconnect.warning": "Veza je zatvorena uz upozorenje.", "disconnect.done": "Veza prekinuta.", "key.choose_failed": "Odabir privatnog ključa nije uspio", "key.choose_failed_body": "Privatni ključ trenutačno nije moguće odabrati.", "profile.load_failed": "Spremljene profile trenutačno nije moguće učitati.",
		"error.session_closing": "Prethodna veza još se sigurno zatvara. Pokušajte ponovno za nekoliko trenutaka.", "error.disconnect_closing": "Prekid veze još se sigurno dovršava. Ponovno povezivanje bit će dostupno čim se stara sesija zatvori.", "error.cancelled": "Povezivanje ili operacija je otkazana.", "error.timeout": "Poslužitelj nije odgovorio na vrijeme. Provjerite adresu, port i mrežnu vezu pa pokušajte ponovno.", "error.hostkey_changed": "Sigurnosni ključ poslužitelja promijenio se. Veza je blokirana radi vaše zaštite.", "error.sftp_unavailable": "SFTP podrška nije dostupna. U postavkama sustava Windows uključite OpenSSH Client.", "error.sftp_hostkey_missing": "SFTP poslužitelj nije vratio sigurnosni host ključ. Provjerite adresu, port i SSH/SFTP servis.", "error.auth": "Prijava nije prihvaćena. Provjerite puni korisnički naziv i lozinku; na shared hostingu FTP korisnik često ima oblik korisnik@domena.", "error.ftp_limit": "FTP poslužitelj ne prihvaća novu sesiju ili je dosegnut limit veza. Zatvorite druge FTP veze i pokušajte ponovno.", "error.ftp_data": "FTP podatkovna veza nije uspostavljena ili je prekinuta. Provjerite firewall/mrežu i pasivne FTP portove.", "error.resolve": "Poslužitelj nije pronađen. Provjerite adresu poslužitelja.", "error.refused": "Poslužitelj odbija vezu na odabranom portu. Provjerite protokol i port.", "error.connection_lost": "Veza s poslužiteljem je prekinuta. Povežite se ponovno i ponovite operaciju.", "error.tls": "Sigurnu vezu nije moguće potvrditi. Provjerite certifikat i FTPS postavke poslužitelja.", "error.disk": "Nema dovoljno prostora za dovršetak operacije.", "error.permission": "Nemate dopuštenje za ovu datoteku ili mapu.", "error.not_found": "Datoteka ili mapa više nije dostupna. Osvježite prikaz i pokušajte ponovno.", "error.exists": "Stavka s tim nazivom već postoji.", "error.not_connected": "Niste povezani s poslužiteljem.", "error.queue_full": "Red prijenosa je pun. Očistite završene prijenose pa pokušajte ponovno.", "error.structure_large": "Odabrana struktura je prevelika za jednu sigurnu operaciju.", "error.invalid_port": "Port mora biti broj između 1 i 65535.", "error.invalid_host": "Adresa poslužitelja nije ispravna.", "error.invalid_user": "Korisničko ime nije ispravno.", "error.invalid_name": "Naziv datoteke ili mape nije dopušten.", "error.generic": "Operacija nije uspjela. Provjerite vezu i pokušajte ponovno.",
		"terminal.secret_hide_failed": "Nije moguće sigurno sakriti unos vjerodajnice u terminalu", "terminal.protocol": "Protokol (ftp/ftps/ftpsi/sftp)", "terminal.server": "Poslužitelj", "terminal.port": "Port", "terminal.username": "Korisničko ime", "terminal.private_key": "Privatni ključ", "terminal.key_passphrase": "Zaporka privatnog ključa (Enter ako je nema)", "terminal.password": "Lozinka", "terminal.port_number": "port mora biti broj", "terminal.sftp_key_required": "Linux/macOS SFTP zahtijeva eksplicitni privatni ključ", "terminal.sftp_passphrase_unsupported": "Linux/macOS SFTP trenutačno podržava privatni ključ bez passphrasea", "terminal.fingerprint": "SFTP host-key fingerprint:", "terminal.trust": "Vjerujete li ovom poslužitelju? (da/ne)", "terminal.server_not_connected": "poslužitelj nije potvrdio aktivnu vezu", "terminal.title": "ByFTP %s — %s terminalni klijent", "terminal.privacy": "FTP/FTPS/SFTP klijent bez telemetrije. Tajne se ne ispisuju u terminal.", "terminal.connect_failed": "Povezivanje nije uspjelo. Provjerite podatke i pokušajte ponovno.", "terminal.connected": "Povezano: %s:%d", "terminal.commands": "Naredbe: ls, cd, mkdir, rename, delete, chmod, get, put, pwd, language, help, quit", "terminal.quote_paths": "Putanje i nazive s razmacima stavite u jednostruke ili dvostruke navodnike.", "terminal.invalid_command": "Neispravna naredba: %s", "terminal.unknown_command": "Nepoznata naredba. Upišite 'help'.", "terminal.transfer_done": "Prijenos dovršen.", "terminal.transfer_skipped": "prijenos je preskočen jer odredište već postoji", "terminal.transfer_missing": "prijenos više nije dostupan u redu", "terminal.transfer_failed": "prijenos nije uspio", "terminal.items": "%d stavki", "terminal.item_file": "dat", "terminal.item_folder": "map", "terminal.item_link": "link", "terminal.language_usage": "Upotreba: language <en|hr|de|fr|es|tr|el|pt|zh|ru|hi|ja>", "terminal.language_saved": "Jezik je promijenjen na %s.",
	},
	"de": makeDE(), "fr": makeFR(), "es": makeES(), "tr": makeTR(), "el": makeEL(), "pt": makePT(), "zh": makeZH(), "ru": makeRU(), "hi": makeHI(), "ja": makeJA(),
}

// Each non-English locale file provides a complete map. Keeping this helper
// independent of catalogs avoids package-initialization cycles; ValidateCatalogs
// enforces exact key parity with the canonical English catalog.
func localizedFromEnglish(overrides map[string]string) map[string]string {
	out := make(map[string]string, len(overrides))
	for key, value := range overrides { out[key] = value }
	return out
}

func makeDE() map[string]string { return localizedFromEnglish(localeDE) }
func makeFR() map[string]string { return localizedFromEnglish(localeFR) }
func makeES() map[string]string { return localizedFromEnglish(localeES) }
func makeTR() map[string]string { return localizedFromEnglish(localeTR) }
func makeEL() map[string]string { return localizedFromEnglish(localeEL) }
func makePT() map[string]string { return localizedFromEnglish(localePT) }
func makeZH() map[string]string { return localizedFromEnglish(localeZH) }
func makeRU() map[string]string { return localizedFromEnglish(localeRU) }
func makeHI() map[string]string { return localizedFromEnglish(localeHI) }
func makeJA() map[string]string { return localizedFromEnglish(localeJA) }
