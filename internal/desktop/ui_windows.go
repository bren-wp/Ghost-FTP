//go:build windows

package desktop

import (
	"fmt"
	"github.com/bren-wp/Ghost-FTP/internal/brand"
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

	// English is the canonical startup locale. loadSettings applies the saved
	// locale immediately after the controls exist.
	a.brandTitle = mk("STATIC", brand.ProductName, 0, 0)
	a.brandSubtitle = mk("STATIC", a.tr("app.subtitle"), 0, 0)
	a.connectionBadge = mk("STATIC", a.tr("badge.disconnected"), 0, 0)
	setFont(a.brandTitle, a.titleFont)
	setFont(a.brandSubtitle, a.smallFont)
	setFont(a.connectionBadge, a.smallFont)

	// Language is placed in the header so long translated toolbar labels never
	// compete with the profile/action row on smaller laptops.
	a.languageCombo = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, idLanguage)
	a.populateLanguageCombo()

	// Profile / application toolbar.
	a.profilesCombo = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, idProfiles)
	sendMessageW.Call(a.profilesCombo, cbAddString, 0, uintptr(unsafe.Pointer(wstr(a.tr("profile.quick")))))
	sendMessageW.Call(a.profilesCombo, cbSetCurSel, 0, 0)
	a.saveProfile = mkButton(a.tr("profile.save"), iconSave, buttonDefault, idSaveProfile)
	a.removeProfile = mkButton(a.tr("profile.delete"), iconDelete, buttonDanger, idRemoveProfile)
	a.siteManagerBtn = mkButton(nativeMenuWords(a.languageCode())[5], iconOpenLocal, buttonDefault, idSiteManager)
	a.settingsBtn = mkButton(a.tr("common.settings"), iconSettings, buttonSubtle, idSettings)
	a.aboutBtn = mkButton(a.tr("common.about"), iconInfo, buttonSubtle, idAbout)

	// Connection row.
	a.protocol = mk("COMBOBOX", "", cbsDropDownList|wsTabStop|wsVScroll, idProtocol)
	for _, spec := range protocolSpecs {
		sendMessageW.Call(a.protocol, cbAddString, 0, uintptr(unsafe.Pointer(wstr(protocolLabel(a.languageCode(), spec.Value)))))
	}
	sendMessageW.Call(a.protocol, cbSetCurSel, 0, 0)
	a.host = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idHost)
	a.port = mk("EDIT", protocolSpecs[0].Port, wsBorder|wsTabStop|esAutoHScroll, idPort)
	a.user = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idUser)
	a.pass = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, idPass)
	a.connect = mkButton(a.tr("common.connect"), iconConnect, buttonAccent, idConnect)
	a.disconnect = mkButton(a.tr("common.disconnect"), iconDisconnect, buttonDanger, idDisconnect)
	enableWindow.Call(a.disconnect, 0)
	cue(a.host, a.tr("cue.host"))
	cue(a.user, a.tr("cue.user"))
	cue(a.pass, a.tr("cue.password"))

	// SFTP authentication row. Disabled for FTP/FTPS.
	a.keyPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idKeyPath)
	a.chooseKey = mkButton(a.tr("auth.private_key"), iconPermissions, buttonDefault, idChooseKey)
	a.passphrase = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll|esPassword, idPassphrase)
	limitEdit(a.host, 253)
	limitEdit(a.port, 5)
	limitEdit(a.user, 1024)
	limitEdit(a.pass, 8192)
	limitEdit(a.keyPath, 32767)
	limitEdit(a.passphrase, 8192)
	cue(a.keyPath, a.tr("cue.private_key"))
	cue(a.passphrase, a.tr("cue.passphrase"))

	// Section labels.
	a.sectionLocal = mk("STATIC", a.tr("section.local"), 0, 0)
	a.sectionRemote = mk("STATIC", a.tr("section.remote"), 0, 0)
	a.sectionTransfers = mk("STATIC", a.tr("section.transfers"), 0, 0)
	setFont(a.sectionLocal, a.smallFont)
	setFont(a.sectionRemote, a.smallFont)
	setFont(a.sectionTransfers, a.smallFont)

	// Local panel.
	a.localPath = mk("EDIT", "", wsBorder|wsTabStop|esAutoHScroll, idLocalPath)
	a.localUp = mkButton(a.tr("common.up"), iconUp, buttonSubtle, idLocalUp)
	a.localChoose = mkButton(a.tr("common.folder"), iconOpenLocal, buttonDefault, idLocalChoose)
	a.localRefresh = mkButton(a.tr("common.refresh"), iconRefresh, buttonSubtle, idLocalRefresh)
	a.localMkdir = mkButton(a.tr("common.new_folder"), iconNewFolder, buttonDefault, idLocalMkdir)
	a.localRename = mkButton(a.tr("common.rename"), iconRename, buttonDefault, idLocalRename)
	a.localDelete = mkButton(a.tr("common.delete"), iconDelete, buttonDanger, idLocalDelete)
	a.localList = mk("SysListView32", "", wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idLocalList)

	// Remote panel.
	a.remotePath = mk("EDIT", "/", wsBorder|wsTabStop|esAutoHScroll, idRemotePath)
	limitEdit(a.localPath, 32767)
	limitEdit(a.remotePath, 4096)
	a.remoteUp = mkButton(a.tr("common.up"), iconUp, buttonSubtle, idRemoteUp)
	a.remoteRefresh = mkButton(a.tr("common.refresh"), iconRefresh, buttonSubtle, idRemoteRefresh)
	a.remoteMkdir = mkButton(a.tr("common.new_folder"), iconNewFolder, buttonDefault, idRemoteMkdir)
	a.remoteRename = mkButton(a.tr("common.rename"), iconRename, buttonDefault, idRemoteRename)
	a.remoteDelete = mkButton(a.tr("common.delete"), iconDelete, buttonDanger, idRemoteDelete)
	a.remoteChmod = mkButton(a.tr("common.permissions"), iconPermissions, buttonDefault, idRemoteChmod)
	a.remoteList = mk("SysListView32", "", wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idRemoteList)

	a.upload = mkButton(a.tr("transfer.upload"), iconUpload, buttonAccent, idUpload)
	a.download = mkButton(a.tr("transfer.download"), iconDownload, buttonAccent, idDownload)

	// Transfer queue.
	a.pauseQueue = mkButton(a.tr("transfer.pause"), iconPause, buttonDefault, idPauseQueue)
	a.resumeQueue = mkButton(a.tr("transfer.resume"), iconPlay, buttonDefault, idResumeQueue)
	a.cancelJob = mkButton(a.tr("common.cancel"), iconCancel, buttonDanger, idCancelJob)
	a.retryJob = mkButton(a.tr("transfer.retry"), iconSync, buttonDefault, idRetryJob)
	a.clearQueue = mkButton(a.tr("transfer.clear"), iconClear, buttonSubtle, idClearQueue)
	a.transferList = mk("SysListView32", "", wsBorder|wsTabStop|lvsReport|lvsShowSelAlways, idTransferList)
	a.status = mk("STATIC", "", 0, idStatus)
	a.statusVersion = mk("STATIC", brand.ProductName+" "+a.version+"  •  "+brand.Company, 0, 0)
	a.transferSummary = mk("STATIC", a.tr("transfer.summary", 0, 0, 0), 0, 0)
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
	a.setupFileColumns(a.localList, false)
	a.setupFileColumns(a.remoteList, true)
	a.setupTransferColumns(a.transferList)
	a.setRemoteControls(false)
	a.updateProtocolControls()
	a.updateActionControls()
	a.applyLanguage(a.languageCode())
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
	if width < premiumMinWidth {
		width = premiumMinWidth
	}
	if height < premiumMinHeight {
		height = premiumMinHeight
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
	invalidateRect.Call(a.hwnd, 0, 0)
}

