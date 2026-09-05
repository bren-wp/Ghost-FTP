//go:build windows

package desktop

import (
	"fmt"
	"strings"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	idToolbarConnect = 801 + iota
	idToolbarDisconnect
	idToolbarUpload
	idToolbarDownload
	idToolbarRefresh
	idToolbarNewFolder
	idToolbarRename
	idToolbarDelete
	idToolbarSites
	idToolbarSettings
	idToolbarDiagnostics
	idRemoteSearch
)

const enChange = 0x0300

var (
	getFocusW     = user32.NewProc("GetFocus")
	setWindowPosW = user32.NewProc("SetWindowPos")
)

const (
	hwndBottom    = 1
	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
)

type referenceWordsSet struct {
	LogTitle      string
	Diagnostics   string
	Search        string
	PrivacyTitle  string
	PrivacyBody   string
	LocalDevice   string
	RemoteOffline string
}

var referenceWords = map[string]referenceWordsSet{
	"en": {"Connection log", "Diagnostics", "Search remote files…", "No account required", "No telemetry or tracking. Saved profiles stay on this computer.", "This device", "Not connected"},
	"hr": {"Zapis veze", "Dijagnostika", "Pretraži udaljene datoteke…", "Račun nije potreban", "Bez telemetrije i praćenja. Spremljeni profili ostaju na ovom računalu.", "Ovaj uređaj", "Nije povezano"},
	"de": {"Verbindungsprotokoll", "Diagnose", "Remote-Dateien suchen…", "Kein Konto erforderlich", "Keine Telemetrie oder Verfolgung. Gespeicherte Profile bleiben auf diesem Computer.", "Dieses Gerät", "Nicht verbunden"},
	"fr": {"Journal de connexion", "Diagnostics", "Rechercher des fichiers distants…", "Aucun compte requis", "Aucune télémétrie ni suivi. Les profils enregistrés restent sur cet ordinateur.", "Cet appareil", "Non connecté"},
	"es": {"Registro de conexión", "Diagnóstico", "Buscar archivos remotos…", "No se requiere cuenta", "Sin telemetría ni seguimiento. Los perfiles guardados permanecen en este equipo.", "Este dispositivo", "Sin conexión"},
	"tr": {"Bağlantı günlüğü", "Tanılama", "Uzak dosyalarda ara…", "Hesap gerekmez", "Telemetri veya izleme yok. Kaydedilen profiller bu bilgisayarda kalır.", "Bu cihaz", "Bağlı değil"},
	"el": {"Αρχείο σύνδεσης", "Διαγνωστικά", "Αναζήτηση απομακρυσμένων αρχείων…", "Δεν απαιτείται λογαριασμός", "Χωρίς τηλεμετρία ή παρακολούθηση. Τα αποθηκευμένα προφίλ μένουν σε αυτόν τον υπολογιστή.", "Αυτή η συσκευή", "Δεν υπάρχει σύνδεση"},
	"pt": {"Registo de ligação", "Diagnóstico", "Pesquisar ficheiros remotos…", "Não é necessária conta", "Sem telemetria ou rastreio. Os perfis guardados ficam neste computador.", "Este dispositivo", "Não ligado"},
	"zh": {"连接日志", "诊断", "搜索远程文件…", "无需账户", "无遥测或跟踪。已保存的配置保留在此计算机上。", "此设备", "未连接"},
	"ru": {"Журнал подключения", "Диагностика", "Поиск удалённых файлов…", "Учётная запись не требуется", "Без телеметрии и отслеживания. Сохранённые профили остаются на этом компьютере.", "Это устройство", "Не подключено"},
	"hi": {"कनेक्शन लॉग", "निदान", "दूरस्थ फ़ाइलें खोजें…", "खाते की आवश्यकता नहीं", "कोई टेलीमेट्री या ट्रैकिंग नहीं। सहेजे गए प्रोफ़ाइल इसी कंप्यूटर पर रहते हैं।", "यह डिवाइस", "कनेक्ट नहीं"},
	"ja": {"接続ログ", "診断", "リモートファイルを検索…", "アカウント不要", "テレメトリや追跡はありません。保存したプロファイルはこのコンピューター内に残ります。", "このデバイス", "未接続"},
	"it": {"Registro connessione", "Diagnostica", "Cerca file remoti…", "Nessun account richiesto", "Nessuna telemetria o tracciamento. I profili salvati restano su questo computer.", "Questo dispositivo", "Non connesso"},
	"pl": {"Dziennik połączenia", "Diagnostyka", "Szukaj plików zdalnych…", "Konto nie jest wymagane", "Brak telemetrii i śledzenia. Zapisane profile pozostają na tym komputerze.", "To urządzenie", "Brak połączenia"},
	"nl": {"Verbindingslog", "Diagnostiek", "Externe bestanden zoeken…", "Geen account vereist", "Geen telemetrie of tracking. Opgeslagen profielen blijven op deze computer.", "Dit apparaat", "Niet verbonden"},
	"cs": {"Protokol připojení", "Diagnostika", "Hledat vzdálené soubory…", "Účet není potřeba", "Bez telemetrie a sledování. Uložené profily zůstávají v tomto počítači.", "Toto zařízení", "Nepřipojeno"},
	"uk": {"Журнал з’єднання", "Діагностика", "Пошук віддалених файлів…", "Обліковий запис не потрібен", "Без телеметрії та відстеження. Збережені профілі залишаються на цьому комп’ютері.", "Цей пристрій", "Не підключено"},
	"sv": {"Anslutningslogg", "Diagnostik", "Sök fjärrfiler…", "Inget konto krävs", "Ingen telemetri eller spårning. Sparade profiler stannar på den här datorn.", "Den här enheten", "Inte ansluten"},
	"ro": {"Jurnal conexiune", "Diagnosticare", "Caută fișiere la distanță…", "Nu este necesar un cont", "Fără telemetrie sau urmărire. Profilurile salvate rămân pe acest computer.", "Acest dispozitiv", "Neconectat"},
	"hu": {"Kapcsolati napló", "Diagnosztika", "Távoli fájlok keresése…", "Nem szükséges fiók", "Nincs telemetria vagy követés. A mentett profilok ezen a számítógépen maradnak.", "Ez az eszköz", "Nincs kapcsolat"},
	"da": {"Forbindelseslog", "Diagnostik", "Søg i fjernfiler…", "Ingen konto kræves", "Ingen telemetri eller sporing. Gemte profiler forbliver på denne computer.", "Denne enhed", "Ikke forbundet"},
	"fi": {"Yhteysloki", "Diagnostiikka", "Hae etätiedostoja…", "Tiliä ei tarvita", "Ei telemetriaa tai seurantaa. Tallennetut profiilit pysyvät tällä tietokoneella.", "Tämä laite", "Ei yhteyttä"},
	"no": {"Tilkoblingslogg", "Diagnostikk", "Søk i eksterne filer…", "Ingen konto kreves", "Ingen telemetri eller sporing. Lagrede profiler forblir på denne datamaskinen.", "Denne enheten", "Ikke tilkoblet"},
	"ko": {"연결 로그", "진단", "원격 파일 검색…", "계정이 필요하지 않음", "원격 측정이나 추적이 없습니다. 저장된 프로필은 이 컴퓨터에만 남습니다.", "이 장치", "연결되지 않음"},
}

