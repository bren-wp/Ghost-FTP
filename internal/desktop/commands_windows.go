//go:build windows

package desktop

import "path/filepath"

func (a *app) command(id int) {
	switch id {
	case idConnect, idToolbarConnect:
		a.connectNow()
	case idDisconnect, idToolbarDisconnect:
		a.disconnectNow()
	case idSiteManager, idToolbarSites:
		a.openSiteManager()
	case idExitApp:
		postMessageW.Call(a.hwnd, wmClose, 0, 0)
	case idChooseKey:
		a.choosePrivateKey()
	case idSaveProfile:
		a.saveCurrentProfile()
	case idRemoveProfile:
		a.removeCurrentProfile()
	case idSettings, idToolbarSettings:
		a.openSettings()
	case idAbout:
		a.openAbout()
	case idToolbarDiagnostics:
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
	case idRemoteMkdir:
		a.remoteMkdirAction()
	case idRemoteRename:
		a.remoteRenameAction()
	case idRemoteDelete:
		a.remoteDeleteAction()
	case idRemoteChmod:
		a.remoteChmodAction()
	case idUpload, idToolbarUpload:
		a.uploadSelected()
	case idDownload, idToolbarDownload:
		a.downloadSelected()
	case idToolbarNewFolder:
		a.toolbarNewFolderAction()
	case idToolbarRename:
		a.toolbarRenameAction()
	case idToolbarDelete:
		a.toolbarDeleteAction()
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
	case idRefreshAll, idToolbarRefresh:
		a.refreshLocal(getText(a.localPath))
		if a.connected {
			a.refreshRemote(getText(a.remotePath))
		}
		a.setStatus(a.tr("status.refresh_all"))
	}
}
