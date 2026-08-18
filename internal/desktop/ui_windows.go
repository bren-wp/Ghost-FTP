//go:build windows

package desktop

import (
	"brendigo.com/byftp/internal/brand"
	"fmt"
	"syscall"
	"unsafe"
)

func createUIFont(height int32, weight uint32) uintptr {
	fontName := wstr("Segoe UI")
	font, _, _ := createFontW.Call(
		uintptr(uint32(height)), 0, 0, 0, uintptr(weight), 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(fontName)),
	)
	return font
}

func (a *app) createControls(hinst uintptr) error {
	a.createFonts()

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
	setFont := func(h, f uintptr) {
		if h != 0 && f != 0 {
			sendMessageW.Call(h, wmSetFont, f, 1)
		}
	}
	mkButton := func(label, icon string, variant buttonVariant, id int) uintptr {
		h := mk("BUTTON", label, bsOwnerDraw|wsTabStop, id)
		return a.registerButton(h, icon, label, variant)
	}

	// Svaki zasebni EXE dobiva stvarni vlastiti identitet, ali koristi isti
	// provjereni Win32 file-manager core.
	a.brandTitle = mk("STATIC", clientProductName(), 0, 0)
	a.brandSubtitle = mk("STATIC", clientSubtitle(), 0, 0)
	a.connectionBadge = mk("STATIC", "● NIJE POVEZANO", 0, 0)
	setFont(a.brandTitle, a.titleFont)
	setFont(a.brandSubtitle, a.smallFont)
	setFont(a.connectionBadge, a.smallFont)

	// Profile / application toolbar.
	a.profilesCombo = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, idProfiles)
	sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr("Brzi spoj (bez profila)"))))
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, 0, 0)
	a.saveProfile = mkButton("Spremi profil", iconSave, buttonDefault, idSaveProfile)
	a.removeProfile = mkButton("Obriši profil", iconDelete, buttonDanger, idRemoveProfile)
	a.settingsBtn = mkButton("Postavke", iconSettings, buttonSubtle, idSettings)
	a.aboutBtn = mkButton("O programu", iconInfo, buttonSubtle, idAbout)

	// Connection row.
	a.protocol = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, idProtocol)
	for _, spec := range protocolSpecs {
		sendMessageW.Call(a.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(spec.Label))))
	}
	sendMessageW.Call(a.protocol, cbSetCurSel, 0, 0)
	a.host = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idHost)
	a.port = mk("EDIT", protocolSpecs[0].Port, wsBorder|wsTabStop|esAutoHScroll, idPort)
	a.user = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idUser)
	a.pass = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, idPass)
	a.connect = mkButton("Poveži", iconConnect, buttonAccent, idConnect)
	a.disconnect = mkButton("Prekini", iconDisconnect, buttonDanger, idDisconnect)
	enableWindow.Call(a.disconnect, 0)
	cue(a.host, "Poslužitelj")
	cue(a.user, "Korisničko ime")
	cue(a.pass, "Lozinka")

	// SFTP authentication row. FTP Client ga uopće ne prikazuje; Suite ga
	// aktivira samo kada je izabran SFTP, a SFTP Client ga uvijek koristi.
	a.keyPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idKeyPath)
	a.chooseKey = mkButton("Privatni ključ…", iconPermissions, buttonDefault, idChooseKey)
	a.passphrase = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, idPassphrase)
	limitEdit(a.host, 253)
	limitEdit(a.port, 5)
	limitEdit(a.user, 1024)
	limitEdit(a.pass, 8192)
	limitEdit(a.keyPath, 32767)
	limitEdit(a.passphrase, 8192)
	cue(a.keyPath, "Privatni ključ za SFTP (opcionalno)")
	cue(a.passphrase, "Zaporka privatnog ključa")
	if !clientShowsSFTPAuth() {
		for _, h := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
			showWindow.Call(h, 0)
		}
	}

	// Section labels.
	a.sectionLocal = mk("STATIC", "LOKALNO RAČUNALO", 0, 0)
	a.sectionRemote = mk("STATIC", "POSLUŽITELJ", 0, 0)
	a.sectionTransfers = mk("STATIC", "PRIJENOSI", 0, 0)
	setFont(a.sectionLocal, a.smallFont)
	setFont(a.sectionRemote, a.smallFont)
	setFont(a.sectionTransfers, a.smallFont)

	// Local panel.
	a.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idLocalPath)
	a.localUp = mkButton("", iconUp, buttonSubtle, idLocalUp)
	a.localChoose = mkButton("Mapa…", iconOpenLocal, buttonDefault, idLocalChoose)
	a.localRefresh = mkButton("Osvježi", iconRefresh, buttonSubtle, idLocalRefresh)
	a.localMkdir = mkButton("Nova mapa", iconNewFolder, buttonDefault, idLocalMkdir)
	a.localRename = mkButton("Preimenuj", iconRename, buttonDefault, idLocalRename)
	a.localDelete = mkButton("Obriši", iconDelete, buttonDanger, idLocalDelete)
	a.localList = mk("SysListView32", "", wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idLocalList)

	// Remote panel.
	a.remotePath = mk("EDIT", "/", wsBorder|wsTabStop|esAutoHScroll, idRemotePath)
	limitEdit(a.localPath, 32767)
	limitEdit(a.remotePath, 4096)
	a.remoteUp = mkButton("", iconUp, buttonSubtle, idRemoteUp)
	a.remoteRefresh = mkButton("Osvježi", iconRefresh, buttonSubtle, idRemoteRefresh)
	a.remoteMkdir = mkButton("Nova mapa", iconNewFolder, buttonDefault, idRemoteMkdir)
	a.remoteRename = mkButton("Preimenuj", iconRename, buttonDefault, idRemoteRename)
	a.remoteDelete = mkButton("Obriši", iconDelete, buttonDanger, idRemoteDelete)
	a.remoteChmod = mkButton("Dozvole", iconPermissions, buttonDefault, idRemoteChmod)
	a.remoteList = mk("SysListView32", "", wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idRemoteList)

	a.upload = mkButton("Pošalji", iconUpload, buttonAccent, idUpload)
	a.download = mkButton("Preuzmi", iconDownload, buttonAccent, idDownload)

	// Transfer queue.
	a.pauseQueue = mkButton("Pauziraj", iconPause, buttonDefault, idPauseQueue)
	a.resumeQueue = mkButton("Nastavi", iconPlay, buttonDefault, idResumeQueue)
	a.cancelJob = mkButton("Otkaži", iconCancel, buttonDanger, idCancelJob)
	a.retryJob = mkButton("Ponovi", iconSync, buttonDefault, idRetryJob)
	a.clearQueue = mkButton("Očisti završene", iconClear, buttonSubtle, idClearQueue)
	a.transferList = mk("SysListView32", "", wsBorder|lvsReport|lvsShowSelAlways, idTransferList)
	a.status = mk("STATIC", "", 0, idStatus)
	a.statusVersion = mk("STATIC", clientProductName()+" "+a.version+"  •  "+brand.Company, 0, 0)
	a.transferSummary = mk("STATIC", "0 aktivnih  •  0 na čekanju  •  0 završeno", 0, 0)
	setFont(a.status, a.smallFont)
	setFont(a.statusVersion, a.smallFont)
	setFont(a.transferSummary, a.smallFont)

	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		sendMessageW.Call(list, lvmSetExtendedListViewStyle, 0, lvsExFullRowSelect|lvsExDoubleBuffer)
		sendMessageW.Call(list, lvmSetBkColor, 0, listColor())
		sendMessageW.Call(list, lvmSetTextBkColor, 0, listColor())
		sendMessageW.Call(list, lvmSetTextColor, 0, textColor())
	}
	attachSystemImageList(a.localList)
	attachSystemImageList(a.remoteList)
	a.setupFileColumns(a.localList)
	a.setupFileColumns(a.remoteList)
	a.setupTransferColumns(a.transferList)
	a.setRemoteControls(false)
	a.updateProtocolControls()
	setText(a.hwnd, clientProductName()+" "+a.version+" — "+brand.Company)
	return a.validateControls()
}

