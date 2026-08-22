//go:build windows

package desktop

func maxInt(values ...int) int {
	max := 0
	for _, value := range values {
		if value > max {
			max = value
		}
	}
	return max
}

// layoutResponsive starts from the proven legacy geometry, then replaces the
// interaction-heavy rows with measured multilingual geometry. This keeps all
// existing panel/list behavior while guaranteeing that owner-drawn button text
// is never deliberately truncated and that dense rows wrap on compact windows.
func (a *app) layoutResponsive(width, height int) {
	a.layout(width, height)
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

	// Profile/application toolbar: measure the actual localized labels.
	toolbarY := 51
	availableToolbar := width - 2*margin
	toolbarHandles := []uintptr{a.saveProfile, a.removeProfile, a.settingsBtn, a.aboutBtn}
	toolbarFloors := []int{112, 112, 104, 96}
	toolbarWidths := make([]int, len(toolbarHandles))
	for i, hwnd := range toolbarHandles {
		toolbarWidths[i] = a.preferredButtonWidth(hwnd, toolbarFloors[i])
	}
	profileW := availableToolbar - totalWidths(toolbarWidths, gap) - gap
	if profileW < 190 {
		profileW = 190
	}
	if profileW > 360 {
		profileW = 360
	}
	x := margin
	a.move(a.profilesCombo, x, toolbarY, profileW, rowH)
	x += profileW + gap
	for i, hwnd := range toolbarHandles {
		a.move(hwnd, x, toolbarY, toolbarWidths[i], rowH)
		x += toolbarWidths[i] + gap
	}

	available := width - 2*margin
	connectW := a.preferredButtonWidth(a.connect, 112)
	disconnectW := a.preferredButtonWidth(a.disconnect, 112)
	chooseKeyW := a.preferredButtonWidth(a.chooseKey, 136)
	showSFTP := a.protocolValue() == "sftp"
	for _, hwnd := range []uintptr{a.keyPath, a.chooseKey, a.passphrase} {
		if showSFTP {
			showWindow.Call(hwnd, swShow)
		} else {
			showWindow.Call(hwnd, 0)
		}
	}

	sectionY := 168
	if !compact {
		y := 89
		x = margin
		a.move(a.protocol, x, y, 138, rowH)
		x += 138 + gap
		remaining := available - 138 - 70 - 220 - 4*gap - connectW - disconnectW
		hostW := clampInt(remaining*55/100, 220, 330)
		passW := remaining - hostW
		if passW < 140 {
			passW = 140
		}
		a.move(a.host, x, y, hostW, rowH)
		x += hostW + gap
		a.move(a.port, x, y, 70, rowH)
		x += 70 + gap
		a.move(a.user, x, y, 220, rowH)
		x += 220 + gap
		a.move(a.pass, x, y, passW, rowH)
		x += passW + gap
		a.move(a.connect, x, y, connectW, rowH)
		x += connectW + gap
		a.move(a.disconnect, x, y, disconnectW, rowH)
		if showSFTP {
			y = 126
			passphraseW := 240
			keyW := available - chooseKeyW - passphraseW - 2*gap
			if keyW < 300 {
				passphraseW = 190
				keyW = available - chooseKeyW - passphraseW - 2*gap
			}
			a.move(a.keyPath, margin, y, keyW, rowH)
			a.move(a.chooseKey, margin+keyW+gap, y, chooseKeyW, rowH)
			a.move(a.passphrase, margin+keyW+gap+chooseKeyW+gap, y, passphraseW, rowH)
			sectionY = 168
		} else {
			sectionY = 130
		}
	} else {
		y := 89
		protocolW := 138
		hostW := clampInt(available-protocolW-70-2*gap, 280, 520)
		x = margin
		a.move(a.protocol, x, y, protocolW, rowH)
		x += protocolW + gap
		a.move(a.host, x, y, hostW, rowH)
		x += hostW + gap
		a.move(a.port, x, y, 70, rowH)

		y = 126
		fieldSpace := available - connectW - disconnectW - 3*gap
		userW := fieldSpace * 54 / 100
		passW := fieldSpace - userW
		if userW < 180 {
			userW = 180
		}
		if passW < 150 {
			passW = 150
		}
		x = margin
		a.move(a.user, x, y, userW, rowH)
		x += userW + gap
		a.move(a.pass, x, y, passW, rowH)
		x += passW + gap
		a.move(a.connect, x, y, connectW, rowH)
		x += connectW + gap
		a.move(a.disconnect, x, y, disconnectW, rowH)
		if showSFTP {
			y = 163
			passphraseW := 190
			keyW := available - chooseKeyW - passphraseW - 2*gap
			if keyW < 250 {
				passphraseW = 160
				keyW = available - chooseKeyW - passphraseW - 2*gap
			}
			a.move(a.keyPath, margin, y, keyW, rowH)
			a.move(a.chooseKey, margin+keyW+gap, y, chooseKeyW, rowH)
			a.move(a.passphrase, margin+keyW+gap+chooseKeyW+gap, y, passphraseW, rowH)
			sectionY = 204
		} else {
			sectionY = 167
		}
	}

	centerW := maxInt(96, a.preferredButtonWidth(a.upload, 96), a.preferredButtonWidth(a.download, 96))
	if centerW > 170 {
		centerW = 170
	}
	panelGap := 12
	if compact {
		panelGap = 10
	}
	usableW := width - 2*margin - centerW - 2*panelGap
	panelW := usableW / 2
	leftX := margin
	centerX := leftX + panelW + panelGap
	rightX := centerX + centerW + panelGap
	a.move(a.sectionLocal, leftX, sectionY, panelW, 20)
	a.move(a.sectionRemote, rightX, sectionY, panelW, 20)

	// Full-width path fields keep navigation labels from competing with paths.
	pathY := sectionY + 24
	a.move(a.localPath, leftX, pathY, panelW, rowH)
	a.move(a.remotePath, rightX, pathY, panelW, rowH)
	navY := pathY + rowH + 6
	localNav := []uintptr{a.localUp, a.localChoose, a.localRefresh}
	localNavW := []int{
		a.preferredButtonWidth(a.localUp, 70),
		a.preferredButtonWidth(a.localChoose, 94),
		a.preferredButtonWidth(a.localRefresh, 92),
	}
	remoteNav := []uintptr{a.remoteUp, a.remoteRefresh}
	remoteNavW := []int{
		a.preferredButtonWidth(a.remoteUp, 70),
		a.preferredButtonWidth(a.remoteRefresh, 92),
	}
	localNavRows := a.placeButtonFlow(localNav, localNavW, leftX, navY, panelW, 6, rowH)
	remoteNavRows := a.placeButtonFlow(remoteNav, remoteNavW, rightX, navY, panelW, 6, rowH)
	navRows := maxInt(localNavRows, remoteNavRows)

	actionY := navY + navRows*(rowH+6)
	localActions := []uintptr{a.localMkdir, a.localRename, a.localDelete}
	localActionW := []int{
		a.preferredButtonWidth(a.localMkdir, 104),
		a.preferredButtonWidth(a.localRename, 116),
		a.preferredButtonWidth(a.localDelete, 100),
	}
	remoteActions := []uintptr{a.remoteMkdir, a.remoteRename, a.remoteDelete, a.remoteChmod}
	remoteActionW := []int{
		a.preferredButtonWidth(a.remoteMkdir, 104),
		a.preferredButtonWidth(a.remoteRename, 116),
		a.preferredButtonWidth(a.remoteDelete, 100),
		a.preferredButtonWidth(a.remoteChmod, 112),
	}
	localActionRows := a.placeButtonFlow(localActions, localActionW, leftX, actionY, panelW, 6, rowH)
	remoteActionRows := a.placeButtonFlow(remoteActions, remoteActionW, rightX, actionY, panelW, 6, rowH)
	actionRows := maxInt(localActionRows, remoteActionRows)

	statusH := 24
	queueButtonsH := 33
	listY := actionY + actionRows*(rowH+6) + 3
	queueHandles := []uintptr{a.pauseQueue, a.resumeQueue, a.cancelJob, a.retryJob, a.clearQueue}
	queueWidths := []int{
		a.preferredButtonWidth(a.pauseQueue, 110),
		a.preferredButtonWidth(a.resumeQueue, 110),
		a.preferredButtonWidth(a.cancelJob, 104),
		a.preferredButtonWidth(a.retryJob, 104),
		a.preferredButtonWidth(a.clearQueue, 150),
	}
	queueRows := buttonFlowRows(queueWidths, gap, available)
	if queueRows < 1 {
		queueRows = 1
	}
	contentBottom := height - statusH - 12
	availableH := contentBottom - listY
	fixedQueueH := 10 + 18 + 4 + queueRows*queueButtonsH + (queueRows-1)*gap + 7
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
	usedRows := a.placeButtonFlow(queueHandles, queueWidths, margin, queueButtonsY, available, gap, queueButtonsH)
	if usedRows < 1 {
		usedRows = 1
	}
	queueY := queueButtonsY + usedRows*queueButtonsH + (usedRows-1)*gap + 7
	a.move(a.transferList, margin, queueY, width-2*margin, queueH)
	a.move(a.status, margin, height-statusH-5, width-2*margin-250, statusH)
	a.move(a.statusVersion, width-margin-238, height-statusH-5, 238, statusH)
	a.resizeListColumns()
	invalidateRect.Call(a.hwnd, 0, 1)
}
