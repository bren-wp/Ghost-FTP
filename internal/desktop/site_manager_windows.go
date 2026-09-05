//go:build windows

package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	siteIDList     = 8101
	siteIDName     = 8102
	siteIDProtocol = 8103
	siteIDHost     = 8104
	siteIDPort     = 8105
	siteIDUser     = 8106
	siteIDLocal    = 8107
	siteIDRemote   = 8108
	siteIDKey      = 8109
	siteIDSecurity = 8110
	siteIDSave     = 8111
	siteIDDelete   = 8112
	siteIDConnect  = 8113
	siteIDClose    = 8114

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

type siteManagerState struct {
	parent       *app
	hwnd         uintptr
	list         uintptr
	name         uintptr
	protocol     uintptr
	host         uintptr
	port         uintptr
	user         uintptr
	localPath    uintptr
	remotePath   uintptr
	keyPath      uintptr
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
					} else {
						state.connectAfter = true
					}
					destroyWindow.Call(hwnd)
					return 0
				case siteIDClose:
					destroyWindow.Call(hwnd)
					return 0
				}
			}
		case wmCtlColorEdit, wmCtlColorBtn, wmCtlColorStatic, siteWMCtlColorListBox:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, windowColor())
			return state.parent.brush
		case wmClose:
			destroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
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
}

func (state *siteManagerState) syncProtocolPort() {
	protocol := state.protocolValue()
	current := strings.TrimSpace(getText(state.port))
	for _, spec := range protocolSpecs {
		if current == spec.Port {
			setText(state.port, protocolSpecs[protocolIndex(protocol)].Port)
			return
		}
	}
	if current == "" {
		setText(state.port, protocolSpecs[protocolIndex(protocol)].Port)
	}
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
	if index <= 0 || index > len(state.profiles) {
		state.selected = 0
		setText(state.name, "")
		state.setProtocol("ftp")
		setText(state.host, "")
		setText(state.port, "21")
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
		parts = append(parts, "● "+state.parent.tr("terminal.key_passphrase"))
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
		PrivateKeyPath: getText(state.keyPath),
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
	saved, err := state.parent.engine.SaveProfile(input)
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
	setText(state.parent.keyPath, getText(state.keyPath))
	setText(state.parent.pass, "")
	setText(state.parent.passphrase, "")
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

	state.list = mk("LISTBOX", "", wsBorder|wsTabStop|wsVScroll|siteLBSNotify|siteLBSNoIntegralHeight, 20, 48, 270, 446, siteIDList)
	if state.list != 0 {
		setWindowTheme.Call(state.list, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
	}

	label(parent.tr("column.name"), 310, 54, 130)
	state.name = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 450, 48, 430, 30, siteIDName)
	label(parent.tr("terminal.protocol"), 310, 96, 130)
	state.protocol = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, 450, 90, 180, 220, siteIDProtocol)
	for _, spec := range protocolSpecs {
		sendMessageW.Call(state.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(protocolLabel(parent.languageCode(), spec.Value)))))
	}
	label(parent.tr("terminal.server"), 310, 138, 130)
	state.host = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 450, 132, 430, 30, siteIDHost)
	label(parent.tr("terminal.port"), 310, 180, 130)
	state.port = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 450, 174, 110, 30, siteIDPort)
	label(parent.tr("terminal.username"), 580, 180, 110)
	state.user = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 690, 174, 190, 30, siteIDUser)
	label(parent.tr("column.local"), 310, 222, 130)
	state.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 450, 216, 430, 30, siteIDLocal)
	label(parent.tr("column.remote"), 310, 264, 130)
	state.remotePath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 450, 258, 430, 30, siteIDRemote)
	label(parent.tr("terminal.private_key"), 310, 306, 130)
	state.keyPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, 450, 300, 430, 30, siteIDKey)
	label(parent.tr("sftp.security"), 310, 348, 570)
	state.security = mk("STATIC", "", wsBorder, 310, 374, 570, 72, siteIDSecurity)

	state.save = mk("BUTTON", parent.tr("profile.save"), wsTabStop, 310, 464, 146, 34, siteIDSave)
	state.delete = mk("BUTTON", parent.tr("profile.delete"), wsTabStop, 466, 464, 146, 34, siteIDDelete)
	state.connect = mk("BUTTON", parent.tr("common.connect"), wsTabStop|siteBSDefPushButton, 622, 464, 120, 34, siteIDConnect)
	state.close = mk("BUTTON", parent.tr("common.cancel"), wsTabStop, 752, 464, 128, 34, siteIDClose)

	for _, control := range []uintptr{state.list, state.name, state.protocol, state.host, state.port, state.user, state.localPath, state.remotePath, state.keyPath, state.security, state.save, state.delete, state.connect, state.close} {
		if control == 0 {
			return fmt.Errorf("Site Manager control initialization failed")
		}
	}
	limitEdit(state.name, 120)
	limitEdit(state.host, 253)
	limitEdit(state.port, 5)
	limitEdit(state.user, 1024)
	limitEdit(state.localPath, 32767)
	limitEdit(state.remotePath, 4096)
	limitEdit(state.keyPath, 32767)
	return nil
}

func (a *app) openSiteManager() {
	if a == nil || a.hwnd == 0 || a.connectionBusy || a.connected {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	siteManagerOnce.Do(func() {
		cursor, _, _ := loadCursorW.Call(0, 32512)
		class := wndClassEx{
			CbSize:     uint32(unsafe.Sizeof(wndClassEx{})),
			WndProc:    siteManagerProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: a.brush,
			ClassName:  wstr(siteManagerClass),
		}
		registerClassExW.Call(uintptr(unsafe.Pointer(&class)))
	})

	logicalW, logicalH := 920, 560
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
			// Preserve the application's quit request if the main window is closed
			// by the operating system while this modal dialog is active.
			postQuitMessage.Call(0)
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	enableWindow.Call(a.hwnd, 1)
	showWindow.Call(a.hwnd, swShow)
	if state.connectAfter && state.selected > 0 && state.selected <= len(state.profiles) {
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
		a.connectNow()
	}
}