func windowDPI(hwnd uintptr) uint32 {
	if err := getDpiForWindow.Find(); err == nil {
		if dpi, _, _ := getDpiForWindow.Call(hwnd); dpi >= 96 {
			return uint32(dpi)
		}
	}
	return 96
}

func (a *app) scale(v int) int {
	dpi := a.dpi
	if dpi == 0 {
		dpi = 96
	}
	return (v*int(dpi) + 48) / 96
}

func (a *app) unscale(v int) int {
	dpi := a.dpi
	if dpi == 0 {
		dpi = 96
	}
	return (v*96 + int(dpi)/2) / int(dpi)
}

func (a *app) createFonts() {
	a.font = createUIFont(int32(-a.scale(15)), 400)
	a.titleFont = createUIFont(int32(-a.scale(27)), 700)
	a.smallFont = createUIFont(int32(-a.scale(13)), 400)
	a.iconFont = createIconFont(int32(-a.scale(16)))
}

func (a *app) applyDPI(dpi uint32) {
	if dpi < 96 {
		dpi = 96
	}
	if a.dpi == dpi {
		return
	}
	oldFonts := []uintptr{a.font, a.titleFont, a.smallFont, a.iconFont}
	a.dpi = dpi
	a.createFonts()
	for _, h := range a.defaultFontControls() {
		if h != 0 && a.font != 0 {
			sendMessageW.Call(h, wmSetFont, a.font, 1)
		}
	}
	for _, h := range []uintptr{a.brandSubtitle, a.connectionBadge, a.sectionLocal, a.sectionRemote, a.sectionTransfers, a.status, a.statusVersion, a.transferSummary} {
		if h != 0 && a.smallFont != 0 {
			sendMessageW.Call(h, wmSetFont, a.smallFont, 1)
		}
	}
	if a.brandTitle != 0 && a.titleFont != 0 {
		sendMessageW.Call(a.brandTitle, wmSetFont, a.titleFont, 1)
	}
	a.resizeListColumns()
	for _, f := range oldFonts {
		if f != 0 {
			deleteObject.Call(f)
		}
	}
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) defaultFontControls() []uintptr {
	return []uintptr{
		a.profilesCombo, a.saveProfile, a.removeProfile, a.settingsBtn, a.aboutBtn,
		a.protocol, a.host, a.port, a.user, a.pass, a.keyPath, a.chooseKey, a.passphrase, a.connect, a.disconnect,
		a.localPath, a.localUp, a.localRefresh, a.localChoose, a.localList, a.localMkdir, a.localRename, a.localDelete,
		a.remotePath, a.remoteUp, a.remoteRefresh, a.remoteList, a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod,
		a.upload, a.download, a.transferList, a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue,
	}
}

