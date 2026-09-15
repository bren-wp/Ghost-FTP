package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

type applicationNavigationLabels struct {
	SiteManager string
	Diagnostics string
}

var applicationNavigation = map[string]applicationNavigationLabels{
	"en": {"Site Manager", "Connection info"},
	"hr": {"Upravitelj poslužitelja", "Informacije o vezi"},
	"de": {"Serververwaltung", "Verbindungsinfo"},
	"fr": {"Gestionnaire de serveurs", "Infos de connexion"},
	"es": {"Gestor de servidores", "Información de conexión"},
	"tr": {"Sunucu Yöneticisi", "Bağlantı bilgisi"},
	"el": {"Διαχείριση διακομιστών", "Πληροφορίες σύνδεσης"},
	"pt": {"Gestor de servidores", "Informações da ligação"},
	"zh": {"服务器管理器", "连接信息"},
	"ru": {"Менеджер серверов", "Сведения о подключении"},
	"hi": {"सर्वर प्रबंधक", "कनेक्शन जानकारी"},
	"ja": {"サーバーマネージャー", "接続情報"},
	"it": {"Gestione server", "Informazioni connessione"},
	"pl": {"Menedżer serwerów", "Informacje o połączeniu"},
	"nl": {"Serverbeheer", "Verbindingsinfo"},
	"cs": {"Správce serverů", "Informace o připojení"},
	"uk": {"Менеджер серверів", "Відомості про підключення"},
	"sv": {"Serverhanterare", "Anslutningsinfo"},
	"ro": {"Manager servere", "Informații conexiune"},
	"hu": {"Kiszolgálókezelő", "Kapcsolati adatok"},
	"da": {"Serveradministrator", "Forbindelsesinfo"},
	"fi": {"Palvelinten hallinta", "Yhteystiedot"},
	"no": {"Serverbehandling", "Tilkoblingsinfo"},
	"ko": {"서버 관리자", "연결 정보"},
}

func navigationLabelsForLanguage(language string) applicationNavigationLabels {
	if labels, ok := applicationNavigation[i18n.Normalize(language)]; ok {
		return labels
	}
	return applicationNavigation[i18n.DefaultLanguage]
}
