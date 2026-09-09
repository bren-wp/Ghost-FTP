//go:build linux

package desktop

import (
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

func (u *linuxDesktop) linuxTransferActionState() transferActionState {
	selected := []int(nil)
	if u.selectedTransfer >= 0 && u.selectedTransfer < len(u.transferJobs) {
		selected = []int{u.selectedTransfer}
	}
	return deriveTransferActionState(u.transferJobs, selected, u.connected, u.queuePaused)
}

func (u *linuxDesktop) refreshLinuxTransfersPreservingSelection(id string) {
	u.transferJobs = u.engine.Transfers()
	u.selectedTransfer = -1
	if id == "" {
		return
	}
	for index := range u.transferJobs {
		if u.transferJobs[index].ID == id {
			u.selectedTransfer = index
			return
		}
	}
}

func (u *linuxDesktop) cancelSelectedTransferLinux() {
	state := u.linuxTransferActionState()
	if u.busy || !state.Cancel || u.selectedTransfer < 0 || u.selectedTransfer >= len(u.transferJobs) {
		return
	}

	id := u.transferJobs[u.selectedTransfer].ID
	if err := u.engine.CancelTransfer(id); err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	u.refreshLinuxTransfersPreservingSelection(id)
	u.setStatus(u.tr("common.cancel"))
}

func (u *linuxDesktop) retrySelectedTransferLinux() {
	state := u.linuxTransferActionState()
	if u.busy || !state.Retry || u.selectedTransfer < 0 || u.selectedTransfer >= len(u.transferJobs) {
		return
	}

	id := u.transferJobs[u.selectedTransfer].ID
	if err := u.engine.RetryTransfer(id); err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	u.refreshLinuxTransfersPreservingSelection(id)
	u.setStatus(u.tr("transfer.retry"))
}

func (u *linuxDesktop) clearFinishedTransfersLinux() {
	state := u.linuxTransferActionState()
	if u.busy || !state.Clear {
		return
	}

	selectedID := ""
	if u.selectedTransfer >= 0 && u.selectedTransfer < len(u.transferJobs) {
		selectedID = u.transferJobs[u.selectedTransfer].ID
	}
	u.engine.ClearFinishedTransfers()
	u.refreshLinuxTransfersPreservingSelection(selectedID)
	u.setStatus(u.tr("transfer.clear"))
}