func (a *app) resizeListColumns() {
	fileWidths := []int{300, 92, 104, 150}
	for _, list := range []uintptr{a.localList, a.remoteList} {
		for i, width := range fileWidths {
			if list != 0 {
				sendMessageW.Call(list, lvmSetColumnWidth, uintptr(i), uintptr(a.scale(width)))
			}
		}
	}
	transferWidths := []int{100, 325, 325, 180, 92}
	for i, width := range transferWidths {
		if a.transferList != 0 {
			sendMessageW.Call(a.transferList, lvmSetColumnWidth, uintptr(i), uintptr(a.scale(width)))
		}
	}
}

func applyDarkTitleBar(hwnd uintptr) {
	value := int32(1)
	// DWMWA_USE_IMMERSIVE_DARK_MODE is 20 on current Windows 10/11.
	_, _, _ = dwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
}

func applyDarkControl(hwnd uintptr, class string) {
	switch class {
	case "SysListView32", "COMBOBOX", "BUTTON":
		setWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
	case "EDIT":
		setWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_CFD"))), 0)
	}
}

func cue(hwnd uintptr, text string) {
	sendMessageW.Call(hwnd, emSetCueBanner, 1, uintptr(unsafe.Pointer(wstr(text))))
}

func limitEdit(hwnd uintptr, maxChars uintptr) {
	if hwnd != 0 && maxChars > 0 {
		sendMessageW.Call(hwnd, emSetLimitText, maxChars, 0)
	}
}

func (a *app) setupFileColumns(list uintptr) {
	a.insertColumn(list, 0, "Naziv", 300)
	a.insertColumn(list, 1, "Vrsta", 92)
	a.insertColumn(list, 2, "Veličina", 104)
	a.insertColumn(list, 3, "Izmijenjeno", 150)
}

func (a *app) setupTransferColumns(list uintptr) {
	a.insertColumn(list, 0, "Smjer", 100)
	a.insertColumn(list, 1, "Lokalno", 325)
	a.insertColumn(list, 2, "Poslužitelj", 325)
	a.insertColumn(list, 3, "Status", 180)
	a.insertColumn(list, 4, "Napredak", 92)
}

