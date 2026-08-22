//go:build windows

package desktop

import "brendigo.com/byftp/internal/i18n"

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
