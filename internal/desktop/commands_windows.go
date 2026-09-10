//go:build windows

package desktop

import (
	"path/filepath"

	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

func (a *app) command(id int) {
	// Some legacy file-operation call sites intentionally keep the compact
	// platform.PromptDialog wrapper. Synchronize its action labels at the command
	// boundary so the modal always follows Ghost FTP's active runtime language
	// instead of falling back to English (for example Croatian "Otkaži").
	platform.SetDialogActionLabels(okLabel(a.languageCode()), a.tr("common.cancel"))

	switch id {
	case idConnect:
		a.connectNow()
	case idDisconnect:
		a.disconnectNow()
	case idSiteManager:
		a.openSiteManager()
	case idChooseKey:
		a.choosePrivateKey()
	case idSaveProfile:
		a.saveCurrentProfile()
	case idRemoveProfile:
		a.removeCurrentProfile()
	case idSettings:
		a.openSettings()
	case idAbout:
		a.openAbout()
	case idDiagnostics:
		a.showDiagnostics()
	case idLocalRefresh:
		a.refreshLocal(getText(a.localPath))
	case idLocalChoose:
		a.chooseLocalDirectory()
	case idRemoteRefresh:
		a.refreshRemote(getText(a.remotePath))
	case idLocalUp:
		a.refreshLocal(filepath.Dir(getText(a.localPath)))
	case idRemoteUp:
		a.remoteUpOne()
	case idLocalMkdir:
		a.localMkdirAction()
	case idLocalRename:
		a.localRenameAction()
	case idLocalDelete:
		a.localDeleteAction()
	case idLocalFilter:
		a.localFilterAction()
	case idRemoteMkdir:
		a.remoteMkdirAction()
	case idRemoteRename:
		a.remoteRenameAction()
	case idRemoteDelete:
		a.remoteDeleteAction()
	case idRemoteChmod:
		a.remoteChmodAction()
	case idRemoteEdit:
		a.remoteEditAction()
	case idRemoteFilter:
		a.remoteFilterAction()
	case idUpload:
		a.uploadSelected()
	case idDownload:
		a.downloadSelected()
	case idPauseQueue:
		a.pauseTransfers()
	case idResumeQueue:
		a.resumeTransfers()
	case idCancelJob:
		a.cancelSelectedTransfer()
	case idRetryJob:
		a.retrySelectedTransfer()
	case idClearQueue:
		a.clearFinishedTransfers()
	case idMoveQueueUp:
		a.moveSelectedTransfer(-1)
	case idMoveQueueDown:
		a.moveSelectedTransfer(1)
	case idRefreshAll:
		a.refreshLocal(getText(a.localPath))
		if a.connected {
			a.refreshRemote(getText(a.remotePath))
		}
		a.setStatus(a.tr("status.refresh_all"))
	case idFocusLocalPath:
		focusAndSelectEdit(a.localPath)
	case idFocusRemotePath:
		if a.connected && !a.connectionBusy {
			focusAndSelectEdit(a.remotePath)
		}
	}
}