func (a *app) insertColumn(list uintptr, idx int, title string, width int) {
	text := syscall.StringToUTF16(title)
	c := lvColumn{Mask: lvcfText | lvcfWidth | lvcfFmt, Cx: int32(a.scale(width)), Text: &text[0], Fmt: 0}
	sendMessageW.Call(list, lvmInsertColumnW, uintptr(idx), uintptr(unsafe.Pointer(&c)))
}

func (a *app) layout(width, height int) {
	width = a.unscale(width)
	height = a.unscale(height)
	if width < 1180 {
		width = 1180
	}
	if height < 760 {
		height = 760
	}
	margin, gap, rowH := 14, 8, 29

	// Branded header and compact application toolbar.
	headerY := 10
	a.move(a.brandTitle, margin, headerY, 220, 35)
	a.move(a.brandSubtitle, margin+228, headerY+8, 530, 22)
	a.move(a.connectionBadge, width-112, headerY+8, 98, 22)

	toolbarY := 51
	a.move(a.profilesCombo, margin, toolbarY, 300, rowH)
	a.move(a.saveProfile, margin+308, toolbarY, 126, rowH)
	a.move(a.removeProfile, margin+442, toolbarY, 126, rowH)
	a.move(a.settingsBtn, margin+576, toolbarY, 110, rowH)
	a.move(a.aboutBtn, margin+694, toolbarY, 112, rowH)

	// Connection row.
	y := 89
	x := margin
	a.move(a.protocol, x, y, 118, rowH)
	x += 118 + gap
	a.move(a.host, x, y, 300, rowH)
	x += 300 + gap
	a.move(a.port, x, y, 70, rowH)
	x += 70 + gap
	a.move(a.user, x, y, 190, rowH)
	x += 190 + gap
	a.move(a.pass, x, y, 190, rowH)
	x += 190 + gap
	a.move(a.connect, x, y, 106, rowH)
	x += 106 + gap
	a.move(a.disconnect, x, y, 106, rowH)

	// SFTP authentication row. FTP-only klijent koristi kompaktniji raspored.
	sectionY := 168
	if clientShowsSFTPAuth() {
		y = 126
		a.move(a.keyPath, margin, y, 500, rowH)
		a.move(a.chooseKey, margin+508, y, 112, rowH)
		a.move(a.passphrase, margin+628, y, 240, rowH)
	} else {
		sectionY = 130
	}

	// File panels.
	centerW, panelGap := 126, 12
	usableW := width - 2*margin - centerW - 2*panelGap
	panelW := usableW / 2
	leftX := margin
	centerX := leftX + panelW + panelGap
	rightX := centerX + centerW + panelGap
	a.move(a.sectionLocal, leftX, sectionY, panelW, 20)
	a.move(a.sectionRemote, rightX, sectionY, panelW, 20)

	pathY := sectionY + 24
	a.move(a.localPath, leftX, pathY, panelW-218, rowH)
	a.move(a.localUp, leftX+panelW-210, pathY, 38, rowH)
	a.move(a.localChoose, leftX+panelW-164, pathY, 82, rowH)
	a.move(a.localRefresh, leftX+panelW-74, pathY, 74, rowH)
	a.move(a.remotePath, rightX, pathY, panelW-132, rowH)
	a.move(a.remoteUp, rightX+panelW-124, pathY, 38, rowH)
	a.move(a.remoteRefresh, rightX+panelW-78, pathY, 78, rowH)

	actionY := pathY + rowH + 7
	a.move(a.localMkdir, leftX, actionY, 92, rowH)
	a.move(a.localRename, leftX+100, actionY, 112, rowH)
	a.move(a.localDelete, leftX+220, actionY, 94, rowH)
	a.move(a.remoteMkdir, rightX, actionY, 92, rowH)
	a.move(a.remoteRename, rightX+100, actionY, 112, rowH)
	a.move(a.remoteDelete, rightX+220, actionY, 94, rowH)
	a.move(a.remoteChmod, rightX+322, actionY, 96, rowH)

	statusH := 24
	queueH := 190
	queueButtonsH := 33
	listY := actionY + rowH + 9
	listH := height - listY - queueH - queueButtonsH - statusH - 62
	if listH < 280 {
		listH = 280
	}
	a.move(a.localList, leftX, listY, panelW, listH)
	a.move(a.remoteList, rightX, listY, panelW, listH)
	a.move(a.upload, centerX, listY+98, centerW, 38)
	a.move(a.download, centerX, listY+145, centerW, 38)

	queueLabelY := listY + listH + 10
	a.move(a.sectionTransfers, margin, queueLabelY, 180, 18)
	a.move(a.transferSummary, margin+180, queueLabelY, 430, 18)
	queueButtonsY := queueLabelY + 22
	qx := margin
	for _, item := range []struct {
		h uintptr
		w int
	}{{a.pauseQueue, 108}, {a.resumeQueue, 108}, {a.cancelJob, 96}, {a.retryJob, 96}, {a.clearQueue, 146}} {
		a.move(item.h, qx, queueButtonsY, item.w, queueButtonsH)
		qx += item.w + gap
	}
	queueY := queueButtonsY + queueButtonsH + 7
	a.move(a.transferList, margin, queueY, width-2*margin, queueH)
	a.move(a.status, margin, height-statusH-5, width-2*margin-320, statusH)
	a.move(a.statusVersion, width-margin-308, height-statusH-5, 308, statusH)
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) move(hwnd uintptr, x, y, w, h int) {
	if hwnd != 0 {
		moveWindow.Call(hwnd, uintptr(a.scale(x)), uintptr(a.scale(y)), uintptr(a.scale(w)), uintptr(a.scale(h)), 1)
	}
}

