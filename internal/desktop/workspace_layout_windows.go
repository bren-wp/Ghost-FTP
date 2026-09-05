//go:build windows

package desktop

import "unsafe"

const (
	workspaceLVMGetHeader           = 0x101F
	workspaceLVMSetColumnOrderArray = lvmFirst + 58
	workspaceSWHide                 = 0
)

func styleWorkspaceList(list uintptr) {
	if list == 0 {
		return
	}
	setWindowTheme.Call(list, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
	sendMessageW.Call(list, lvmSetBkColor, 0, listColor())
	sendMessageW.Call(list, lvmSetTextBkColor, 0, listColor())
	sendMessageW.Call(list, lvmSetTextColor, 0, textColor())
	header, _, _ := sendMessageW.Call(list, workspaceLVMGetHeader, 0, 0)
	if header != 0 {
		setWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_ItemsView"))), 0)
	}
}

func applyReferenceFileColumnOrder(list uintptr, remote bool) {
	if list == 0 {
		return
	}
	if remote {
		// Logical storage: Name, Type, Size, Modified, Permissions.
		// Reference order: Name, Size, Type, Modified, Permissions.
		order := [5]int32{0, 2, 1, 3, 4}
		sendMessageW.Call(list, workspaceLVMSetColumnOrderArray, uintptr(len(order)), uintptr(unsafe.Pointer(&order[0])))
		return
	}
	order := [4]int32{0, 2, 1, 3}
	sendMessageW.Call(list, workspaceLVMSetColumnOrderArray, uintptr(len(order)), uintptr(unsafe.Pointer(&order[0])))
}

func (a *app) resizeReferenceWorkspaceColumns(panelW, mainW int) {
	if panelW <= 0 || mainW <= 0 {
		return
	}
	sizeW, typeW, modifiedW := 84, 92, 130
	localNameW := panelW - sizeW - typeW - modifiedW - 24
	if localNameW < 116 {
		localNameW = 116
	}
	applyReferenceFileColumnOrder(a.localList, false)
	for index, columnW := range []int{localNameW, typeW, sizeW, modifiedW} {
		if a.localList != 0 {
			sendMessageW.Call(a.localList, lvmSetColumnWidth, uintptr(index), uintptr(a.scale(columnW)))
		}
	}

	permissionsW := 108
	remoteNameW := panelW - sizeW - typeW - modifiedW - permissionsW - 24
	if remoteNameW < 94 {
		remoteNameW = 94
	}
	applyReferenceFileColumnOrder(a.remoteList, true)
	for index, columnW := range []int{remoteNameW, typeW, sizeW, modifiedW, permissionsW} {
		if a.remoteList != 0 {
			sendMessageW.Call(a.remoteList, lvmSetColumnWidth, uintptr(index), uintptr(a.scale(columnW)))
		}
	}

	available := mainW - 22
	directionW, statusW, progressW := 96, 150, 78
	pathSpace := available - directionW - statusW - progressW - 12
	if pathSpace < 300 {
		pathSpace = 300
	}
	localW := pathSpace / 2
	remoteW := pathSpace - localW
	for index, columnW := range []int{directionW, localW, remoteW, statusW, progressW} {
		if a.transferList != 0 {
			sendMessageW.Call(a.transferList, lvmSetColumnWidth, uintptr(index), uintptr(a.scale(columnW)))
		}
	}
}

func showControls(show bool, controls ...uintptr) {
	state := workspaceSWHide
	if show {
		state = swShow
	}
	for _, control := range controls {
		if control != 0 {
			showWindow.Call(control, uintptr(state))
		}
	}
}

// refineWorkspaceLayout is the canonical presentation pass for the Windows
// executable. Setup and Portable package the same binary, so they necessarily
// expose the same shell. The geometry mirrors the supplied Ghost FTP reference:
// persistent left navigation, top action toolbar, connection log + Quick
// Connect cards, equal local/remote file panes and a full-width transfer queue.
// Security and transfer semantics remain in the existing Engine/action layer.
func (a *app) refineWorkspaceLayout() {
	if a == nil || a.hwnd == 0 || a.localList == 0 || a.remoteList == 0 || a.transferList == 0 {
		return
	}
	a.ensureReferenceShellControls()
	a.updateReferenceShellLanguage()

	var client rect
	if result, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); result == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	if width < 900 {
		width = 900
	}
	if height < 620 {
		height = 620
	}

	compact := width < 1280
	veryCompact := width < 1080
	sidebarW := 244
	if compact {
		sidebarW = 210
	}
	if veryCompact {
		sidebarW = 184
	}
	gap := 10
	outer := 10
	rowH := 29
	mainX := sidebarW + 1
	mainW := width - mainX
	contentX := mainX + outer
	contentW := mainW - 2*outer

	// Left navigation / local-only profile surface.
	a.move(a.shellSidebar, 0, 0, sidebarW, height)
	a.move(a.brandTitle, 18, 14, sidebarW-36, 31)
	a.move(a.brandSubtitle, 18, 49, sidebarW-36, 43)
	a.move(a.sidebarServersLabel, 18, 112, sidebarW-72, 20)
	a.move(a.siteManagerBtn, sidebarW-49, 103, 34, 30)
	a.setButtonLabel(a.siteManagerBtn, "+")
	a.move(a.profilesCombo, 14, 142, sidebarW-28, rowH)
	a.move(a.saveProfile, 14, 179, (sidebarW-34)/2, rowH)
	a.move(a.removeProfile, 20+(sidebarW-34)/2, 179, (sidebarW-34)/2, rowH)

	privacyH := 96
	footerY := height - 46
	privacyY := footerY - privacyH - 12
	if privacyY < 225 {
		privacyY = 225
	}
	a.move(a.sidebarPrivacyTitle, 18, privacyY+12, sidebarW-36, 20)
	a.move(a.sidebarPrivacyBody, 18, privacyY+35, sidebarW-36, privacyH-42)
	a.move(a.settingsBtn, 14, footerY, (sidebarW-34)/2, 31)
	a.move(a.aboutBtn, 20+(sidebarW-34)/2, footerY, (sidebarW-34)/2, 31)
	a.move(a.connectionBadge, 14, height-24, sidebarW-28, 18)

	// Top action toolbar. On compact runner desktops the least-used mutation
	// actions collapse while the same commands remain available in the panes and
	// menu, preventing overlap without changing behavior.
	toolbarH := 70
	a.move(a.shellToolbar, mainX, 0, mainW, toolbarH)
	toolbarY := 5
	toolbarX := contentX
	languageW := 154
	if compact {
		languageW = 132
	}
	languageX := width - outer - languageW
	a.move(a.languageCombo, languageX, 8, languageW, 28)

	buttons := []struct {
		h uintptr
		w int
	}{
		{a.toolbarConnect, 78}, {a.toolbarDisconnect, 82}, {a.toolbarUpload, 78}, {a.toolbarDownload, 82},
		{a.toolbarRefresh, 78}, {a.toolbarNewFolder, 88}, {a.toolbarRename, 82}, {a.toolbarDelete, 76},
		{a.toolbarSites, 92}, {a.toolbarSettings, 82}, {a.toolbarDiagnostics, 90},
	}
	availableToolbar := languageX - gap - toolbarX
	needed := 0
	for _, button := range buttons {
		needed += button.w + 4
	}
	showMutations := needed <= availableToolbar
	for index, button := range buttons {
		visible := true
		if !showMutations && index >= 5 && index <= 7 {
			visible = false
		}
		if !visible {
			showControls(false, button.h)
			continue
		}
		showControls(true, button.h)
		buttonW := button.w
		if compact {
			buttonW = clampInt(buttonW-8, 62, buttonW)
		}
		a.move(button.h, toolbarX, toolbarY, buttonW, 60)
		toolbarX += buttonW + 4
		if toolbarX+62 > languageX {
			showControls(false, button.h)
			toolbarX -= buttonW + 4
		}
	}

	// Upper cards: connection log on the left and Quick Connect on the right.
	topY := toolbarH + gap
	topH := 160
	if height < 760 {
		topH = 142
	}
	logW := contentW * 48 / 100
	quickX := contentX + logW + gap
	quickW := contentW - logW - gap
	a.move(a.shellLogCard, contentX, topY, logW, topH)
	a.move(a.shellQuickCard, quickX, topY, quickW, topH)
	a.move(a.logTitle, contentX+12, topY+10, logW-24, 20)
	a.move(a.status, contentX+12, topY+36, logW-24, topH-48)
	a.move(a.quickTitle, quickX+12, topY+10, quickW-24, 20)

	quickInnerX := quickX + 12
	quickInnerW := quickW - 24
	fieldY := topY + 38
	fieldGap := 7
	quickBottom := fieldY + rowH
	if width >= 1540 {
		protocolW, portW, connectW := 112, 64, 104
		remaining := quickInnerW - protocolW - portW - connectW - 5*fieldGap
		hostW := remaining * 30 / 100
		userW := remaining * 35 / 100
		passW := remaining - hostW - userW
		if hostW < 100 {
			hostW = 100
		}
		if userW < 92 {
			userW = 92
		}
		if passW < 92 {
			passW = 92
		}
		x := quickInnerX
		a.move(a.host, x, fieldY, hostW, rowH)
		x += hostW + fieldGap
		a.move(a.port, x, fieldY, portW, rowH)
		x += portW + fieldGap
		a.move(a.protocol, x, fieldY, protocolW, rowH)
		x += protocolW + fieldGap
		a.move(a.user, x, fieldY, userW, rowH)
		x += userW + fieldGap
		a.move(a.pass, x, fieldY, passW, rowH)
		x += passW + fieldGap
		a.move(a.connect, x, fieldY, connectW, rowH)
	} else {
		protocolW, portW := 94, 62
		hostW := quickInnerW - protocolW - portW - 2*fieldGap
		if hostW < 110 {
			hostW = 110
		}
		x := quickInnerX
		a.move(a.host, x, fieldY, hostW, rowH)
		x += hostW + fieldGap
		a.move(a.port, x, fieldY, portW, rowH)
		x += portW + fieldGap
		a.move(a.protocol, x, fieldY, protocolW, rowH)

		identityY := fieldY + rowH + 8
		connectW := 92
		identityFieldsW := quickInnerW - connectW - 2*fieldGap
		userW := identityFieldsW / 2
		passW := identityFieldsW - userW
		a.move(a.user, quickInnerX, identityY, userW, rowH)
		a.move(a.pass, quickInnerX+userW+fieldGap, identityY, passW, rowH)
		a.move(a.connect, quickInnerX+userW+fieldGap+passW+fieldGap, identityY, connectW, rowH)
		quickBottom = identityY + rowH
	}
	showControls(true, a.connect)
	showControls(false, a.disconnect)

	// SFTP-specific secret controls occupy the next card row only for SFTP;
	// FTP/FTPS keeps the card visually quiet, as in the reference.
	sftp := a.protocolValue() == "sftp"
	if sftp {
		showControls(true, a.keyPath, a.chooseKey, a.passphrase)
		keyY := quickBottom + 8
		chooseW, passphraseW := 126, 170
		keyW := quickInnerW - chooseW - passphraseW - 2*fieldGap
		if keyW < 130 {
			passphraseW = 135
			keyW = quickInnerW - chooseW - passphraseW - 2*fieldGap
		}
		a.move(a.keyPath, quickInnerX, keyY, keyW, rowH)
		a.move(a.chooseKey, quickInnerX+keyW+fieldGap, keyY, chooseW, rowH)
		a.move(a.passphrase, quickInnerX+keyW+fieldGap+chooseW+fieldGap, keyY, passphraseW, rowH)
	} else {
		showControls(false, a.keyPath, a.chooseKey, a.passphrase)
	}

	// File cards consume the main visual area. Pane mutation actions are provided
	// by the top toolbar; path/navigation controls stay adjacent to each pane.
	filesY := topY + topH + gap
	footerH := 23
	queueH := clampInt(height/5, 148, 190)
	queueY := height - footerH - outer - queueH
	filesH := queueY - gap - filesY
	if filesH < 220 {
		queueH = clampInt(queueH-(220-filesH), 118, queueH)
		queueY = height - footerH - outer - queueH
		filesH = queueY - gap - filesY
	}
	panelGap := 10
	panelW := (contentW - panelGap) / 2
	leftX := contentX
	rightX := leftX + panelW + panelGap
	a.move(a.shellLocalCard, leftX, filesY, panelW, filesH)
	a.move(a.shellRemoteCard, rightX, filesY, panelW, filesH)

	headerY := filesY + 10
	a.move(a.sectionLocal, leftX+10, headerY, 120, 20)
	a.move(a.localDeviceLabel, leftX+126, headerY, panelW-136, 20)
	a.move(a.sectionRemote, rightX+10, headerY, 170, 20)
	a.move(a.remoteStateLabel, rightX+panelW-135, headerY, 125, 20)

	pathY := headerY + 27
	navW, navGap := 35, 5
	localPathX := leftX + 10 + navW + navGap + navW + navGap
	localPathW := panelW - 20 - 2*(navW+navGap) - navW - navGap
	a.move(a.localUp, leftX+10, pathY, navW, rowH)
	a.move(a.localRefresh, leftX+10+navW+navGap, pathY, navW, rowH)
	a.move(a.localPath, localPathX, pathY, localPathW, rowH)
	a.move(a.localChoose, leftX+panelW-10-navW, pathY, navW, rowH)

	remotePathX := rightX + 10 + navW + navGap + navW + navGap
	searchW := clampInt(panelW/4, 100, 190)
	chmodW := 38
	remotePathW := panelW - 20 - 2*(navW+navGap) - searchW - navGap - chmodW - navGap
	if remotePathW < 72 {
		remotePathW = 72
	}
	a.move(a.remoteUp, rightX+10, pathY, navW, rowH)
	a.move(a.remoteRefresh, rightX+10+navW+navGap, pathY, navW, rowH)
	a.move(a.remotePath, remotePathX, pathY, remotePathW, rowH)
	a.move(a.remoteSearch, remotePathX+remotePathW+navGap, pathY, searchW, rowH)
	a.move(a.remoteChmod, rightX+panelW-10-chmodW, pathY, chmodW, rowH)
	showControls(true, a.remoteSearch, a.remoteChmod)

	listY := pathY + rowH + 7
	listH := filesY + filesH - 10 - listY
	if listH < 100 {
		listH = 100
	}
	a.move(a.localList, leftX+1, listY, panelW-2, listH)
	a.move(a.remoteList, rightX+1, listY, panelW-2, listH)
	showControls(false, a.localMkdir, a.localRename, a.localDelete, a.remoteMkdir, a.remoteRename, a.remoteDelete, a.upload, a.download)

	// Bottom queue card. Existing queue controls remain fully functional; the
	// header reads like the reference tabs/summary without introducing a second
	// transfer state model.
	a.move(a.shellQueueCard, contentX, queueY, contentW, queueH)
	queueHeaderY := queueY + 8
	a.move(a.sectionTransfers, contentX+10, queueHeaderY+5, 105, 18)
	a.move(a.transferSummary, contentX+112, queueHeaderY+5, clampInt(contentW/2, 220, 560), 18)
	queueWidths := []int{82, 86, 82, 78, 132}
	queueGap := 5
	queueButtonsW := 0
	for _, w := range queueWidths {
		queueButtonsW += w
	}
	queueButtonsW += (len(queueWidths) - 1) * queueGap
	qx := contentX + contentW - queueButtonsW - 10
	for index, control := range []uintptr{a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue} {
		a.move(control, qx, queueHeaderY, queueWidths[index], 29)
		qx += queueWidths[index] + queueGap
	}
	queueListY := queueHeaderY + 36
	a.move(a.transferList, contentX+1, queueListY, contentW-2, queueY+queueH-queueListY-1)

	// Footer mirrors the quiet disconnected/version row in the reference.
	a.move(a.statusVersion, width-outer-250, height-footerH, 250, footerH)

	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		styleWorkspaceList(list)
	}
	// A disconnected remote list remains readable but every mutating/transfer
	// action still fails closed through canonical action state.
	setControlEnabled(a.remoteList, true)
	a.resizeReferenceWorkspaceColumns(panelW, mainW)
	invalidateRect.Call(a.hwnd, 0, 1)
}
