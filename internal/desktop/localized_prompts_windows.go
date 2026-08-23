//go:build windows

package desktop

import "github.com/bren-wp/by-ftp/internal/i18n"

func localizedPair(language string, values [12]string) string {
	codes := []string{"en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja"}
	language = i18n.Normalize(language)
	for index, code := range codes {
		if code == language {
			return values[index]
		}
	}
	return values[0]
}

func okLabel(language string) string {
	return localizedPair(language, [12]string{
		"OK", "U redu", "OK", "OK", "Aceptar", "Tamam", "OK", "OK", "确定", "ОК", "ठीक है", "OK",
	})
}

func closeQuestion(language string) string {
	return localizedPair(language, [12]string{
		"Close ByFTP?", "Zatvoriti ByFTP?", "ByFTP schließen?", "Fermer ByFTP ?", "¿Cerrar ByFTP?", "ByFTP kapatılsın mı?",
		"Κλείσιμο του ByFTP;", "Fechar o ByFTP?", "关闭 ByFTP？", "Закрыть ByFTP?", "ByFTP बंद करें?", "ByFTP を終了しますか？",
	})
}

func closeBody(language string) string {
	return localizedPair(language, [12]string{
		"A connection, connection attempt, or transfer is still active. Closing will cancel active operations.",
		"Veza, povezivanje ili prijenosi još su aktivni. Zatvaranje će prekinuti aktivne operacije.",
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
	})
}

func privateKeyDialogTitle(language string) string {
	return localizedPair(language, [12]string{
		"Select SSH private key", "Odaberi SSH privatni ključ", "SSH-Privatschlüssel auswählen", "Sélectionner la clé privée SSH",
		"Seleccionar clave privada SSH", "SSH özel anahtarını seç", "Επιλογή ιδιωτικού κλειδιού SSH", "Selecionar chave privada SSH",
		"选择 SSH 私钥", "Выберите закрытый ключ SSH", "SSH निजी कुंजी चुनें", "SSH 秘密鍵を選択",
	})
}

func privateKeyFilterLabel(language string) string {
	return localizedPair(language, [12]string{
		"SSH private keys", "SSH privatni ključevi", "SSH-Privatschlüssel", "Clés privées SSH", "Claves privadas SSH", "SSH özel anahtarları",
		"Ιδιωτικά κλειδιά SSH", "Chaves privadas SSH", "SSH 私钥", "Закрытые ключи SSH", "SSH निजी कुंजियाँ", "SSH 秘密鍵",
	})
}

func allFilesFilterLabel(language string) string {
	return localizedPair(language, [12]string{
		"All files", "Sve datoteke", "Alle Dateien", "Tous les fichiers", "Todos los archivos", "Tüm dosyalar", "Όλα τα αρχεία",
		"Todos os ficheiros", "所有文件", "Все файлы", "सभी फ़ाइलें", "すべてのファイル",
	})
}

func directoryDialogTitle(language string) string {
	return localizedPair(language, [12]string{
		"Select local folder for ByFTP", "Odaberi lokalnu mapu za ByFTP", "Lokalen Ordner für ByFTP auswählen", "Sélectionner le dossier local pour ByFTP",
		"Seleccionar carpeta local para ByFTP", "ByFTP için yerel klasör seç", "Επιλογή τοπικού φακέλου για το ByFTP", "Selecionar pasta local para o ByFTP",
		"选择 ByFTP 本地文件夹", "Выберите локальную папку для ByFTP", "ByFTP के लिए स्थानीय फ़ोल्डर चुनें", "ByFTP のローカルフォルダーを選択",
	})
}