func (a *app) setRemoteControls(enabled bool) {
	val := uintptr(0)
	if enabled {
		val = 1
	}
	for _, h := range []uintptr{
		a.remotePath, a.remoteUp, a.remoteRefresh, a.remoteList,
		a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod,
		a.upload, a.download,
	} {
		enableWindow.Call(h, val)
	}
}

func (a *app) updateProtocolControls() {
	if clientHasFixedProtocol() {
		enableWindow.Call(a.protocol, 0)
	}
	sftp := a.protocolValue() == "sftp" && !a.connected && clientShowsSFTPAuth()
	val := uintptr(0)
	if sftp {
		val = 1
	}
	for _, h := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
		enableWindow.Call(h, val)
	}
}

func (a *app) validateControls() error {
	controls := []struct {
		name string
		h    uintptr
	}{
		{"naslov", a.brandTitle}, {"podnaslov", a.brandSubtitle}, {"status veze", a.connectionBadge},
		{"profili", a.profilesCombo}, {"spremi profil", a.saveProfile}, {"obriši profil", a.removeProfile}, {"postavke", a.settingsBtn}, {"o programu", a.aboutBtn},
		{"protokol", a.protocol}, {"poslužitelj", a.host}, {"port", a.port}, {"korisnik", a.user}, {"lozinka", a.pass},
		{"privatni ključ", a.keyPath}, {"odabir ključa", a.chooseKey}, {"zaporka ključa", a.passphrase}, {"poveži", a.connect}, {"prekini", a.disconnect},
		{"lokalna putanja", a.localPath}, {"lokalno gore", a.localUp}, {"lokalni odabir", a.localChoose}, {"lokalno osvježi", a.localRefresh},
		{"lokalna lista", a.localList}, {"lokalna nova mapa", a.localMkdir}, {"lokalno preimenuj", a.localRename}, {"lokalno obriši", a.localDelete},
		{"udaljena putanja", a.remotePath}, {"udaljeno gore", a.remoteUp}, {"udaljeno osvježi", a.remoteRefresh}, {"udaljena lista", a.remoteList},
		{"udaljena nova mapa", a.remoteMkdir}, {"udaljeno preimenuj", a.remoteRename}, {"udaljeno obriši", a.remoteDelete}, {"udaljene dozvole", a.remoteChmod},
		{"pošalji", a.upload}, {"preuzmi", a.download}, {"lista prijenosa", a.transferList}, {"pauziraj", a.pauseQueue}, {"nastavi", a.resumeQueue},
		{"otkaži prijenos", a.cancelJob}, {"ponovi prijenos", a.retryJob}, {"očisti prijenose", a.clearQueue},
		{"status", a.status}, {"verzija", a.statusVersion}, {"sažetak prijenosa", a.transferSummary},
	}
	for _, control := range controls {
		if control.h == 0 {
			return fmt.Errorf("nije moguće inicijalizirati kontrolu: %s", control.name)
		}
	}
	return nil
}
