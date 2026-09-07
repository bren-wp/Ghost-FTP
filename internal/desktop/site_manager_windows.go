//go:build windows

package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	siteIDList       = 8101
	siteIDName       = 8102
	siteIDProtocol   = 8103
	siteIDHost       = 8104
	siteIDPort       = 8105
	siteIDUser       = 8106
	siteIDLocal      = 8107
	siteIDRemote     = 8108
	siteIDKey        = 8109
	siteIDSecurity   = 8110
	siteIDSave       = 8111
	siteIDDelete     = 8112
	siteIDConnect    = 8113
	siteIDClose      = 8114
	siteIDPassword   = 8115
	siteIDPassphrase = 8116

	siteLBSNotify           = 0x0001
	siteLBSNoIntegralHeight = 0x0100
	siteLBAddString         = 0x0180
	siteLBResetContent      = 0x0184
	siteLBSetCurSel         = 0x0186
	siteLBGetCurSel         = 0x0188
	siteLBNSelChange        = 1
	siteLBNDblClk           = 2
	siteBSDefPushButton     = 0x00000001
	siteWindowStyle         = 0x00C80000 // WS_CAPTION | WS_SYSMENU
	siteWMCtlColorListBox   = 0x0134
)

var sitePathLabels = map[string][2]string{
	"en": {"Local path", "Remote path"},
	"hr": {"Lokalna putanja", "Udaljena putanja"},
	"de": {"Lokaler Pfad", "Remote-Pfad"},
	"fr": {"Chemin local", "Chemin distant"},
	"es": {"Ruta local", "Ruta remota"},
	"tr": {"Yerel yol", "Uzak yol"},
	"el": {"Τοπική διαδρομή", "Απομακρυσμένη διαδρομή"},
	"pt": {"Caminho local", "Caminho remoto"},
	"zh": {"本地路径", "远程路径"},
	"ru": {"Локальный путь", "Удалённый путь"},
	"hi": {"स्थानीय पथ", "दूरस्थ पथ"},
	"ja": {"ローカルパス", "リモートパス"},
	"it": {"Percorso locale", "Percorso remoto"},
	"pl": {"Ścieżka lokalna", "Ścieżka zdalna"},
	"nl": {"Lokaal pad", "Extern pad"},
	"cs": {"Místní cesta", "Vzdálená cesta"},
	"uk": {"Локальний шлях", "Віддалений шлях"},
	"sv": {"Lokal sökväg", "Fjärrsökväg"},
	"ro": {"Cale locală", "Cale la distanță"},
	"hu": {"Helyi elérési út", "Távoli elérési út"},
	"da": {"Lokal sti", "Fjernsti"},
	"fi": {"Paikallinen polku", "Etäpolku"},
	"no": {"Lokal sti", "Ekstern sti"},
	"ko": {"로컬 경로", "원격 경로"},
}

func sitePathLabel(language string, remote bool) string {
	pair, ok := sitePathLabels[i18n.Normalize(language)]
	if !ok {
		pair = sitePathLabels[i18n.DefaultLanguage]
	}
	if remote {
		return pair[1]
	}
	return pair[0]
}

func cleanSFTPSecurityTitle(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"GhostFTP —", "Ghost FTP —"} {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	}
	if value == "" {
		return "SFTP security"
	}
	return value
}

type siteManagerState struct {
	parent       *app
	hwnd         uintptr
	list         uintptr
	listBrush    uintptr
	name         uintptr
	protocol     uintptr
	host         uintptr
	port         uintptr
	user         uintptr
	password     uintptr
	localPath    uintptr
	remotePath   uintptr
	keyPath      uintptr
	passphrase   uintptr
	security     uintptr
	save         uintptr
	delete       uintptr
	connect      uintptr
	close        uintptr
	profiles     []model.PublicProfile
	selected     int
	closed       bool
	connectAfter bool
}

var (
	siteManagerStates sync.Map
	siteManagerOnce   sync.Once
	siteManagerClass  = "GhostFTP.SiteManager"
	siteManagerProc   = syscall.NewCallback(siteManagerWndProc)
)

func siteManagerWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	value, ok := siteManagerStates.Load(hwnd)
	if ok {
		state := value.(*siteManagerState)
		switch message {
		case wmCommand:
			id := int(wParam & 0xffff)
			notify := int((wParam >> 16) & 0xffff)
			if id == siteIDList && (notify == siteLBNSelChange || notify == siteLBNDblClk) {
				state.loadCurrentSelection()
				if notify == siteLBNDblClk && state.selected > 0 {
					state.connectAfter = true
					destroyWindow.Call(hwnd)
				}
				return 0
			}
			if id == siteIDProtocol && notify == cbnSelChange {
				state.syncProtocolPort()
				return 0
			}
			if notify == bnClicked {
				switch id {
				case siteIDSave:
					state.saveCurrent()
					return 0
				case siteIDDelete:
					state.deleteCurrent()
					return 0
				case siteIDConnect:
					if state.selected == 0 {
						state.applyQuickConnectToMain()
					}
					state.connectAfter = true
					destroyWindow.Call(hwnd)
					return 0
				case siteIDClose:
					destroyWindow.Call(hwnd)
					return 0
				}
			}
		case wmMeasureItem:
			if state.measureNavigationItem(lParam) {
				return 1
			}
		case wmDrawItem:
			if lParam != 0 {
				d := drawItemFromLParam(lParam)
				if state.drawNavigationItem(&d) {
					return 1
				}
				if state.parent.drawButton(&d) {
					return 1
				}
			}
		case siteWMCtlColorListBox:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, listColor())
			if state.listBrush != 0 {
				return state.listBrush
			}
			return state.parent.panelBrush
		case wmCtlColorEdit, wmCtlColorBtn, wmCtlColorStatic:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, windowColor())
			return state.parent.brush
		case wmClose:
			destroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			for _, button := range []uintptr{state.save, state.delete, state.connect, state.close} {
				delete(state.parent.buttons, button)
			}
			if state.listBrush != 0 {
				deleteObject.Call(state.listBrush)
				state.listBrush = 0
			}
			state.closed = true
			return 0
		}
	}
	result, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func (state *siteManagerState) protocolValue() string {
	index := selectedComboIndex(state.protocol)
	if index < 0 || index >= len(protocolSpecs) {
		return protocolSpecs[0].Value
	}
	return protocolSpecs[index].Value
}

func (state *siteManagerState) setProtocol(protocol string) {
	sendMessageW.Call(state.protocol, cbSetCurSel, protocolIndex(protocol), 0)
	state.syncProtocolControls()
}

func (state *siteManagerState) syncProtocolControls() {
	sftp := state.protocolValue() == "sftp"
	setControlEnabled(state.keyPath, sftp)
	setControlEnabled(state.passphrase, sftp)
	if !sftp {
		setText(state.passphrase, "")
	}
}

func (state *siteManagerState) syncProtocolPort() {
	protocol := state.protocolValue()
	current := strings.TrimSpace(getText(state.port))
	for _, spec := range protocolSpecs {
		if current == spec.Port {
			setText(state.port, protocolSpecs[protocolIndex(protocol)].Port)
			state.syncProtocolControls()
			return
		}
	}
	if current == "" {
		setText(state.port, protocolSpecs[protocolIndex(protocol)].Port)
	}
	state.syncProtocolControls()
}

