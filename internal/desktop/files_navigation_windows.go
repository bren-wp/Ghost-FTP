//go:build windows

package desktop

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"strings"
	"time"

	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

func (a *app) suppressExpectedDisconnectError(err error) bool {
	return err != nil && errors.Is(err, context.Canceled) && (a.connectionBusy || !a.connected || a.closing)
}

func (a *app) chooseLocalDirectory() {
	p, err := a.engine.ChooseDirectory()
	if err != nil {
		platform.ErrorDialog("ByFTP", a.tr("file.choose_folder_failed_title"), a.userMessage(err, "file.choose_folder_failed_body"))
		return
	}
	if p != "" {
		a.refreshLocal(p)
	}
}

func (a *app) refreshLocal(p string) {
	selected := map[string]struct{}{}
	if strings.TrimSpace(p) != "" && a.localCurrent != "" && filepath.Clean(p) == filepath.Clean(a.localCurrent) {
		selected = selectedItemNames(a.localList, a.localItems)
	}
	if a.localNavCancel != nil {
		a.localNavCancel()
	}
	a.localNavSeq++
	seq := a.localNavSeq
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	a.localNavCancel = cancel
	a.goSafe(func() {
		defer cancel()
		resolvedPath, items, err := a.engine.LocalList(ctx, p)
		a.dispatch(func() {
			if seq != a.localNavSeq {
				return
			}
			a.localNavCancel = nil
			if err != nil {
				a.setStatus(a.userMessage(err, "local.open_failed"))
				a.updateActionControls()
				return
			}
			a.localCurrent = resolvedPath
			a.localItems = items
			setText(a.localPath, resolvedPath)
			a.fillItemList(a.localList, items)
			restoreItemSelection(a.localList, items, selected)
			a.updateActionControls()
		})
	})
}

func (a *app) refreshRemote(p string) {
	if !a.connected || a.connectionBusy {
		return
	}
	if strings.TrimSpace(p) == "" {
		if a.protocolValue() == "sftp" {
			p = "."
		} else {
			p = "/"
		}
	}
	p = cleanRemote(p)
	selected := map[string]struct{}{}
	if p == a.remoteCurrent {
		selected = selectedItemNames(a.remoteList, a.remoteItems)
	}
	if a.remoteNavCancel != nil {
		a.remoteNavCancel()
	}
	a.remoteNavSeq++
	seq := a.remoteNavSeq
	generation := a.connectionGeneration
	target := p
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	a.remoteNavCancel = cancel
	a.goSafe(func() {
		defer cancel()
		items, err := a.engine.RemoteList(ctx, target)
		a.dispatch(func() {
			if seq != a.remoteNavSeq || generation != a.connectionGeneration || !a.connected {
				return
			}
			a.remoteNavCancel = nil
			if err != nil {
				if a.suppressExpectedDisconnectError(err) {
					return
				}
				a.setStatus(a.userMessage(err, "remote.open_failed"))
				a.checkConnectionAfterError()
				return
			}
			a.remoteCurrent = cleanRemote(target)
			a.remoteItems = items
			setText(a.remotePath, a.remoteCurrent)
			a.fillItemList(a.remoteList, items)
			restoreItemSelection(a.remoteList, items, selected)
			a.updateActionControls()
		})
	})
}

func (a *app) remoteUpOne() {
	a.refreshRemote(remoteParent(getText(a.remotePath)))
}

func (a *app) openSelectedLocal() {
	idx := selectedIndex(a.localList)
	if idx < 0 || idx >= len(a.localItems) {
		return
	}
	item := a.localItems[idx]
	if item.IsDirectory {
		a.refreshLocal(filepath.Join(a.localCurrent, item.Name))
		return
	}
	if a.connected && !a.connectionBusy {
		a.addTransfer("upload", filepath.Join(a.localCurrent, item.Name), path.Join(a.remoteCurrent, item.Name), a.localCurrent)
	}
}

func (a *app) openSelectedRemote() {
	idx := selectedIndex(a.remoteList)
	if idx < 0 || idx >= len(a.remoteItems) || !a.connected || a.connectionBusy {
		return
	}
	item := a.remoteItems[idx]
	if err := security.ValidateRemoteName(item.Name); err != nil {
		a.setStatus(a.tr("item.open_unsafe"))
		return
	}
	if item.IsDirectory {
		a.refreshRemote(path.Join(a.remoteCurrent, item.Name))
		return
	}
	if item.IsSymlink {
		a.setStatus(a.tr("item.symlink_download_blocked"))
		return
	}
	local, err := security.SafeLocalChild(a.localCurrent, item.Name)
	if err != nil {
		a.setStatus(a.tr("item.save_unsafe"))
		return
	}
	a.addTransfer("download", local, path.Join(a.remoteCurrent, item.Name), a.localCurrent)
}
