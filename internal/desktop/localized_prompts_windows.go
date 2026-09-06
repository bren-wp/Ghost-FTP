//go:build windows

package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

// localizedPrompt follows the canonical i18n language order. Keeping these
// native-dialog strings in the same 24-language contract prevents auxiliary
// Windows surfaces from silently falling back to English for half the catalog.
func localizedPrompt(language string, values [24]string) string {
	language = i18n.Normalize(language)
	for index, item := range i18n.Languages() {
		if item.Code == language {
			if values[index] != "" {
				return values[index]
			}
			break
		}
	}
	return values[0]
}

func okLabel(language string) string {
	return localizedPrompt(language, [24]string{
		"OK", "U redu", "OK", "OK", "Aceptar", "Tamam", "OK", "OK", "确定", "ОК", "ठीक है", "OK",
		"OK", "OK", "OK", "OK", "Гаразд", "OK", "OK", "OK", "OK", "OK", "OK", "확인",
	})
}

func closeQuestion(language string) string {
	return localizedPrompt(language, [24]string{
		"Close Ghost FTP?", "Zatvoriti Ghost FTP?", "Ghost FTP schließen?", "Fermer Ghost FTP ?", "¿Cerrar Ghost FTP?", "Ghost FTP kapatılsın mı?",
		"Κλείσιμο του Ghost FTP;", "Fechar o Ghost FTP?", "关闭 Ghost FTP？", "Закрыть Ghost FTP?", "Ghost FTP बंद करें?", "Ghost FTP を終了しますか？",
		"Chiudere Ghost FTP?", "Zamknąć Ghost FTP?", "Ghost FTP sluiten?", "Zavřít Ghost FTP?", "Закрити Ghost FTP?", "Stäng Ghost FTP?",
		"Închideți Ghost FTP?", "Bezárja a Ghost FTP-t?", "Luk Ghost FTP?", "Suljetaanko Ghost FTP?", "Lukke Ghost FTP?", "Ghost FTP를 닫으시겠습니까?",
	})
}

func closeBody(language string) string {
	return localizedPrompt(language, [24]string{
		"A connection, connection attempt, or transfer is still active. Closing will cancel active operations.",
		"Veza, pokušaj povezivanja ili prijenos još je aktivan. Zatvaranje će otkazati aktivne operacije.",
		"Eine Verbindung, ein Verbindungsversuch oder eine Übertragung ist noch aktiv. Beim Schließen werden aktive Vorgänge abgebrochen.",
		"Une connexion, une tentative de connexion ou un transfert est encore actif. La fermeture annulera les opérations actives.",
		"Hay una conexión, un intento de conexión o una transferencia activa. Al cerrar se cancelarán las operaciones activas.",
		"Bir bağlantı, bağlantı denemesi veya aktarım hâlâ etkin. Kapatma işlemi etkin işlemleri iptal eder.",
		"Μια σύνδεση, προσπάθεια σύνδεσης ή μεταφορά είναι ακόμη ενεργή. Το κλείσιμο θα ακυρώσει τις ενεργές λειτουργίες.",
		"Ainda existe uma ligação, tentativa de ligação ou transferência ativa. Ao fechar, as operações ativas serão canceladas.",
		"仍有连接、连接尝试或传输处于活动状态。关闭应用将取消活动操作。",
		"Соединение, попытка подключения или передача всё ещё активны. При закрытии активные операции будут отменены.",
		"कनेक्शन, कनेक्शन प्रयास या ट्रांसफ़र अभी सक्रिय है। बंद करने पर सक्रिय ऑपरेशन रद्द हो जाएंगे।",
		"接続、接続試行、または転送がまだ実行中です。終了すると実行中の操作はキャンセルされます。",
		"Una connessione, un tentativo di connessione o un trasferimento è ancora attivo. La chiusura annullerà le operazioni attive.",
		"Połączenie, próba połączenia lub transfer jest nadal aktywny. Zamknięcie anuluje aktywne operacje.",
		"Er is nog een verbinding, verbindingspoging of overdracht actief. Sluiten annuleert de actieve bewerkingen.",
		"Připojení, pokus o připojení nebo přenos je stále aktivní. Zavřením se aktivní operace zruší.",
		"З’єднання, спроба з’єднання або передавання ще активні. Закриття скасує активні операції.",
		"En anslutning, ett anslutningsförsök eller en överföring är fortfarande aktiv. Stängning avbryter aktiva åtgärder.",
		"O conexiune, o încercare de conectare sau un transfer este încă activ. Închiderea va anula operațiunile active.",
		"Egy kapcsolat, kapcsolódási kísérlet vagy átvitel még aktív. A bezárás megszakítja az aktív műveleteket.",
		"En forbindelse, et forbindelsesforsøg eller en overførsel er stadig aktiv. Lukning annullerer aktive handlinger.",
		"Yhteys, yhteysyritys tai siirto on yhä aktiivinen. Sulkeminen peruuttaa aktiiviset toiminnot.",
		"En tilkobling, et tilkoblingsforsøk eller en overføring er fortsatt aktiv. Lukking avbryter aktive operasjoner.",
		"연결, 연결 시도 또는 전송이 아직 진행 중입니다. 닫으면 진행 중인 작업이 취소됩니다.",
	})
}

