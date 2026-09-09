//go:build windows

package desktop

import "unsafe"

const (
	idMoveQueueUp   = 508
	idMoveQueueDown = 509
)

var (
	queueGetDlgItem    = user32.NewProc("GetDlgItem")
	queueGetWindowRect = user32.NewProc("GetWindowRect")
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
	if a.queuePriorityButton(idMoveQueueUp) != 0 && a.queuePriorityButton(idMoveQueueDown) != 0 {
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

	if a.queuePriorityButton(idMoveQueueUp) == 0 {
		create(idMoveQueueUp, words.MoveUp, iconUp)
	}
	if a.queuePriorityButton(idMoveQueueDown) == 0 {
		create(idMoveQueueDown, words.MoveDown, iconDownload)
	}
}

func (a *app) updateQueuePriorityControls(state queuePriorityState) {
	a.ensureQueuePriorityControls()
	up := a.queuePriorityButton(idMoveQueueUp)
	down := a.queuePriorityButton(idMoveQueueDown)
	words := queuePriorityWords(a.languageCode())
	a.setButtonLabel(up, words.MoveUp)
	a.setButtonLabel(down, words.MoveDown)
	setControlEnabled(up, state.MoveUp && !a.connectionBusy)
	setControlEnabled(down, state.MoveDown && !a.connectionBusy)
}

func (a *app) layoutQueuePriorityControls() {
	if a == nil || a.hwnd == 0 || a.clearQueue == 0 {
		return
	}
	a.ensureQueuePriorityControls()
	up := a.queuePriorityButton(idMoveQueueUp)
	down := a.queuePriorityButton(idMoveQueueDown)
	if up == 0 || down == 0 {
		return
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

	gap := a.scale(8)
	x := int(bottomRight.X) + gap
	y := int(topLeft.Y)
	height := int(bottomRight.Y - topLeft.Y)
	var client rect
	if ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok == 0 {
		return
	}
	available := int(client.Right) - a.scale(14) - x - gap
	buttonWidth := available / 2
	if buttonWidth > a.scale(132) {
		buttonWidth = a.scale(132)
	}
	if buttonWidth < a.scale(88) {
		buttonWidth = a.scale(88)
	}
	moveWindow.Call(up, uintptr(x), uintptr(y), uintptr(buttonWidth), uintptr(height), 1)
	moveWindow.Call(down, uintptr(x+buttonWidth+gap), uintptr(y), uintptr(buttonWidth), uintptr(height), 1)
}

func (a *app) moveSelectedTransfer(direction int) {
	selected := selectedIndices(a.transferList)
	state := deriveQueuePriorityState(a.transferJobs, selected)
	if len(selected) != 1 {
		return
	}
	if direction < 0 && !state.MoveUp {
		return
	}
	if direction > 0 && !state.MoveDown {
		return
	}
	index := selected[0]
	if index < 0 || index >= len(a.transferJobs) {
		return
	}

	id := a.transferJobs[index].ID
	var err error
	words := queuePriorityWords(a.languageCode())
	if direction < 0 {
		err = a.engine.MoveTransferUp(id)
	} else {
		err = a.engine.MoveTransferDown(id)
	}
	if err != nil {
		a.setStatus(a.userMessage(err, "error.generic"))
		a.refreshTransfers()
		return
	}
	if direction < 0 {
		a.setStatus(words.MovedUp)
	} else {
		a.setStatus(words.MovedDown)
	}
	// The transfer manager emits a full state snapshot. refreshTransfers keeps
	// selection bound to the transfer ID so the same job remains selected after
	// its row position changes.
	a.refreshTransfers()
}