func (a *app) refWords() referenceWordsSet {
	if words, ok := referenceWords[a.languageCode()]; ok {
		return words
	}
	return referenceWords["en"]
}

func (a *app) ensureReferenceShellControls() {
	if a == nil || a.hwnd == 0 || a.shellSidebar != 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	mk := func(class, text string, style uint32, id int) uintptr {
		h, _, _ := createWindowExW.Call(
			0, uintptr(unsafe.Pointer(wstr(class))), uintptr(unsafe.Pointer(wstr(text))),
			uintptr(wsChild|wsVisible|style), 0, 0, 10, 10,
			a.hwnd, uintptr(id), hinst, 0,
		)
		if h != 0 && a.font != 0 {
			sendMessageW.Call(h, wmSetFont, a.font, 1)
		}
		if h != 0 {
			applyDarkControl(h, class)
		}
		return h
	}
	panel := func() uintptr {
		h := mk("STATIC", "", wsBorder, 0)
		if h != 0 {
			setWindowPosW.Call(h, hwndBottom, 0, 0, 0, 0, swpNoMove|swpNoSize|swpNoActivate)
		}
		return h
	}
	mkToolbar := func(id int, label, icon string, variant buttonVariant) uintptr {
		h := mk("BUTTON", label, bsOwnerDraw|wsTabStop, id)
		return a.registerToolbarButton(h, icon, label, variant)
	}
	words := a.refWords()

	a.shellSidebar = panel()
	a.shellToolbar = panel()
	a.shellLogCard = panel()
	a.shellQuickCard = panel()
	a.shellLocalCard = panel()
	a.shellRemoteCard = panel()
	a.shellQueueCard = panel()

	a.sidebarServersLabel = mk("STATIC", nativeMenuWords(a.languageCode())[1], 0, 0)
	a.sidebarPrivacyTitle = mk("STATIC", words.PrivacyTitle, 0, 0)
	a.sidebarPrivacyBody = mk("STATIC", words.PrivacyBody, 0, 0)
	a.logTitle = mk("STATIC", words.LogTitle, 0, 0)
	a.quickTitle = mk("STATIC", a.tr("profile.quick"), 0, 0)
	a.localDeviceLabel = mk("STATIC", words.LocalDevice, 0, 0)
	a.remoteStateLabel = mk("STATIC", words.RemoteOffline, 0, 0)

	a.remoteSearch = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idRemoteSearch)
	limitEdit(a.remoteSearch, 256)
	cue(a.remoteSearch, words.Search)

	a.toolbarConnect = mkToolbar(idToolbarConnect, a.tr("common.connect"), iconConnect, buttonSubtle)
	a.toolbarDisconnect = mkToolbar(idToolbarDisconnect, a.tr("common.disconnect"), iconDisconnect, buttonSubtle)
	a.toolbarUpload = mkToolbar(idToolbarUpload, a.tr("transfer.upload"), iconUpload, buttonSubtle)
	a.toolbarDownload = mkToolbar(idToolbarDownload, a.tr("transfer.download"), iconDownload, buttonSubtle)
	a.toolbarRefresh = mkToolbar(idToolbarRefresh, a.tr("common.refresh"), iconRefresh, buttonSubtle)
	a.toolbarNewFolder = mkToolbar(idToolbarNewFolder, a.tr("common.new_folder"), iconNewFolder, buttonSubtle)
	a.toolbarRename = mkToolbar(idToolbarRename, a.tr("common.rename"), iconRename, buttonSubtle)
	a.toolbarDelete = mkToolbar(idToolbarDelete, a.tr("common.delete"), iconDelete, buttonSubtle)
	a.toolbarSites = mkToolbar(idToolbarSites, nativeMenuWords(a.languageCode())[5], iconOpenLocal, buttonSubtle)
	a.toolbarSettings = mkToolbar(idToolbarSettings, a.tr("common.settings"), iconSettings, buttonSubtle)
	a.toolbarDiagnostics = mkToolbar(idToolbarDiagnostics, words.Diagnostics, iconDiagnostics, buttonSubtle)

	for _, h := range []uintptr{a.sidebarServersLabel, a.sidebarPrivacyTitle, a.sidebarPrivacyBody, a.logTitle, a.quickTitle, a.localDeviceLabel, a.remoteStateLabel} {
		if h != 0 && a.smallFont != 0 {
			sendMessageW.Call(h, wmSetFont, a.smallFont, 1)
		}
	}
}