func (state *siteManagerState) refillProfiles(selectedID string) {
	sendMessageW.Call(state.list, siteLBResetContent, 0, 0)
	quick := "+  " + state.parent.tr("profile.quick")
	sendMessageW.Call(state.list, siteLBAddString, 0, uintptr(unsafe.Pointer(wstr(quick))))
	selected := 0
	for index, profile := range state.profiles {
		label := profile.Name + "   ·   " + strings.ToUpper(profile.Protocol) + "   ·   " + profile.Host
		sendMessageW.Call(state.list, siteLBAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
		if profile.ID == selectedID {
			selected = index + 1
		}
	}
	state.selected = selected
	sendMessageW.Call(state.list, siteLBSetCurSel, uintptr(selected), 0)
	state.loadSelection(selected)
}

func (state *siteManagerState) loadCurrentSelection() {
	index, _, _ := sendMessageW.Call(state.list, siteLBGetCurSel, 0, 0)
	if int32(index) < 0 {
		return
	}
	state.loadSelection(int(index))
}

func (state *siteManagerState) loadSelection(index int) {
	state.selected = index
	setText(state.password, "")
	setText(state.passphrase, "")
	if index <= 0 || index > len(state.profiles) {
		state.selected = 0
		setText(state.name, "")
		state.setProtocol(defaultConnectionProtocol)
		setText(state.host, "")
		setText(state.port, protocolSpecs[0].Port)
		setText(state.user, "")
		setText(state.localPath, state.parent.localCurrent)
		setText(state.remotePath, "/")
		setText(state.keyPath, "")
		setText(state.security, state.parent.tr("profile.quick")+" · "+state.parent.tr("cue.password"))
		setControlEnabled(state.delete, false)
		return
	}
	profile := state.profiles[index-1]
	setText(state.name, profile.Name)
	state.setProtocol(profile.Protocol)
	setText(state.host, profile.Host)
	setText(state.port, strconv.Itoa(profile.Port))
	setText(state.user, profile.Username)
	setText(state.localPath, profile.LocalPath)
	setText(state.remotePath, profile.RemotePath)
	setText(state.keyPath, profile.PrivateKeyPath)
	setText(state.security, state.securitySummary(profile))
	setControlEnabled(state.delete, true)
}

func (state *siteManagerState) securitySummary(profile model.PublicProfile) string {
	parts := make([]string, 0, 4)
	if profile.HasPassword {
		parts = append(parts, "● "+state.parent.tr("terminal.password"))
	}
	if profile.PrivateKeyPath != "" {
		parts = append(parts, "● "+state.parent.tr("terminal.private_key"))
	}
	if profile.HasPassphrase {
		parts = append(parts, "● "+state.parent.tr("cue.passphrase"))
	}
	if profile.Fingerprint != "" {
		parts = append(parts, "● "+strings.TrimSuffix(state.parent.tr("terminal.fingerprint"), ":")+" "+profile.Fingerprint)
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "    ")
}

func (state *siteManagerState) profileInput() (model.ProfileInput, error) {
	protocol := state.protocolValue()
	host := getText(state.host)
	username := getText(state.user)
	port, err := validateRawConnectionInput(protocol, host, getText(state.port), username)
	if err != nil {
		return model.ProfileInput{}, err
	}
	name := strings.TrimSpace(getText(state.name))
	if name == "" {
		return model.ProfileInput{}, fmt.Errorf("%s", state.parent.tr("column.name"))
	}
	id := ""
	if state.selected > 0 && state.selected <= len(state.profiles) {
		id = state.profiles[state.selected-1].ID
	}
	remotePath := optionalRemotePath(getText(state.remotePath))
	return model.ProfileInput{
		ID:             id,
		Name:           name,
		Protocol:       protocol,
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       getText(state.password),
		PrivateKeyPath: getText(state.keyPath),
		Passphrase:     getText(state.passphrase),
		LocalPath:      getText(state.localPath),
		RemotePath:     remotePath,
	}, nil
}

func (state *siteManagerState) saveCurrent() {
	input, err := state.profileInput()
	if err != nil {
		platform.ErrorDialog("Ghost FTP — "+nativeMenuWords(state.parent.languageCode())[5], state.parent.tr("settings.invalid_value"), state.parent.userMessage(err, "error.generic"))
		return
	}
	if input.Password != "" || input.Passphrase != "" {
		words := credentialConsentText(state.parent.languageCode())
		if !platform.ConfirmDialog(words.Title, words.Question, words.Body) {
			input.Password = ""
			input.Passphrase = ""
			input.ClearPassword = true
			input.ClearPassphrase = true
		}
	}
	saved, err := state.parent.engine.SaveProfile(input)
	input.Password = ""
	input.Passphrase = ""
	setText(state.password, "")
	setText(state.passphrase, "")
	if err != nil {
		platform.ErrorDialog("Ghost FTP — "+nativeMenuWords(state.parent.languageCode())[5], state.parent.tr("settings.save_failed"), state.parent.userMessage(err, "settings.save_failed_body"))
		return
	}
	profiles, err := state.parent.engine.Profiles()
	if err != nil {
		platform.ErrorDialog("Ghost FTP", state.parent.tr("profile.load_failed"), state.parent.userMessage(err, "error.generic"))
		return
	}
	state.profiles = profiles
	state.parent.selectedProfileID = saved.ID
	state.parent.applyProfiles(profiles, nil)
	state.refillProfiles(saved.ID)
	state.parent.setStatus(state.parent.tr("profile.save") + ": " + saved.Name)
}

func (state *siteManagerState) deleteCurrent() {
	if state.selected <= 0 || state.selected > len(state.profiles) {
		return
	}
	profile := state.profiles[state.selected-1]
	if !platform.ConfirmDialog("Ghost FTP — "+nativeMenuWords(state.parent.languageCode())[5], state.parent.tr("profile.delete"), profile.Name) {
		return
	}
	if err := state.parent.engine.RemoveProfile(profile.ID); err != nil {
		platform.ErrorDialog("Ghost FTP", state.parent.tr("profile.delete"), state.parent.userMessage(err, "error.generic"))
		return
	}
	profiles, err := state.parent.engine.Profiles()
	if err != nil {
		platform.ErrorDialog("Ghost FTP", state.parent.tr("profile.load_failed"), state.parent.userMessage(err, "error.generic"))
		return
	}
	state.profiles = profiles
	if state.parent.selectedProfileID == profile.ID {
		state.parent.selectedProfileID = ""
	}
	state.parent.applyProfiles(profiles, nil)
	state.refillProfiles("")
	state.parent.setStatus(state.parent.tr("profile.delete"))
}

func (state *siteManagerState) applyQuickConnectToMain() {
	state.parent.selectedProfileID = ""
	sendMessageW.Call(state.parent.profilesCombo, cbSetCurSel, 0, 0)
	state.parent.setProtocolValue(state.protocolValue())
	setText(state.parent.host, getText(state.host))
	setText(state.parent.port, getText(state.port))
	setText(state.parent.user, getText(state.user))
	setText(state.parent.pass, getText(state.password))
	setText(state.parent.keyPath, getText(state.keyPath))
	setText(state.parent.passphrase, getText(state.passphrase))
	setText(state.parent.localPath, getText(state.localPath))
	setText(state.parent.remotePath, getText(state.remotePath))
	state.parent.resetProfileCredentialCues()
	state.parent.updateActionControls()
}

func (state *siteManagerState) createControls(hinst uintptr) error {
	parent := state.parent
	mk := func(class, text string, style uint32, x, y, width, height, id int) uintptr {
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr(class))),
			uintptr(unsafe.Pointer(wstr(text))),
			uintptr(wsChild|wsVisible|style),
			uintptr(parent.scale(x)), uintptr(parent.scale(y)), uintptr(parent.scale(width)), uintptr(parent.scale(height)),
			state.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd != 0 && parent.font != 0 {
			sendMessageW.Call(hwnd, wmSetFont, parent.font, 1)
		}
		if hwnd != 0 {
			applyDarkControl(hwnd, class)
		}
		return hwnd
	}
	label := func(text string, x, y, width int) uintptr { return mk("STATIC", text, 0, x, y, width, 20, 0) }

	words := nativeMenuWords(parent.languageCode())
	label(strings.ToUpper(words[1]), 20, 18, 260)
	label(strings.ToUpper(words[5]), 310, 18, 570)

	state.list = mk("LISTBOX", "", wsBorder|wsTabStop|wsVScroll|siteLBSNotify|siteLBSNoIntegralHeight|siteLBSOwnerDrawFixed|siteLBSHasStrings, 20, 48, 270, 486, siteIDList)
	if state.list != 0 {
		applySiteManagerNavigationTheme(state.list)
		state.listBrush, _, _ = createSolidBrush.Call(listColor())
	}

	label(parent.tr("column.name"), 310, 54, 160)
	state.name = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 490, 48, 390, 30, siteIDName)

	label(parent.tr("terminal.protocol"), 310, 96, 160)
	state.protocol = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, 490, 90, 180, 220, siteIDProtocol)
	for _, spec := range protocolSpecs {
		sendMessageW.Call(state.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(protocolLabel(parent.languageCode(), spec.Value)))))
	}
	label(parent.tr("terminal.port"), 690, 96, 60)
	state.port = mk("EDIT", protocolSpecs[0].Port, wsBorder|wsTabStop|esAutoHScroll, 760, 90, 120, 30, siteIDPort)

	label(parent.tr("terminal.server"), 310, 138, 160)
	state.host = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 490, 132, 390, 30, siteIDHost)

	label(parent.tr("terminal.username"), 310, 180, 160)
	state.user = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 490, 174, 180, 30, siteIDUser)
	label(parent.tr("terminal.password"), 690, 180, 60)
	state.password = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, 760, 174, 120, 30, siteIDPassword)

	label(sitePathLabel(parent.languageCode(), false), 310, 222, 160)
	state.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 490, 216, 390, 30, siteIDLocal)
	label(sitePathLabel(parent.languageCode(), true), 310, 264, 160)
	state.remotePath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 490, 258, 390, 30, siteIDRemote)

	label(parent.tr("terminal.private_key"), 310, 306, 160)
	state.keyPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 490, 300, 390, 30, siteIDKey)
	label(parent.tr("cue.passphrase"), 310, 348, 160)
	state.passphrase = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, 490, 342, 390, 30, siteIDPassphrase)

	label(cleanSFTPSecurityTitle(parent.tr("sftp.security")), 310, 390, 570)
	state.security = mk("STATIC", "", wsBorder, 310, 416, 570, 58, siteIDSecurity)

	state.save = parent.registerButton(mk("BUTTON", parent.tr("profile.save"), wsTabStop|bsOwnerDraw, 310, 500, 146, 34, siteIDSave), iconSave, parent.tr("profile.save"), buttonDefault)
	state.delete = parent.registerButton(mk("BUTTON", parent.tr("profile.delete"), wsTabStop|bsOwnerDraw, 466, 500, 146, 34, siteIDDelete), iconDelete, parent.tr("profile.delete"), buttonDanger)
	state.connect = parent.registerButton(mk("BUTTON", parent.tr("common.connect"), wsTabStop|siteBSDefPushButton|bsOwnerDraw, 622, 500, 120, 34, siteIDConnect), iconConnect, parent.tr("common.connect"), buttonAccent)
	state.close = parent.registerButton(mk("BUTTON", parent.tr("common.cancel"), wsTabStop|bsOwnerDraw, 752, 500, 128, 34, siteIDClose), iconCancel, parent.tr("common.cancel"), buttonSubtle)

	for _, control := range []uintptr{
		state.list, state.name, state.protocol, state.host, state.port, state.user, state.password,
		state.localPath, state.remotePath, state.keyPath, state.passphrase, state.security,
		state.save, state.delete, state.connect, state.close,
	} {
		if control == 0 {
			return fmt.Errorf("Site Manager control initialization failed")
		}
	}
	limitEdit(state.name, 120)
	limitEdit(state.host, 253)
	limitEdit(state.port, 5)
	limitEdit(state.user, 1024)
	limitEdit(state.password, 8192)
	limitEdit(state.localPath, 32767)
	limitEdit(state.remotePath, 4096)
	limitEdit(state.keyPath, 32767)
	limitEdit(state.passphrase, 8192)
	cue(state.password, parent.tr("cue.password"))
	cue(state.passphrase, parent.tr("cue.passphrase"))
	return nil
}

