//go:build windows

package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"unsafe"
)

const (
	idSiteManager = 701
	idExitApp     = 702

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
var menuWords = map[string][7]string{
	"en": {"File", "Sites", "Transfers", "View", "Help", "Site Manager", "Exit"},
	"hr": {"Datoteka", "Web-mjesta", "Prijenosi", "Prikaz", "Pomoć", "Upravitelj web-mjesta", "Izlaz"},
	"de": {"Datei", "Server", "Übertragungen", "Ansicht", "Hilfe", "Serververwaltung", "Beenden"},
	"fr": {"Fichier", "Sites", "Transferts", "Affichage", "Aide", "Gestionnaire de sites", "Quitter"},
	"es": {"Archivo", "Sitios", "Transferencias", "Ver", "Ayuda", "Gestor de sitios", "Salir"},
	"tr": {"Dosya", "Siteler", "Aktarımlar", "Görünüm", "Yardım", "Site Yöneticisi", "Çıkış"},
	"el": {"Αρχείο", "Ιστότοποι", "Μεταφορές", "Προβολή", "Βοήθεια", "Διαχείριση τοποθεσιών", "Έξοδος"},
	"pt": {"Ficheiro", "Sites", "Transferências", "Ver", "Ajuda", "Gestor de sites", "Sair"},
	"zh": {"文件", "站点", "传输", "查看", "帮助", "站点管理器", "退出"},
	"ru": {"Файл", "Сайты", "Передачи", "Вид", "Справка", "Менеджер сайтов", "Выход"},
	"hi": {"फ़ाइल", "साइटें", "स्थानांतरण", "दृश्य", "सहायता", "साइट प्रबंधक", "बाहर निकलें"},
	"ja": {"ファイル", "サイト", "転送", "表示", "ヘルプ", "サイトマネージャー", "終了"},
	"it": {"File", "Siti", "Trasferimenti", "Visualizza", "Aiuto", "Gestione siti", "Esci"},
	"pl": {"Plik", "Witryny", "Transfery", "Widok", "Pomoc", "Menedżer witryn", "Wyjście"},
	"nl": {"Bestand", "Sites", "Overdrachten", "Beeld", "Help", "Sitebeheer", "Afsluiten"},
	"cs": {"Soubor", "Servery", "Přenosy", "Zobrazení", "Nápověda", "Správce serverů", "Konec"},
	"uk": {"Файл", "Сайти", "Передавання", "Вигляд", "Довідка", "Менеджер сайтів", "Вихід"},
	"sv": {"Arkiv", "Platser", "Överföringar", "Visa", "Hjälp", "Platshanterare", "Avsluta"},
	"ro": {"Fișier", "Site-uri", "Transferuri", "Vizualizare", "Ajutor", "Manager site-uri", "Ieșire"},
	"hu": {"Fájl", "Helyek", "Átvitelek", "Nézet", "Súgó", "Helykezelő", "Kilépés"},
	"da": {"Filer", "Websteder", "Overførsler", "Vis", "Hjælp", "Webstedsadministrator", "Afslut"},
	"fi": {"Tiedosto", "Sivustot", "Siirrot", "Näytä", "Ohje", "Sivustojen hallinta", "Lopeta"},
	"no": {"Fil", "Nettsteder", "Overføringer", "Vis", "Hjelp", "Nettstedsbehandling", "Avslutt"},
	"ko": {"파일", "사이트", "전송", "보기", "도움말", "사이트 관리자", "종료"},
}

func nativeMenuWords(language string) [7]string {
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
	sitesMenu, _, _ := createPopupMenuW.Call()
	transferMenu, _, _ := createPopupMenuW.Call()
	viewMenu, _, _ := createPopupMenuW.Call()
	helpMenu, _, _ := createPopupMenuW.Call()
	if fileMenu == 0 || sitesMenu == 0 || transferMenu == 0 || viewMenu == 0 || helpMenu == 0 {
		destroyMenuW.Call(root)
		return
	}

	words := nativeMenuWords(a.languageCode())
	appendMenuItem(fileMenu, idConnect, a.tr("common.connect"))
	appendMenuItem(fileMenu, idDisconnect, a.tr("common.disconnect"))
	appendMenuSeparator(fileMenu)
	appendMenuItem(fileMenu, idExitApp, words[6])

	appendMenuItem(sitesMenu, idSiteManager, words[5])
	appendMenuSeparator(sitesMenu)
	appendMenuItem(sitesMenu, idSaveProfile, a.tr("profile.save"))
	appendMenuItem(sitesMenu, idRemoveProfile, a.tr("profile.delete"))

	appendMenuItem(transferMenu, idRefreshAll, a.tr("common.refresh"))
	appendMenuSeparator(transferMenu)
	appendMenuItem(transferMenu, idPauseQueue, a.tr("transfer.pause"))
	appendMenuItem(transferMenu, idResumeQueue, a.tr("transfer.resume"))
	appendMenuItem(transferMenu, idClearQueue, a.tr("transfer.clear"))

	appendMenuItem(viewMenu, idSettings, a.tr("common.settings"))
	appendMenuItem(helpMenu, idAbout, a.tr("common.about"))

	appendPopup(root, fileMenu, words[0])
	appendPopup(root, sitesMenu, words[1])
	appendPopup(root, transferMenu, words[2])
	appendPopup(root, viewMenu, words[3])
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
