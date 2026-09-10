//go:build windows

package desktop

import "unsafe"

const (
	idMoveQueueUp     = 508
	idMoveQueueDown   = 509
	idMoveQueueTop    = 510
	idMoveQueueBottom = 511
)

var (
	queueGetDlgItem     = user32.NewProc("GetDlgItem")
	queueGetWindowRect  = user32.NewProc("GetWindowRect")
	queueScreenToClient = user32.NewProc("ScreenToClient")
)

func (a *app) queuePriorityButton(id int) uintptr {
	if a == nil || a.hwnd == 0 {
		return 0
	}
	hwnd, _, _ := queueGetDlgItem.Call(a.hwnd, uintptr(id))
	return hwnd
}

func (a *app) ensureQueuePriorityControls() {
	if a == nil || a.hwnd == 0 {
		return
	}
	ids := []int{idMoveQueueTop, idMoveQueueUp, idMoveQueueDown, idMoveQueueBottom}
	allPresent := true
	for _, id := range ids {
		if a.queuePriorityButton(id) == 0 {
			allPresent = false
			break
		}
	}
	if allPresent {
		return
	}

	hinst, _, _ := getModuleHandleW.Call(0)
	words := queuePriorityWords(a.languageCode())
	create := func(id int, label, icon string) uintptr {
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("BUTTON"))),
			uintptr(unsafe.Pointer(wstr(label))),
			wsChild|wsVisible|wsTabStop|bsOwnerDraw,
			0, 0, 100, 30,
			a.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd == 0 {
			return 0
		}
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
		a.registerButton(hwnd, icon, label, buttonDefault)
		return hwnd
	}

	if a.queuePriorityButton(idMoveQueueTop) == 0 {
		create(idMoveQueueTop, words.MoveTop, iconUp)
	}
	if a.queuePriorityButton(idMoveQueueUp) == 0 {
		create(idMoveQueueUp, words.MoveUp, iconUp)
	}
	if a.queuePriorityButton(idMoveQueueDown) == 0 {
		create(idMoveQueueDown, words.MoveDown, iconDownload)
	}
	if a.queuePriorityButton(idMoveQueueBottom) == 0 {
		create(idMoveQueueBottom, words.MoveBottom, iconDownload)
	}
}

func (a *app) updateQueuePriorityControls(state queuePriorityState) {
	a.ensureQueuePriorityControls()
	top := a.queuePriorityButton(idMoveQueueTop)
	up := a.queuePriorityButton(idMoveQueueUp)
	down := a.queuePriorityButton(idMoveQueueDown)
	bottom := a.queuePriorityButton(idMoveQueueBottom)
	words := queuePriorityWords(a.languageCode())
	a.setButtonLabel(top, words.MoveTop)
	a.setButtonLabel(up, words.MoveUp)
	a.setButtonLabel(down, words.MoveDown)
	a.setButtonLabel(bottom, words.MoveBottom)
	setControlEnabled(top, state.MoveTop && !a.connectionBusy)
	setControlEnabled(up, state.MoveUp && !a.connectionBusy)
	setControlEnabled(down, state.MoveDown && !a.connectionBusy)
	setControlEnabled(bottom, state.MoveBottom && !a.connectionBusy)
}

func (a *app) layoutQueuePriorityControls() {
	if a == nil || a.hwnd == 0 || a.clearQueue == 0 {
		return
	}
	a.ensureQueuePriorityControls()
	controls := []uintptr{
		a.queuePriorityButton(idMoveQueueTop),
		a.queuePriorityButton(idMoveQueueUp),
		a.queuePriorityButton(idMoveQueueDown),
		a.queuePriorityButton(idMoveQueueBottom),
	}
	for _, control := range controls {
		if control == 0 {
			return
		}
	}

	var clearRect rect
	if ok, _, _ := queueGetWindowRect.Call(a.clearQueue, uintptr(unsafe.Pointer(&clearRect))); ok == 0 {
		return
	}
	topLeft := point{X: clearRect.Left, Y: clearRect.Top}
	bottomRight := point{X: clearRect.Right, Y: clearRect.Bottom}
	if ok, _, _ := queueScreenToClient.Call(a.hwnd, uintptr(unsafe.Pointer(&topLeft))); ok == 0 {
		return
	}
	if ok, _, _ := queueScreenToClient.Call(a.hwnd, uintptr(unsafe.Pointer(&bottomRight))); ok == 0 {
		return
	}

	gap := a.scale(6)
	x := int(bottomRight.X) + gap
	y := int(topLeft.Y)
	height := int(bottomRight.Y - topLeft.Y)
	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	available := int(client.Right) - a.scale(14) - x
	buttonWidth := (available - 3*gap) / 4
	if buttonWidth > a.scale(112) {
		buttonWidth = a.scale(112)
	}
	if buttonWidth < a.scale(60) {
		buttonWidth = a.scale(60)
	}
	for i, control := range controls {
		moveWindow.Call(control, uintptr(x+i*(buttonWidth+gap)), uintptr(y), uintptr(buttonWidth), uintptr(height), 1)
	}
}

func (a *app) moveSelectedTransfer(action queuePriorityAction) {
	selected := selectedIndices(a.transferList)
	state := deriveQueuePriorityState(a.transferJobs, selected)
	if len(selected) != 1 {
		return
	}
	allowed := false
	switch action {
	case queuePriorityTop:
		allowed = state.MoveTop
	case queuePriorityUp:
		allowed = state.MoveUp
	case queuePriorityDown:
		allowed = state.MoveDown
	case queuePriorityBottom:
		allowed = state.MoveBottom
	default:
		return
	}
	if !allowed {
		return
	}
	index := selected[0]
	if index < 0 || index >= len(a.transferJobs) {
		return
	}

	id := a.transferJobs[index].ID
	var err error
	words := queuePriorityWords(a.languageCode())
	switch action {
	case queuePriorityTop:
		err = a.engine.MoveTransferTop(id)
	case queuePriorityUp:
		err = a.engine.MoveTransferUp(id)
	case queuePriorityDown:
		err = a.engine.MoveTransferDown(id)
	case queuePriorityBottom:
		err = a.engine.MoveTransferBottom(id)
	}
	if err != nil {
		a.setStatus(a.userMessage(err, "error.generic"))
		a.refreshTransfers()
		return
	}
	switch action {
	case queuePriorityTop:
		a.setStatus(words.MovedTop)
	case queuePriorityUp:
		a.setStatus(words.MovedUp)
	case queuePriorityDown:
		a.setStatus(words.MovedDown)
	case queuePriorityBottom:
		a.setStatus(words.MovedBottom)
	}
	// refreshTransfers restores selection by transfer ID, not by the previous
	// row index, so the same job remains selected after any reorder operation.
	a.refreshTransfers()
}
