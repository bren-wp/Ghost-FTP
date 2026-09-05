//go:build windows

package desktop

import "unsafe"

const (
	workspaceLVMGetHeader = 0x101F
	workspaceSWHide       = 0
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

func (a *app) resizeReferenceWorkspaceColumns(panelW, windowW int) {
	if panelW <= 0 || windowW <= 0 {
		return
	}
	typeW, sizeW, modifiedW := 76, 82, 126
	nameW := panelW - typeW - sizeW - modifiedW - 8
	if nameW < 118 {
		nameW = 118
	}
	for _, list := range []uintptr{a.localList, a.remoteList} {
		for index, width := range []int{nameW, typeW, sizeW, modifiedW} {
			if list != 0 {
				sendMessageW.Call(list, lvmSetColumnWidth, uintptr(index), uintptr(a.scale(width)))
			}
		}
	}

	available := windowW - 28
	directionW, statusW, progressW := 100, 168, 82
	pathSpace := available - directionW - statusW - progressW - 8
	if pathSpace < 320 {
		pathSpace = 320
	}
	localW := pathSpace / 2
	remoteW := pathSpace - localW
	for index, width := range []int{directionW, localW, remoteW, statusW, progressW} {
		if a.transferList != 0 {
			sendMessageW.Call(a.transferList, lvmSetColumnWidth, uintptr(index), uintptr(a.scale(width)))
		}
	}
}

// refineWorkspaceLayout keeps the information architecture intentionally close
// to a professional dual-pane FTP client: connection controls at the top, a
// persistent session/status strip, LOCAL and SERVER panes with equal visual
// weight, transfer controls between them, and one full-width queue at the
// bottom. It retains Ghost FTP's own native graphite/navy design and shared
// control implementation rather than cloning another client's visuals.
func (a *app) refineWorkspaceLayout() {
	if a == nil || a.hwnd == 0 || a.localList == 0 || a.remoteList == 0 || a.transferList == 0 {
		return
	}
	var client rect
	if result, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); result == 0 {
		return
	}
	width := a.unscale(int(client.Right - client.Left))
	height := a.unscale(int(client.Bottom - client.Top))
	if width < 900 {
		width = 900
	}
	if height < 610 {
		height = 610
	}

	margin, gap, rowH := 14, 7, 28
	compact := width < 1320

	// Compact branded header. The native menu above this row owns navigation;
	// the header therefore spends its space on identity, locale and connection
	// state rather than duplicating menu actions.
	headerY := 8
	badgeW, languageW := 140, 160
	if compact {
		badgeW, languageW = 132, 144
	}
	a.move(a.brandTitle, margin, headerY, 140, 34)
	subtitleX := margin + 150
	languageX := width - margin - badgeW - gap - languageW
	subtitleW := languageX - gap - subtitleX
	if subtitleW < 150 {
		subtitleW = 150
	}
	a.move(a.brandSubtitle, subtitleX, headerY+7, subtitleW, 20)
	a.move(a.languageCombo, languageX, headerY+1, languageW, rowH)
	a.move(a.connectionBadge, width-margin-badgeW, headerY+7, badgeW, 20)

	// Saved-site/utility strip. Wider profile selection is more useful than large
	// decorative buttons, especially on the 1024px-class laptop layout.
	toolbarY := 45
	buttonGap := 6
	toolbarButtons := []struct {
		h uintptr
		w int
	}{
		{a.saveProfile, 112},
		{a.removeProfile, 112},
		{a.settingsBtn, 102},
		{a.aboutBtn, 90},
	}
	fixedToolbar := 0
	for _, button := range toolbarButtons {
		fixedToolbar += button.w
	}
	fixedToolbar += len(toolbarButtons) * buttonGap
	profileW := width - 2*margin - fixedToolbar
	if profileW < 250 {
		profileW = 250
	}
	x := margin
	a.move(a.profilesCombo, x, toolbarY, profileW, rowH)
	x += profileW + buttonGap
	for _, button := range toolbarButtons {
		a.move(button.h, x, toolbarY, button.w, rowH)
		x += button.w + buttonGap
	}

	connectionY := 79
	contentW := width - 2*margin
	connectionBottom := connectionY + rowH
	if compact {
		// Row 1: protocol / server / port. Row 2: identity / secret / session
		// actions. This eliminates clipped fields at ~940–1100px widths.
		protocolW, portW := 112, 70
		hostW := contentW - protocolW - portW - 2*gap
		if hostW < 260 {
			hostW = 260
		}
		x = margin
		a.move(a.protocol, x, connectionY, protocolW, rowH)
		x += protocolW + gap
		a.move(a.host, x, connectionY, hostW, rowH)
		x += hostW + gap
		a.move(a.port, x, connectionY, portW, rowH)

		identityY := connectionY + rowH + 6
		connectW, disconnectW := 106, 106
		fieldsW := contentW - connectW - disconnectW - 3*gap
		userW := fieldsW * 48 / 100
		passW := fieldsW - userW
		x = margin
		a.move(a.user, x, identityY, userW, rowH)
		x += userW + gap
		a.move(a.pass, x, identityY, passW, rowH)
		x += passW + gap
		a.move(a.connect, x, identityY, connectW, rowH)
		x += connectW + gap
		a.move(a.disconnect, x, identityY, disconnectW, rowH)
		connectionBottom = identityY + rowH
	} else {
		protocolW, portW, userW, passW, connectW, disconnectW := 118, 70, 205, 190, 112, 112
		fixed := protocolW + portW + userW + passW + connectW + disconnectW + 6*gap
		hostW := contentW - fixed
		if hostW < 250 {
			hostW = 250
		}
		x = margin
		for _, item := range []struct {
			h uintptr
			w int
		}{
			{a.protocol, protocolW}, {a.host, hostW}, {a.port, portW}, {a.user, userW},
			{a.pass, passW}, {a.connect, connectW}, {a.disconnect, disconnectW},
		} {
			a.move(item.h, x, connectionY, item.w, rowH)
			x += item.w + gap
		}
	}

	// SFTP-only controls no longer reserve an empty disabled row for FTP/FTPS.
	sftp := a.protocolValue() == "sftp"
	for _, control := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
		if sftp {
			showWindow.Call(control, swShow)
		} else {
			showWindow.Call(control, workspaceSWHide)
		}
	}
	if sftp {
		keyY := connectionBottom + 6
		chooseW, passphraseW := 138, 238
		keyW := contentW - chooseW - passphraseW - 2*gap
		if keyW < 260 {
			passphraseW = 190
			keyW = contentW - chooseW - passphraseW - 2*gap
		}
		a.move(a.keyPath, margin, keyY, keyW, rowH)
		a.move(a.chooseKey, margin+keyW+gap, keyY, chooseW, rowH)
		a.move(a.passphrase, margin+keyW+gap+chooseW+gap, keyY, passphraseW, rowH)
		connectionBottom = keyY + rowH
	}

	// The current operation/session message is promoted from a tiny footer into a
	// visible activity strip, echoing the useful status/log band of classic FTP
	// clients without adding a noisy permanent multi-line console.
	statusY := connectionBottom + 6
	a.move(a.status, margin, statusY, contentW, 21)

	sectionY := statusY + 27
	centerW, panelGap := 82, 8
	if width >= 1320 {
		centerW, panelGap = 96, 10
	}
	panelW := (contentW - centerW - 2*panelGap) / 2
	leftX := margin
	centerX := leftX + panelW + panelGap
	rightX := centerX + centerW + panelGap
	a.move(a.sectionLocal, leftX, sectionY, panelW, 18)
	a.move(a.sectionRemote, rightX, sectionY, panelW, 18)

	pathY := sectionY + 22
	upW, folderW, refreshW := 62, 100, 100
	pathGap := 5
	localPathW := panelW - upW - folderW - refreshW - 3*pathGap
	if localPathW < 120 {
		localPathW = 120
	}
	a.move(a.localPath, leftX, pathY, localPathW, rowH)
	x = leftX + localPathW + pathGap
	a.move(a.localUp, x, pathY, upW, rowH)
	x += upW + pathGap
	a.move(a.localChoose, x, pathY, folderW, rowH)
	x += folderW + pathGap
	a.move(a.localRefresh, x, pathY, refreshW, rowH)

	remotePathW := panelW - upW - refreshW - 2*pathGap
	if remotePathW < 120 {
		remotePathW = 120
	}
	a.move(a.remotePath, rightX, pathY, remotePathW, rowH)
	x = rightX + remotePathW + pathGap
	a.move(a.remoteUp, x, pathY, upW, rowH)
	x += upW + pathGap
	a.move(a.remoteRefresh, x, pathY, refreshW, rowH)

	// Equal-width pane actions prevent clipped labels such as "New…"/"Rena…"
	// at the runner's authentic 976px window capture.
	actionY := pathY + rowH + 5
	actionGap := 6
	localActionW := (panelW - 2*actionGap) / 3
	a.move(a.localMkdir, leftX, actionY, localActionW, rowH)
	a.move(a.localRename, leftX+localActionW+actionGap, actionY, localActionW, rowH)
	a.move(a.localDelete, leftX+2*(localActionW+actionGap), actionY, panelW-2*(localActionW+actionGap), rowH)

	remoteActionW := (panelW - 3*actionGap) / 4
	for index, control := range []uintptr{a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod} {
		a.move(control, rightX+index*(remoteActionW+actionGap), actionY, remoteActionW, rowH)
	}

	listY := actionY + rowH + 6
	bottomStatusH := 21
	queueListH := clampInt(height/7, 92, 128)
	queueToolsH := 30
	queueY := height - bottomStatusH - 8 - queueListH
	queueToolsY := queueY - queueToolsH - 5
	fileBottom := queueToolsY - 8
	listH := fileBottom - listY
	if listH < 150 {
		shortage := 150 - listH
		queueListH -= shortage
		if queueListH < 72 {
			queueListH = 72
		}
		queueY = height - bottomStatusH - 8 - queueListH
		queueToolsY = queueY - queueToolsH - 5
		fileBottom = queueToolsY - 8
		listH = fileBottom - listY
	}
	if listH < 110 {
		listH = 110
	}
	a.move(a.localList, leftX, listY, panelW, listH)
	a.move(a.remoteList, rightX, listY, panelW, listH)

	transferButtonH := 38
	uploadY := listY + clampInt(listH/2-44, 32, listH-84)
	a.move(a.upload, centerX, uploadY, centerW, transferButtonH)
	a.move(a.download, centerX, uploadY+transferButtonH+8, centerW, transferButtonH)

	// Queue label, summary and queue actions share one toolbar row. The full-width
	// job list remains directly below, leaving substantially more vertical room
	// for the two file browsers on smaller screens.
	a.move(a.sectionTransfers, margin, queueToolsY+6, 108, 18)
	queueWidths := []int{84, 84, 78, 78, 124}
	queueGap := 5
	queueButtonsW := 0
	for _, buttonW := range queueWidths {
		queueButtonsW += buttonW
	}
	queueButtonsW += (len(queueWidths) - 1) * queueGap
	queueButtonsX := width - margin - queueButtonsW
	summaryX := margin + 110
	summaryW := queueButtonsX - queueGap - summaryX
	if summaryW < 150 {
		summaryW = 150
	}
	a.move(a.transferSummary, summaryX, queueToolsY+6, summaryW, 18)
	x = queueButtonsX
	for index, control := range []uintptr{a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue} {
		a.move(control, x, queueToolsY, queueWidths[index], queueToolsH)
		x += queueWidths[index] + queueGap
	}
	a.move(a.transferList, margin, queueY, contentW, queueListH)

	// Version/identity remains a quiet footer; live status now lives above the
	// file panes where it can actually be read during connection work.
	a.move(a.statusVersion, width-margin-250, height-bottomStatusH-3, 250, bottomStatusH)

	for _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
		styleWorkspaceList(list)
	}
	// Keep the empty SERVER surface visually consistent while disconnected.
	// All actual remote operations still fail closed through connected/busy
	// checks and disabled action buttons.
	setControlEnabled(a.remoteList, true)
	a.resizeReferenceWorkspaceColumns(panelW, width)
	invalidateRect.Call(a.hwnd, 0, 1)
}