func (a *app) defaultFontControls() []uintptr {
	return []uintptr{
		a.profilesCombo, a.languageCombo, a.saveProfile, a.removeProfile, a.siteManagerBtn, a.settingsBtn, a.aboutBtn,
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
	// Type/status text can be longer in German/French/Portuguese; reserve a
	// little more room while keeping the filename column elastic.
	typeW, sizeW, modifiedW := 82, 92, 132
	localNameW := panelW - typeW - sizeW - modifiedW - 8
	if localNameW < 120 {
		localNameW = 120
	}
	for i, width := range []int{localNameW, typeW, sizeW, modifiedW} {
		if a.localList != 0 {
			sendMessageW.Call(a.localList, lvmSetColumnWidth, uintptr(i), uintptr(a.scale(width)))
		}
	}

	permissionsW := 112
	remoteNameW := panelW - typeW - sizeW - modifiedW - permissionsW - 8
	if remoteNameW < 100 {
		remoteNameW = 100
	}
	for i, width := range []int{remoteNameW, typeW, sizeW, modifiedW, permissionsW} {
		if a.remoteList != 0 {
			sendMessageW.Call(a.remoteList, lvmSetColumnWidth, uintptr(i), uintptr(a.scale(width)))
		}
	}

	available := logicalWidth - 28
	directionW, statusW, progressW := 108, 184, 92
	pathsW := available - directionW - statusW - progressW - 8
	if pathsW < 340 {
		pathsW = 340
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
	enableImmersiveDarkMode(hwnd)
	value := int32(1)
	_, _, _ = dwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
}

func applyDarkControl(hwnd uintptr, class string) {
	switch class {
	case "SysListView32", "BUTTON":
		setWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
	case "COMBOBOX", "EDIT":
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

func (a *app) setupFileColumns(list uintptr, remote bool) {
	a.insertColumn(list, 0, a.tr("column.name"), 300)
	a.insertColumn(list, 1, a.tr("column.type"), 92)
	a.insertColumn(list, 2, a.tr("column.size"), 104)
	a.insertColumn(list, 3, a.tr("column.modified"), 150)
	if remote {
		a.insertColumn(list, 4, a.tr("common.permissions"), 112)
	}
}

func (a *app) setupTransferColumns(list uintptr) {
	a.insertColumn(list, 0, a.tr("column.direction"), 100)
	a.insertColumn(list, 1, a.tr("column.local"), 325)
	a.insertColumn(list, 2, a.tr("column.remote"), 325)
	a.insertColumn(list, 3, a.tr("column.status"), 180)
	a.insertColumn(list, 4, a.tr("column.progress"), 92)
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
	if width < premiumMinWidth {
		width = premiumMinWidth
	}
	if height < premiumMinHeight {
		height = premiumMinHeight
	}
	margin, gap, rowH := 14, 8, 29
	compact := width < 1180

	// Header keeps the locale selector visible at every supported width without
	// stealing space from file-transfer actions.
	headerY := 10
	badgeW, languageW := 142, 170
	if compact {
		languageW = 142
	}
	// Reserve a stable icon gutter and enough width for the complete product
	// name. The previous 106 px post-layout override visibly clipped “FTP”.
	a.move(a.brandTitle, 54, headerY, 126, 35)
	subtitleX := 188
	languageX := width - margin - badgeW - gap - languageW
	subtitleW := languageX - gap - subtitleX
	if subtitleW < 180 {
		subtitleW = 180
	}
	a.move(a.brandSubtitle, subtitleX, headerY+8, subtitleW, 22)
	a.move(a.languageCombo, languageX, headerY+3, languageW, rowH)
	a.move(a.connectionBadge, width-margin-badgeW, headerY+8, badgeW, 22)

	toolbarY := 51
	availableToolbar := width - 2*margin
	buttonWidths := []int{120, 120, 132, 108, 108}
	if compact {
		buttonWidths = []int{112, 112, 120, 100, 100}
	}
	fixedButtons := 0
	for _, buttonW := range buttonWidths {
		fixedButtons += buttonW
	}
	profileW := clampInt(availableToolbar-fixedButtons-len(buttonWidths)*gap, 200, 340)
	x := margin
	a.move(a.profilesCombo, x, toolbarY, profileW, rowH)
	x += profileW + gap
	for index, h := range []uintptr{a.saveProfile, a.removeProfile, a.siteManagerBtn, a.settingsBtn, a.aboutBtn} {
		a.move(h, x, toolbarY, buttonWidths[index], rowH)
		x += buttonWidths[index] + gap
	}

	sectionY := 168
	if !compact {
		y := 89
		x := margin
		a.move(a.protocol, x, y, 138, rowH)
		x += 138 + gap
		a.move(a.host, x, y, 270, rowH)
		x += 270 + gap
		a.move(a.port, x, y, 70, rowH)
		x += 70 + gap
		a.move(a.user, x, y, 220, rowH)
		x += 220 + gap
		a.move(a.pass, x, y, 166, rowH)
		x += 166 + gap
		a.move(a.connect, x, y, 116, rowH)
		x += 116 + gap
		a.move(a.disconnect, x, y, 116, rowH)

		y = 126
		a.move(a.keyPath, margin, y, 500, rowH)
		a.move(a.chooseKey, margin+508, y, 136, rowH)
		a.move(a.passphrase, margin+652, y, 240, rowH)
	} else {
		available := width - 2*margin
		y := 89
		protocolW := 138
		hostW := clampInt(available-protocolW-70-2*gap, 280, 480)
		x := margin
		a.move(a.protocol, x, y, protocolW, rowH)
		x += protocolW + gap
		a.move(a.host, x, y, hostW, rowH)
		x += hostW + gap
		a.move(a.port, x, y, 70, rowH)

		y = 126
		buttonW := 112
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
		chooseW, passphraseW := 136, 220
		keyW := available - chooseW - passphraseW - 2*gap
		if keyW < 260 {
			passphraseW = 180
			keyW = available - chooseW - passphraseW - 2*gap
		}
		a.move(a.keyPath, margin, y, keyW, rowH)
		a.move(a.chooseKey, margin+keyW+gap, y, chooseW, rowH)
		a.move(a.passphrase, margin+keyW+gap+chooseW+gap, y, passphraseW, rowH)
		sectionY = 204
	}

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
	upW, chooseW, refreshW := 70, 94, 92
	if compact {
		upW, chooseW, refreshW = 64, 82, 84
	}
	localPathW := panelW - upW - chooseW - refreshW - 3*buttonGap
	if localPathW < 80 {
		localPathW = 80
	}
	a.move(a.localPath, leftX, pathY, localPathW, rowH)
	x = leftX + localPathW + buttonGap
	a.move(a.localUp, x, pathY, upW, rowH)
	x += upW + buttonGap
	a.move(a.localChoose, x, pathY, chooseW, rowH)
	x += chooseW + buttonGap
	a.move(a.localRefresh, x, pathY, refreshW, rowH)

	remotePathW := panelW - upW - refreshW - 2*buttonGap
	if remotePathW < 80 {
		remotePathW = 80
	}
	a.move(a.remotePath, rightX, pathY, remotePathW, rowH)
	x = rightX + remotePathW + buttonGap
	a.move(a.remoteUp, x, pathY, upW, rowH)
	x += upW + buttonGap
	a.move(a.remoteRefresh, x, pathY, refreshW, rowH)

	actionY := pathY + rowH + 7
	mkdirW, renameW, deleteW, chmodW, actionGap := 104, 124, 104, 112, 8
	if compact {
		mkdirW, renameW, deleteW, chmodW, actionGap = 88, 102, 86, 96, 6
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
	a.move(a.sectionTransfers, margin, queueLabelY, 180, 18)
	summaryW := width - margin - (margin + 180)
	if summaryW > 580 {
		summaryW = 580
	}
	a.move(a.transferSummary, margin+180, queueLabelY, summaryW, 18)
	queueButtonsY := queueLabelY + 22
	qx := margin
	queueWidths := []int{118, 118, 108, 108, 164}
	if compact {
		queueWidths = []int{108, 108, 100, 100, 150}
	}
	for index, h := range []uintptr{a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue} {
		a.move(h, qx, queueButtonsY, queueWidths[index], queueButtonsH)
		qx += queueWidths[index] + gap
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
		{"title", a.brandTitle}, {"subtitle", a.brandSubtitle}, {"connection badge", a.connectionBadge}, {"language", a.languageCombo},
		{"profiles", a.profilesCombo}, {"save profile", a.saveProfile}, {"delete profile", a.removeProfile}, {"site manager", a.siteManagerBtn}, {"settings", a.settingsBtn}, {"about", a.aboutBtn},
		{"protocol", a.protocol}, {"server", a.host}, {"port", a.port}, {"username", a.user}, {"password", a.pass},
		{"private key", a.keyPath}, {"choose key", a.chooseKey}, {"key passphrase", a.passphrase}, {"connect", a.connect}, {"disconnect", a.disconnect},
		{"local path", a.localPath}, {"local up", a.localUp}, {"local choose", a.localChoose}, {"local refresh", a.localRefresh},
		{"local list", a.localList}, {"local new folder", a.localMkdir}, {"local rename", a.localRename}, {"local delete", a.localDelete},
		{"remote path", a.remotePath}, {"remote up", a.remoteUp}, {"remote refresh", a.remoteRefresh}, {"remote list", a.remoteList},
		{"remote new folder", a.remoteMkdir}, {"remote rename", a.remoteRename}, {"remote delete", a.remoteDelete}, {"remote permissions", a.remoteChmod},
		{"upload", a.upload}, {"download", a.download}, {"transfer list", a.transferList}, {"pause", a.pauseQueue}, {"resume", a.resumeQueue},
		{"cancel transfer", a.cancelJob}, {"retry transfer", a.retryJob}, {"clear transfers", a.clearQueue},
		{"status", a.status}, {"version", a.statusVersion}, {"transfer summary", a.transferSummary},
	}
	for _, control := range controls {
		if control.h == 0 {
			return fmt.Errorf("unable to initialize control: %s", control.name)
		}
	}
	return nil
}