func privateKeyDialogTitle(language string) string {
	return localizedPrompt(language, [24]string{
		"Select SSH private key", "Odaberi SSH privatni ključ", "SSH-Privatschlüssel auswählen", "Sélectionner la clé privée SSH",
		"Seleccionar clave privada SSH", "SSH özel anahtarını seç", "Επιλογή ιδιωτικού κλειδιού SSH", "Selecionar chave privada SSH",
		"选择 SSH 私钥", "Выберите закрытый ключ SSH", "SSH निजी कुंजी चुनें", "SSH 秘密鍵を選択",
		"Seleziona chiave privata SSH", "Wybierz klucz prywatny SSH", "SSH-privésleutel selecteren", "Vyberte soukromý klíč SSH",
		"Виберіть приватний ключ SSH", "Välj privat SSH-nyckel", "Selectați cheia privată SSH", "SSH privát kulcs kiválasztása",
		"Vælg privat SSH-nøgle", "Valitse SSH-yksityisavain", "Velg privat SSH-nøkkel", "SSH 개인 키 선택",
	})
}

func privateKeyFilterLabel(language string) string {
	return localizedPrompt(language, [24]string{
		"SSH private keys", "SSH privatni ključevi", "SSH-Privatschlüssel", "Clés privées SSH", "Claves privadas SSH", "SSH özel anahtarları",
		"Ιδιωτικά κλειδιά SSH", "Chaves privadas SSH", "SSH 私钥", "Закрытые ключи SSH", "SSH निजी कुंजियाँ", "SSH 秘密鍵",
		"Chiavi private SSH", "Klucze prywatne SSH", "SSH-privésleutels", "Soukromé klíče SSH", "Приватні ключі SSH", "Privata SSH-nycklar",
		"Chei private SSH", "SSH privát kulcsok", "Private SSH-nøgler", "SSH-yksityisavaimet", "Private SSH-nøkler", "SSH 개인 키",
	})
}

func allFilesFilterLabel(language string) string {
	return localizedPrompt(language, [24]string{
		"All files", "Sve datoteke", "Alle Dateien", "Tous les fichiers", "Todos los archivos", "Tüm dosyalar", "Όλα τα αρχεία",
		"Todos os ficheiros", "所有文件", "Все файлы", "सभी फ़ाइलें", "すべてのファイル",
		"Tutti i file", "Wszystkie pliki", "Alle bestanden", "Všechny soubory", "Усі файли", "Alla filer", "Toate fișierele", "Minden fájl",
		"Alle filer", "Kaikki tiedostot", "Alle filer", "모든 파일",
	})
}

func directoryDialogTitle(language string) string {
	return localizedPrompt(language, [24]string{
		"Select local folder for Ghost FTP", "Odaberi lokalnu mapu za Ghost FTP", "Lokalen Ordner für Ghost FTP auswählen", "Sélectionner le dossier local pour Ghost FTP",
		"Seleccionar carpeta local para Ghost FTP", "Ghost FTP için yerel klasör seç", "Επιλογή τοπικού φακέλου για το Ghost FTP", "Selecionar pasta local para o Ghost FTP",
		"选择 Ghost FTP 本地文件夹", "Выберите локальную папку для Ghost FTP", "Ghost FTP के लिए स्थानीय फ़ोल्डर चुनें", "Ghost FTP のローカルフォルダーを選択",
		"Seleziona cartella locale per Ghost FTP", "Wybierz lokalny folder dla Ghost FTP", "Lokale map voor Ghost FTP selecteren", "Vyberte místní složku pro Ghost FTP",
		"Виберіть локальну папку для Ghost FTP", "Välj lokal mapp för Ghost FTP", "Selectați folderul local pentru Ghost FTP", "Helyi mappa kiválasztása a Ghost FTP számára",
		"Vælg lokal mappe til Ghost FTP", "Valitse Ghost FTP:n paikallinen kansio", "Velg lokal mappe for Ghost FTP", "Ghost FTP의 로컬 폴더 선택",
	})
}
