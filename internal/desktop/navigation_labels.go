package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type applicationNavigationLabels struct {
	Files         string
	Connections   string
	TransferQueue string
	SiteManager   string
	Diagnostics   string
}

// applicationNavigation is intentionally independent from the large i18n
// catalog. These short nouns are the native desktop navigation contract and
// must remain compact enough for the fixed-width application rail.
var applicationNavigation = map[string]applicationNavigationLabels{
	"en": {"Files", "Connections", "Transfer Queue", "Site Manager", "Connection info"},
	"hr": {"Datoteke", "Veze", "Red prijenosa", "Upravitelj poslužitelja", "Informacije o vezi"},
	"de": {"Dateien", "Verbindungen", "Übertragungen", "Serververwaltung", "Verbindungsinfo"},
	"fr": {"Fichiers", "Connexions", "Transferts", "Gestionnaire de serveurs", "Infos de connexion"},
	"es": {"Archivos", "Conexiones", "Transferencias", "Gestor de servidores", "Información de conexión"},
	"tr": {"Dosyalar", "Bağlantılar", "Aktarımlar", "Sunucu Yöneticisi", "Bağlantı bilgisi"},
	"el": {"Αρχεία", "Συνδέσεις", "Μεταφορές", "Διαχείριση διακομιστών", "Πληροφορίες σύνδεσης"},
	"pt": {"Ficheiros", "Ligações", "Transferências", "Gestor de servidores", "Informações da ligação"},
	"zh": {"文件", "连接", "传输队列", "服务器管理器", "连接信息"},
	"ru": {"Файлы", "Подключения", "Передачи", "Менеджер серверов", "Сведения о подключении"},
	"hi": {"फ़ाइलें", "कनेक्शन", "स्थानांतरण", "सर्वर प्रबंधक", "कनेक्शन जानकारी"},
	"ja": {"ファイル", "接続", "転送キュー", "サーバーマネージャー", "接続情報"},
	"it": {"File", "Connessioni", "Trasferimenti", "Gestione server", "Informazioni connessione"},
	"pl": {"Pliki", "Połączenia", "Transfery", "Menedżer serwerów", "Informacje o połączeniu"},
	"nl": {"Bestanden", "Verbindingen", "Overdrachten", "Serverbeheer", "Verbindingsinfo"},
	"cs": {"Soubory", "Připojení", "Přenosy", "Správce serverů", "Informace o připojení"},
	"uk": {"Файли", "Підключення", "Передавання", "Менеджер серверів", "Відомості про підключення"},
	"sv": {"Filer", "Anslutningar", "Överföringar", "Serverhanterare", "Anslutningsinfo"},
	"ro": {"Fișiere", "Conexiuni", "Transferuri", "Manager servere", "Informații conexiune"},
	"hu": {"Fájlok", "Kapcsolatok", "Átvitelek", "Kiszolgálókezelő", "Kapcsolati adatok"},
	"da": {"Filer", "Forbindelser", "Overførsler", "Serveradministrator", "Forbindelsesinfo"},
	"fi": {"Tiedostot", "Yhteydet", "Siirrot", "Palvelinten hallinta", "Yhteystiedot"},
	"no": {"Filer", "Tilkoblinger", "Overføringer", "Serverbehandling", "Tilkoblingsinfo"},
	"ko": {"파일", "연결", "전송", "서버 관리자", "연결 정보"},
}

func navigationLabelsForLanguage(language string) applicationNavigationLabels {
	if labels, ok := applicationNavigation[i18n.Normalize(language)]; ok {
		return labels
	}
	return applicationNavigation[i18n.DefaultLanguage]
}
