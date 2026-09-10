//go:build linux

package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	linuxQueuePriorityGap         = 6
	linuxQueuePriorityTopWidth    = 82
	linuxQueuePriorityUpWidth     = 92
	linuxQueuePriorityDownWidth   = 104
	linuxQueuePriorityBottomWidth = 88
)

func (u *linuxDesktop) queuePriorityRects() (linuxRect, linuxRect, linuxRect, linuxRect) {
	y := u.layout.clearQueue.top
	top := linuxRectWH(u.layout.clearQueue.right+linuxQueuePriorityGap, y, linuxQueuePriorityTopWidth, 28)
	up := linuxRectWH(top.right+linuxQueuePriorityGap, y, linuxQueuePriorityUpWidth, 28)
	down := linuxRectWH(up.right+linuxQueuePriorityGap, y, linuxQueuePriorityDownWidth, 28)
	bottom := linuxRectWH(down.right+linuxQueuePriorityGap, y, linuxQueuePriorityBottomWidth, 28)
	return top, up, down, bottom
}

func (u *linuxDesktop) selectedQueuePriorityState() queuePriorityState {
	if u.selectedTransfer < 0 || u.selectedTransfer >= len(u.transferJobs) {
		return queuePriorityState{}
	}
	return deriveQueuePriorityState(u.transferJobs, []int{u.selectedTransfer})
}

func (u *linuxDesktop) renderQueuePriorityControls() error {
	top, up, down, bottom := u.queuePriorityRects()
	state := u.selectedQueuePriorityState()
	words := queuePriorityWords(u.language)
	if err := u.drawButton(top, words.MoveTop, state.MoveTop && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(up, words.MoveUp, state.MoveUp && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(down, words.MoveDown, state.MoveDown && !u.busy, false); err != nil {
		return err
	}
	return u.drawButton(bottom, words.MoveBottom, state.MoveBottom && !u.busy, false)
}

func (u *linuxDesktop) restoreQueuePrioritySelection(id string) {
	u.selectedTransfer = -1
	for index := range u.transferJobs {
		if u.transferJobs[index].ID == id {
			u.selectedTransfer = index
			return
		}
	}
}

func (u *linuxDesktop) moveSelectedQueueTransfer(action queuePriorityAction) {
	if u.busy || u.selectedTransfer < 0 || u.selectedTransfer >= len(u.transferJobs) {
		return
	}
	state := u.selectedQueuePriorityState()
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

	id := u.transferJobs[u.selectedTransfer].ID
	var err error
	switch action {
	case queuePriorityTop:
		err = u.engine.MoveTransferTop(id)
	case queuePriorityUp:
		err = u.engine.MoveTransferUp(id)
	case queuePriorityDown:
		err = u.engine.MoveTransferDown(id)
	case queuePriorityBottom:
		err = u.engine.MoveTransferBottom(id)
	}
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}

	u.transferJobs = u.engine.Transfers()
	u.restoreQueuePrioritySelection(id)
	words := queuePriorityWords(u.language)
	switch action {
	case queuePriorityTop:
		u.setStatus(words.MovedTop)
	case queuePriorityUp:
		u.setStatus(words.MovedUp)
	case queuePriorityDown:
		u.setStatus(words.MovedDown)
	case queuePriorityBottom:
		u.setStatus(words.MovedBottom)
	}
}

func (u *linuxDesktop) handleQueuePriorityMouse(x, y int) bool {
	top, up, down, bottom := u.queuePriorityRects()
	switch {
	case top.contains(x, y):
		u.moveSelectedQueueTransfer(queuePriorityTop)
		return true
	case up.contains(x, y):
		u.moveSelectedQueueTransfer(queuePriorityUp)
		return true
	case down.contains(x, y):
		u.moveSelectedQueueTransfer(queuePriorityDown)
		return true
	case bottom.contains(x, y):
		u.moveSelectedQueueTransfer(queuePriorityBottom)
		return true
	default:
		return false
	}
}
