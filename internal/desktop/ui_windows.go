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

	// Branded application header.
	a.brandTitle = mk("STATIC", brand.ProductName, 0, 0)
	a.brandSubtitle = mk("STATIC", "FTP • FTPS • SFTP  ·  Brzo upravljanje hostingom  ·  "+brand.Company, 0, 0)
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
	cue(a.host, "FTP/SFTP poslužitelj, npr. ftp.domena.hr")
	cue(a.user, "Korisničko ime, može korisnik@domena")
	cue(a.pass, "FTP / SFTP lozinka")

	// SFTP authentication row. Disabled for FTP/FTPS.
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

	// Section labels.
	a.sectionLocal = mk("STATIC", "LOKALNO RAČUNALO", 0, 0)
	a.sectionRemote = mk("STATIC", "POSLUŽITELJ", 0, 0)
	a.sectionTransfers = mk("STATIC", "PRIJENOSI", 0, 0)
	setFont(a.sectionLocal, a.smallFont)
	setFont(a.sectionRemote, a.smallFont)
	setFont(a.sectionTransfers, a.smallFont)

	// Local panel.
	a.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idLocalPath)
	a.localUp = mkButton("Gore", iconUp, buttonSubtle, idLocalUp)
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
	a.remoteUp = mkButton("Gore", iconUp, buttonSubtle, idRemoteUp)
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
	a.transferList = mk("SysListView32", "", wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idTransferList)
	a.status = mk("STATIC", "", 0, idStatus)
	a.statusVersion = mk("STATIC", brand.ProductName+" "+a.version+"  •  "+brand.Company, 0, 0)
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
	a.updateActionControls()
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