func (a *app) updateReferenceShellLanguage() {
	if a == nil || a.shellSidebar == 0 {
		return
	}
	words := a.refWords()
	setText(a.sidebarServersLabel, nativeMenuWords(a.languageCode())[1])
	setText(a.sidebarPrivacyTitle, words.PrivacyTitle)
	setText(a.sidebarPrivacyBody, words.PrivacyBody)
	setText(a.logTitle, words.LogTitle)
	setText(a.quickTitle, a.tr("profile.quick"))
	setText(a.localDeviceLabel, words.LocalDevice)
	if !a.connected {
		setText(a.remoteStateLabel, words.RemoteOffline)
	}
	cue(a.remoteSearch, words.Search)
	for _, pair := range []struct {
		h     uintptr
		label string
	}{
		{a.toolbarConnect, a.tr("common.connect")},
		{a.toolbarDisconnect, a.tr("common.disconnect")},
		{a.toolbarUpload, a.tr("transfer.upload")},
		{a.toolbarDownload, a.tr("transfer.download")},
		{a.toolbarRefresh, a.tr("common.refresh")},
		{a.toolbarNewFolder, a.tr("common.new_folder")},
		{a.toolbarRename, a.tr("common.rename")},
		{a.toolbarDelete, a.tr("common.delete")},
		{a.toolbarSites, nativeMenuWords(a.languageCode())[5]},
		{a.toolbarSettings, a.tr("common.settings")},
		{a.toolbarDiagnostics, words.Diagnostics},
	} {
		a.setButtonLabel(pair.h, pair.label)
	}
}

