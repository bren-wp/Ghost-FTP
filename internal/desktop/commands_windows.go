//go:build windows

package desktop

import "path/filepath"

func (a *app) command(id int) {
	switch id {
	case idConnect:
		a.connectNow()
	case idDisconnect:
		a.disconnectNow()
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
	case idRemoteMkdir:
		a.remoteMkdirAction()
	case idRemoteRename:
		a.remoteRenameAction()
	case idRemoteDelete:
		a.remoteDeleteAction()
	case idRemoteChmod:
		a.remoteChmodAction()
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
	case idRefreshAll:
		a.refreshLocal(getText(a.localPath))
		if a.connected {
			a.refreshRemote(getText(a.remotePath))
		}
		a.setStatus("Osvježavanje lokalnog i udaljenog prikaza…")
	}
}