func (a *app) preferredWindowBounds() (x, y, width, height int) {
	width, height = 1500, 960
	x, y = 40, 30
	screenWRaw, _, _ := getSystemMetrics.Call(smCxScreen)
	screenHRaw, _, _ := getSystemMetrics.Call(smCyScreen)
	if screenWRaw == 0 || screenHRaw == 0 {
		return
	}
	screenW := a.unscale(int(screenWRaw))
	screenH := a.unscale(int(screenHRaw))
	if maxW := screenW - 48; maxW > 0 && width > maxW {
		width = maxW
	}
	if maxH := screenH - 72; maxH > 0 && height > maxH {
		height = maxH
	}
	if width < 940 {
		width = 940
	}
	if height < 680 {
		height = 680
	}
	x = (screenW - width) / 2
	y = (screenH - height) / 2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	return
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

func layoutPanelWidth(width int) int {
	centerW, panelGap := 126, 12
	if width < 1180 {
		centerW, panelGap = 96, 10
	}
	usable := width - 28 - centerW - 2*panelGap
	if usable < 0 {
		return 0
	}
	return usable / 2
}

func (a *app) resizeListColumns() {
	logicalWidth := 1180
	var client rect
	if a.hwnd != 0 {
		if r, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); r != 0 {
			logicalWidth = a.unscale(int(client.Right - client.Left))
		}
	}
	if logicalWidth < 900 {
		logicalWidth = 900
	}
	panelW := layoutPanelWidth(logicalWidth)
	typeW, sizeW, modifiedW := 70, 86, 128
	nameW := panelW - typeW - sizeW - modifiedW - 8
	if nameW < 120 {
		nameW = 120
	}
	fileWidths := []int{nameW, typeW, sizeW, modifiedW}
	for _, list := range []uintptr{a.localList, a.remoteList} {
		for i, width := range fileWidths {
			if list != 0 {
				sendMessageW.Call(list, lvmSetColumnWidth, uintptr(i), uintptr(a.scale(width)))
			}
		}
	}

	available := logicalWidth - 28
	directionW, statusW, progressW := 96, 160, 88
	pathsW := available - directionW - statusW - progressW - 8
	if pathsW < 360 {
		pathsW = 360
	}
	localW := pathsW / 2
	remoteW := pathsW - localW
	transferWidths := []int{directionW, localW, remoteW, statusW, progressW}
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

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (a *app) layout(width, height int) {
	width = a.unscale(width)
	height = a.unscale(height)
	if width < 900 {
		width = 900
	}
	if height < 620 {
		height = 620
	}
	margin, gap, rowH := 14, 8, 29
	compact := width < 1180

	// Branded header and compact application toolbar.
	headerY := 10
	badgeW := 142
	a.move(a.brandTitle, margin, headerY, 145, 35)
	subtitleW := width - (margin + 152) - badgeW - 2*margin
	if subtitleW < 260 {
		subtitleW = 260
	}
	a.move(a.brandSubtitle, margin+152, headerY+8, subtitleW, 22)
	a.move(a.connectionBadge, width-margin-badgeW, headerY+8, badgeW, 22)

	toolbarY := 51
	a.move(a.profilesCombo, margin, toolbarY, 300, rowH)
	a.move(a.saveProfile, margin+308, toolbarY, 126, rowH)
	a.move(a.removeProfile, margin+442, toolbarY, 126, rowH)
	a.move(a.settingsBtn, margin+576, toolbarY, 110, rowH)
	a.move(a.aboutBtn, margin+694, toolbarY, 112, rowH)

	sectionY := 168
	if !compact {
		// Connection row.
		y := 89
		x := margin
		a.move(a.protocol, x, y, 118, rowH)
		x += 118 + gap
		a.move(a.host, x, y, 280, rowH)
		x += 280 + gap
		a.move(a.port, x, y, 70, rowH)
		x += 70 + gap
		a.move(a.user, x, y, 230, rowH)
		x += 230 + gap
		a.move(a.pass, x, y, 170, rowH)
		x += 170 + gap
		a.move(a.connect, x, y, 106, rowH)
		x += 106 + gap
		a.move(a.disconnect, x, y, 106, rowH)

		// SFTP authentication row.
		y = 126
		a.move(a.keyPath, margin, y, 500, rowH)
		a.move(a.chooseKey, margin+508, y, 112, rowH)
		a.move(a.passphrase, margin+628, y, 240, rowH)
	} else {
		// Compact layout keeps all connection fields usable on common laptop
		// resolutions instead of letting the right side fall outside the client.
		available := width - 2*margin
		y := 89
		hostW := clampInt(available-118-70-2*gap, 300, 500)
		x := margin
		a.move(a.protocol, x, y, 118, rowH)
		x += 118 + gap
		a.move(a.host, x, y, hostW, rowH)
		x += hostW + gap
		a.move(a.port, x, y, 70, rowH)

		y = 126
		buttonW := 100
		fieldSpace := available - 2*buttonW - 3*gap
		userW := fieldSpace * 55 / 100
		passW := fieldSpace - userW
		x = margin
		a.move(a.user, x, y, userW, rowH)
		x += userW + gap
		a.move(a.pass, x, y, passW, rowH)
		x += passW + gap
		a.move(a.connect, x, y, buttonW, rowH)
		x += buttonW + gap
		a.move(a.disconnect, x, y, buttonW, rowH)

		y = 163
		chooseW, passphraseW := 112, 240
		keyW := available - chooseW - passphraseW - 2*gap
		if keyW < 280 {
			passphraseW = 190
			keyW = available - chooseW - passphraseW - 2*gap
		}
		a.move(a.keyPath, margin, y, keyW, rowH)
		a.move(a.chooseKey, margin+keyW+gap, y, chooseW, rowH)
		a.move(a.passphrase, margin+keyW+gap+chooseW+gap, y, passphraseW, rowH)
		sectionY = 204
	}

	// File panels.
	centerW, panelGap := 126, 12
	if compact {
		centerW, panelGap = 96, 10
	}
	usableW := width - 2*margin - centerW - 2*panelGap
	panelW := usableW / 2
	leftX := margin
	centerX := leftX + panelW + panelGap
	rightX := centerX + centerW + panelGap
	a.move(a.sectionLocal, leftX, sectionY, panelW, 20)
	a.move(a.sectionRemote, rightX, sectionY, panelW, 20)

	pathY := sectionY + 24
	buttonGap := 6
	upW, chooseW, refreshW := 64, 82, 78
	localPathW := panelW - upW - chooseW - refreshW - 3*buttonGap
	a.move(a.localPath, leftX, pathY, localPathW, rowH)
	x := leftX + localPathW + buttonGap
	a.move(a.localUp, x, pathY, upW, rowH)
	x += upW + buttonGap
	a.move(a.localChoose, x, pathY, chooseW, rowH)
	x += chooseW + buttonGap
	a.move(a.localRefresh, x, pathY, refreshW, rowH)

	remotePathW := panelW - upW - refreshW - 2*buttonGap
	a.move(a.remotePath, rightX, pathY, remotePathW, rowH)
	x = rightX + remotePathW + buttonGap
	a.move(a.remoteUp, x, pathY, upW, rowH)
	x += upW + buttonGap
	a.move(a.remoteRefresh, x, pathY, refreshW, rowH)

	actionY := pathY + rowH + 7
	mkdirW, renameW, deleteW, chmodW, actionGap := 92, 112, 94, 96, 8
	if compact {
		mkdirW, renameW, deleteW, chmodW, actionGap = 82, 98, 82, 90, 6
	}
	a.move(a.localMkdir, leftX, actionY, mkdirW, rowH)
	a.move(a.localRename, leftX+mkdirW+actionGap, actionY, renameW, rowH)
	a.move(a.localDelete, leftX+mkdirW+actionGap+renameW+actionGap, actionY, deleteW, rowH)
	a.move(a.remoteMkdir, rightX, actionY, mkdirW, rowH)
	a.move(a.remoteRename, rightX+mkdirW+actionGap, actionY, renameW, rowH)
	a.move(a.remoteDelete, rightX+mkdirW+actionGap+renameW+actionGap, actionY, deleteW, rowH)
	a.move(a.remoteChmod, rightX+mkdirW+actionGap+renameW+actionGap+deleteW+actionGap, actionY, chmodW, rowH)

	statusH := 24
	queueButtonsH := 33
	listY := actionY + rowH + 9
	contentBottom := height - statusH - 12
	availableH := contentBottom - listY
	fixedQueueH := 10 + 18 + 4 + queueButtonsH + 7
	queueH := clampInt(availableH/3, 80, 190)
	listH := availableH - fixedQueueH - queueH
	if listH < 120 {
		queueH -= 120 - listH
		if queueH < 80 {
			queueH = 80
		}
		listH = availableH - fixedQueueH - queueH
	}
	if listH < 90 {
		listH = 90
	}
	a.move(a.localList, leftX, listY, panelW, listH)
	a.move(a.remoteList, rightX, listY, panelW, listH)
	a.move(a.upload, centerX, listY+clampInt(listH/3, 45, 98), centerW, 38)
	a.move(a.download, centerX, listY+clampInt(listH/3+47, 92, 145), centerW, 38)

	queueLabelY := listY + listH + 10
	a.move(a.sectionTransfers, margin, queueLabelY, 160, 18)
	summaryW := width - margin - (margin + 160)
	if summaryW > 520 {
		summaryW = 520
	}
	a.move(a.transferSummary, margin+160, queueLabelY, summaryW, 18)
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
	a.move(a.status, margin, height-statusH-5, width-2*margin-250, statusH)
	a.move(a.statusVersion, width-margin-238, height-statusH-5, 238, statusH)
	a.resizeListColumns()
	invalidateRect.Call(a.hwnd, 0, 1)
}

func (a *app) move(hwnd uintptr, x, y, w, h int) {
	if hwnd != 0 {
		moveWindow.Call(hwnd, uintptr(a.scale(x)), uintptr(a.scale(y)), uintptr(a.scale(w)), uintptr(a.scale(h)), 1)
	}
}

func (a *app) setRemoteControls(enabled bool) {
	for _, h := range []uintptr{a.remotePath, a.remoteUp, a.remoteRefresh, a.remoteList} {
		setControlEnabled(h, enabled)
	}
	setControlEnabled(a.remoteMkdir, enabled && !a.connectionBusy)
	a.updateActionControls()
}

func (a *app) updateProtocolControls() {
	sftp := a.protocolValue() == "sftp" && !a.connected && !a.connectionBusy
	for _, h := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
		setControlEnabled(h, sftp)
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
