//go:build windows

package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"unsafe"
)

const (
	idSiteManager = 701
	idExitApp     = 702
	idDiagnostics = 703

	mfString    = 0x0000
	mfPopup     = 0x0010
	mfSeparator = 0x0800
)

var (
	createMenuW      = user32.NewProc("CreateMenu")
	createPopupMenuW = user32.NewProc("CreatePopupMenu")
	appendMenuW      = user32.NewProc("AppendMenuW")
	setMenuW         = user32.NewProc("SetMenu")
	getMenuW         = user32.NewProc("GetMenu")
	drawMenuBarW     = user32.NewProc("DrawMenuBar")
	destroyMenuW     = user32.NewProc("DestroyMenu")
)

// menuWords contains only native-menu nouns that are not part of the shared
// application catalog. Action labels reuse i18n.T so the menu follows the same
// 24-language runtime selection as the rest of the Windows UI.
//
// Index contract: File, Servers, Transfers, View, Help, Site Manager, Exit,
// Tools, Diagnostics. Keep the stable indices because Site Manager and the
// Tools menu use the same localized nouns across runtime language changes.
var menuWords = map[string][9]string{
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
	if words, ok := menuWords[i18n.Normalize(language)]; ok {
		return words
	}
	return menuWords["en"]
}

func appendMenuItem(menu uintptr, id int, label string) {
	appendMenuW.Call(menu, mfString, uintptr(id), uintptr(unsafe.Pointer(wstr(label))))
}

func appendMenuSeparator(menu uintptr) { appendMenuW.Call(menu, mfSeparator, 0, 0) }

func appendPopup(root, popup uintptr, label string) {
	appendMenuW.Call(root, mfPopup, popup, uintptr(unsafe.Pointer(wstr(label))))
}

func (a *app) installMainMenu() {
	if a == nil || a.hwnd == 0 {
		return
	}
	root, _, _ := createMenuW.Call()
	if root == 0 {
		return
	}
	fileMenu, _, _ := createPopupMenuW.Call()
	viewMenu, _, _ := createPopupMenuW.Call()
	transferMenu, _, _ := createPopupMenuW.Call()
	serversMenu, _, _ := createPopupMenuW.Call()
	toolsMenu, _, _ := createPopupMenuW.Call()
	helpMenu, _, _ := createPopupMenuW.Call()
	if fileMenu == 0 || viewMenu == 0 || transferMenu == 0 || serversMenu == 0 || toolsMenu == 0 || helpMenu == 0 {
		destroyMenuW.Call(root)
		return
	}

	words := nativeMenuWords(a.languageCode())
	appendMenuItem(fileMenu, idConnect, a.tr("common.connect"))
	appendMenuItem(fileMenu, idDisconnect, a.tr("common.disconnect"))
	appendMenuSeparator(fileMenu)
	appendMenuItem(fileMenu, idExitApp, words[6])

	appendMenuItem(viewMenu, idRefreshAll, a.tr("common.refresh"))

	appendMenuItem(transferMenu, idPauseQueue, a.tr("transfer.pause"))
	appendMenuItem(transferMenu, idResumeQueue, a.tr("transfer.resume"))
	appendMenuSeparator(transferMenu)
	appendMenuItem(transferMenu, idClearQueue, a.tr("transfer.clear"))

	appendMenuItem(serversMenu, idSiteManager, words[5])
	appendMenuSeparator(serversMenu)
	appendMenuItem(serversMenu, idSaveProfile, a.tr("profile.save"))
	appendMenuItem(serversMenu, idRemoveProfile, a.tr("profile.delete"))

	appendMenuItem(toolsMenu, idSettings, a.tr("common.settings"))
	appendMenuItem(toolsMenu, idDiagnostics, words[8])

	appendMenuItem(helpMenu, idAbout, a.tr("common.about"))

	appendPopup(root, fileMenu, words[0])
	appendPopup(root, viewMenu, words[3])
	appendPopup(root, transferMenu, words[2])
	appendPopup(root, serversMenu, words[1])
	appendPopup(root, toolsMenu, words[7])
	appendPopup(root, helpMenu, words[4])

	old, _, _ := getMenuW.Call(a.hwnd)
	if ok, _, _ := setMenuW.Call(a.hwnd, root); ok == 0 {
		destroyMenuW.Call(root)
		return
	}
	drawMenuBarW.Call(a.hwnd)
	if old != 0 && old != root {
		destroyMenuW.Call(old)
	}
}
