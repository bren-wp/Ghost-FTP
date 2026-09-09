//go:build linux

package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	linuxQueuePriorityGap       = 8
	linuxQueuePriorityUpWidth   = 96
	linuxQueuePriorityDownWidth = 112
)

func (u *linuxDesktop) queuePriorityRects() (linuxRect, linuxRect) {
	y := u.layout.clearQueue.top
	up := linuxRectWH(u.layout.clearQueue.right+linuxQueuePriorityGap, y, linuxQueuePriorityUpWidth, 28)
	down := linuxRectWH(up.right+linuxQueuePriorityGap, y, linuxQueuePriorityDownWidth, 28)
	return up, down
}

func (u *linuxDesktop) selectedQueuePriorityState() queuePriorityState {
	if u.selectedTransfer < 0 || u.selectedTransfer >= len(u.transferJobs) {
		return queuePriorityState{}
	}
	return deriveQueuePriorityState(u.transferJobs, []int{u.selectedTransfer})
}

func (u *linuxDesktop) renderQueuePriorityControls() error {
	up, down := u.queuePriorityRects()
	state := u.selectedQueuePriorityState()
	words := queuePriorityWords(u.language)
	if err := u.drawButton(up, words.MoveUp, state.MoveUp && !u.busy, false); err != nil {
		return err
	}
	return u.drawButton(down, words.MoveDown, state.MoveDown && !u.busy, false)
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

func (u *linuxDesktop) moveSelectedQueueTransfer(moveUp bool) {
	if u.busy || u.selectedTransfer < 0 || u.selectedTransfer >= len(u.transferJobs) {
		return
	}
	state := u.selectedQueuePriorityState()
	if (moveUp && !state.MoveUp) || (!moveUp && !state.MoveDown) {
		return
	}

	id := u.transferJobs[u.selectedTransfer].ID
	var err error
	if moveUp {
		err = u.engine.MoveTransferUp(id)
	} else {
		err = u.engine.MoveTransferDown(id)
	}
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}

	u.transferJobs = u.engine.Transfers()
	u.restoreQueuePrioritySelection(id)
	words := queuePriorityWords(u.language)
	if moveUp {
		u.setStatus(words.MovedUp)
	} else {
		u.setStatus(words.MovedDown)
	}
}

func (u *linuxDesktop) handleQueuePriorityMouse(x, y int) bool {
	up, down := u.queuePriorityRects()
	if up.contains(x, y) {
		u.moveSelectedQueueTransfer(true)
		return true
	}
	if down.contains(x, y) {
		u.moveSelectedQueueTransfer(false)
		return true
	}
	return false
}