func (a *app) openSiteManager() {
	if a == nil || a.hwnd == 0 || a.connectionBusy || a.connected {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	siteManagerOnce.Do(func() {
		cursor, _, _ := loadCursorW.Call(0, 32512)
		icon, _, _ := loadIconW.Call(hinst, 1)
		class := wndClassEx{
			CbSize:     uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:    siteManagerProc,
			Instance:   hinst,
			Icon:       icon,
			Cursor:     cursor,
			Background: a.brush,
			ClassName:  wstr(siteManagerClass),
			IconSm:     icon,
		}
		registerClassExW.Call(uintptr(unsafe.Pointer(&class)))
	})

	logicalW, logicalH := 920, 610
	pixelW, pixelH := a.scale(logicalW), a.scale(logicalH)
	screenW, _, _ := getSystemMetrics.Call(smCxScreen)
	screenH, _, _ := getSystemMetrics.Call(smCyScreen)
	x := (int(screenW) - pixelW) / 2
	y := (int(screenH) - pixelH) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	words := nativeMenuWords(a.languageCode())
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr(siteManagerClass))),
		uintptr(unsafe.Pointer(wstr("Ghost FTP — "+words[5]))),
		siteWindowStyle,
		uintptr(x), uintptr(y), uintptr(pixelW), uintptr(pixelH),
		a.hwnd, 0, hinst, 0,
	)
	if hwnd == 0 {
		platform.ErrorDialog("Ghost FTP", words[5], a.tr("error.generic"))
		return
	}
	state := &siteManagerState{parent: a, hwnd: hwnd, profiles: append([]model.PublicProfile(nil), a.profiles...)}
	siteManagerStates.Store(hwnd, state)
	defer siteManagerStates.Delete(hwnd)
	applyDarkTitleBar(hwnd)
	if err := state.createControls(hinst); err != nil {
		destroyWindow.Call(hwnd)
		platform.ErrorDialog("Ghost FTP", words[5], err.Error())
		return
	}
	selectedID := a.selectedProfileID
	state.refillProfiles(selectedID)
	enableWindow.Call(a.hwnd, 0)
	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)

	var message msg
	for !state.closed {
		result, _, callErr := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			if callErr != nil && callErr != syscall.Errno(0) {
				a.setStatus(callErr.Error())
			}
			break
		}
		if result == 0 {
			postQuitMessage.Call(0)
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	enableWindow.Call(a.hwnd, 1)
	showWindow.Call(a.hwnd, swShow)

	if !state.connectAfter {
		return
	}
	if state.selected > 0 && state.selected <= len(state.profiles) {
		selectedID = state.profiles[state.selected-1].ID
		a.selectedProfileID = selectedID
		a.applyProfiles(state.profiles, nil)
		for index, profile := range a.profiles {
			if profile.ID == selectedID {
				sendMessageW.Call(a.profilesCombo, cbSetCurSel, uintptr(index+1), 0)
				a.selectProfile()
				break
			}
		}
	}
	a.connectNow()
}