func (a *app) applyRemoteSearch() {
	if a == nil || a.remoteList == 0 {
		return
	}
	query := strings.ToLower(strings.TrimSpace(getText(a.remoteSearch)))
	source := a.remoteAllItems
	if source == nil {
		source = a.remoteItems
	}
	if query == "" {
		a.remoteItems = append(a.remoteItems[:0], source...)
	} else {
		filtered := make([]model.Item, 0, len(source))
		for _, item := range source {
			if strings.Contains(strings.ToLower(item.Name), query) {
				filtered = append(filtered, item)
			}
		}
		a.remoteItems = filtered
	}
	a.fillItemList(a.remoteList, a.remoteItems)
	a.updateActionControls()
}

func (a *app) toolbarTargetsRemote() bool {
	focus, _, _ := getFocusW.Call()
	for _, h := range []uintptr{a.remotePath, a.remoteSearch, a.remoteList, a.remoteUp, a.remoteRefresh, a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod} {
		if focus == h {
			return true
		}
	}
	for _, h := range []uintptr{a.localPath, a.localList, a.localUp, a.localRefresh, a.localChoose, a.localMkdir, a.localRename, a.localDelete} {
		if focus == h {
			return false
		}
	}
	return a.connected && validSelectionCount(a.remoteList, len(a.remoteItems)) > 0 && validSelectionCount(a.localList, len(a.localItems)) == 0
}

func (a *app) toolbarNewFolderAction() {
	if a.toolbarTargetsRemote() && a.connected && !a.connectionBusy {
		a.remoteMkdirAction()
		return
	}
	a.localMkdirAction()
}

func (a *app) toolbarRenameAction() {
	if a.toolbarTargetsRemote() && a.connected && !a.connectionBusy {
		a.remoteRenameAction()
		return
	}
	a.localRenameAction()
}

func (a *app) toolbarDeleteAction() {
	if a.toolbarTargetsRemote() && a.connected && !a.connectionBusy {
		a.remoteDeleteAction()
		return
	}
	a.localDeleteAction()
}

func (a *app) showDiagnostics() {
	if a == nil {
		return
	}
	words := a.refWords()
	state := words.RemoteOffline
	if a.connected {
		state = a.tr("badge.connected")
	}
	body := fmt.Sprintf(
		"Ghost FTP %s\n\n%s · %s\n%s\n\n%s",
		a.version, strings.ToUpper(a.protocolValue()), state, a.remoteCurrent, words.PrivacyBody,
	)
	platform.InfoDialog("Ghost FTP — "+words.Diagnostics, words.Diagnostics, body)
}
