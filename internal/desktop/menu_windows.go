//go:build windows

package desktop

import "github.com/bren-wp/Ghost-FTP/internal/i18n"

const (
	idSiteManager = 701
	idDiagnostics = 703
)

// navigationWords contains the small set of application-navigation nouns that
// are not part of the shared catalog yet. The native top menu was removed in
// 1.1.2 because it duplicated the canonical workspace/sidebar actions. Stable
// indices are retained for Site Manager and Diagnostics call sites while the
// remaining words keep legacy translations available for titles/documentation.
//
// Index contract: File, Servers, Transfers, View, Help, Site Manager, Exit,
// Tools, Diagnostics.
var navigationWords = map[string][9]string{
	"en": {"File", "Servers", "Transfers", "View", "Help", "Site Manager", "Exit", "Tools", "Diagnostics"},
	"hr": {"Datoteka", "Poslužitelji", "Prijenosi", "Prikaz", "Pomoć", "Upravitelj poslužitelja", "Izlaz", "Alati", "Dijagnostika"},
	"de": {"Datei", "Server", "Übertragungen", "Ansicht", "Hilfe", "Serververwaltung", "Beenden", "Werkzeuge", "Diagnose"},
	"fr": {"Fichier", "Serveurs", "Transferts", "Affichage", "Aide", "Gestionnaire de serveurs", "Quitter", "Outils", "Diagnostics"},
	"es": {"Archivo", "Servidores", "Transferencias", "Ver", "Ayuda", "Gestor de servidores", "Salir", "Herramientas", "Diagnóstico"},
	"tr": {"Dosya", "Sunucular", "Aktarımlar", "Görünüm", "Yardım", "Sunucu Yöneticisi", "Çıkış", "Araçlar", "Tanılama"},
	"el": {"Αρχείο", "Διακομιστές", "Μεταφορές", "Προβολή", "Βοήθεια", "Διαχείριση διακομιστών", "Έξοδος", "Εργαλεία", "Διαγνωστικά"},
	"pt": {"Ficheiro", "Servidores", "Transferências", "Ver", "Ajuda", "Gestor de servidores", "Sair", "Ferramentas", "Diagnóstico"},
	"zh": {"文件", "服务器", "传输", "查看", "帮助", "服务器管理器", "退出", "工具", "诊断"},
	"ru": {"Файл", "Серверы", "Передачи", "Вид", "Справка", "Менеджер серверов", "Выход", "Инструменты", "Диагностика"},
	"hi": {"फ़ाइल", "सर्वर", "स्थानांतरण", "दृश्य", "सहायता", "सर्वर प्रबंधक", "बाहर निकलें", "उपकरण", "निदान"},
	"ja": {"ファイル", "サーバー", "転送", "表示", "ヘルプ", "サーバーマネージャー", "終了", "ツール", "診断"},
	"it": {"File", "Server", "Trasferimenti", "Visualizza", "Aiuto", "Gestione server", "Esci", "Strumenti", "Diagnostica"},
	"pl": {"Plik", "Serwery", "Transfery", "Widok", "Pomoc", "Menedżer serwerów", "Wyjście", "Narzędzia", "Diagnostyka"},
	"nl": {"Bestand", "Servers", "Overdrachten", "Beeld", "Help", "Serverbeheer", "Afsluiten", "Hulpmiddelen", "Diagnostiek"},
	"cs": {"Soubor", "Servery", "Přenosy", "Zobrazení", "Nápověda", "Správce serverů", "Konec", "Nástroje", "Diagnostika"},
	"uk": {"Файл", "Сервери", "Передавання", "Вигляд", "Довідка", "Менеджер серверів", "Вихід", "Інструменти", "Діагностика"},
	"sv": {"Arkiv", "Servrar", "Överföringar", "Visa", "Hjälp", "Serverhanterare", "Avsluta", "Verktyg", "Diagnostik"},
	"ro": {"Fișier", "Servere", "Transferuri", "Vizualizare", "Ajutor", "Manager servere", "Ieșire", "Instrumente", "Diagnosticare"},
	"hu": {"Fájl", "Kiszolgálók", "Átvitelek", "Nézet", "Súgó", "Kiszolgálókezelő", "Kilépés", "Eszközök", "Diagnosztika"},
	"da": {"Filer", "Servere", "Overførsler", "Vis", "Hjælp", "Serveradministrator", "Afslut", "Værktøjer", "Diagnostik"},
	"fi": {"Tiedosto", "Palvelimet", "Siirrot", "Näytä", "Ohje", "Palvelinten hallinta", "Lopeta", "Työkalut", "Diagnostiikka"},
	"no": {"Fil", "Servere", "Overføringer", "Vis", "Hjelp", "Serverbehandling", "Avslutt", "Verktøy", "Diagnostikk"},
	"ko": {"파일", "서버", "전송", "보기", "도움말", "서버 관리자", "종료", "도구", "진단"},
}

func nativeMenuWords(language string) [9]string {
	if words, ok := navigationWords[i18n.Normalize(language)]; ok {
		return words
	}
	return navigationWords[i18n.DefaultLanguage]
}
