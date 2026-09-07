//go:build windows

package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

const (
	idSiteManager = 701
	idDiagnostics = 703
)

type applicationNavigationLabels struct {
	SiteManager string
	Diagnostics string
}

var applicationNavigation = map[string]applicationNavigationLabels{
	"en": {"Site Manager", "Diagnostics"},
	"hr": {"Upravitelj poslužitelja", "Dijagnostika"},
	"de": {"Serververwaltung", "Diagnose"},
	"fr": {"Gestionnaire de serveurs", "Diagnostics"},
	"es": {"Gestor de servidores", "Diagnóstico"},
	"tr": {"Sunucu Yöneticisi", "Tanılama"},
	"el": {"Διαχείριση διακομιστών", "Διαγνωστικά"},
	"pt": {"Gestor de servidores", "Diagnóstico"},
	"zh": {"服务器管理器", "诊断"},
	"ru": {"Менеджер серверов", "Диагностика"},
	"hi": {"सर्वर प्रबंधक", "निदान"},
	"ja": {"サーバーマネージャー", "診断"},
	"it": {"Gestione server", "Diagnostica"},
	"pl": {"Menedżer serwerów", "Diagnostyka"},
	"nl": {"Serverbeheer", "Diagnostiek"},
	"cs": {"Správce serverů", "Diagnostika"},
	"uk": {"Менеджер серверів", "Діагностика"},
	"sv": {"Serverhanterare", "Diagnostik"},
	"ro": {"Manager servere", "Diagnosticare"},
	"hu": {"Kiszolgálókezelő", "Diagnosztika"},
	"da": {"Serveradministrator", "Diagnostik"},
	"fi": {"Palvelinten hallinta", "Diagnostiikka"},
	"no": {"Serverbehandling", "Diagnostikk"},
	"ko": {"서버 관리자", "진단"},
}

func navigationLabelsForLanguage(language string) applicationNavigationLabels {
	if labels, ok := applicationNavigation[i18n.Normalize(language)]; ok {
		return labels
	}
	return applicationNavigation[i18n.DefaultLanguage]
}
