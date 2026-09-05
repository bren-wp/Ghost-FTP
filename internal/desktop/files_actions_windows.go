//go:build windows

package desktop

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

func (a *app) suppressExpectedDisconnectError(err error) bool {
	return err != nil && errors.Is(err, context.Canceled) && (a.connectionBusy || !a.connected || a.closing)
}

func (a *app) chooseLocalDirectory() {
	p, err := a.engine.ChooseDirectory()
	if err != nil {
		platform.ErrorDialog("Ghost FTP — "+a.tr("common.folder"), a.tr("error.generic"), a.userMessage(err, "error.generic"))
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
				a.setStatus(a.userMessage(err, "error.generic"))
				a.updateActionControls()
				return
			}
			a.localCurrent = resolvedPath
			a.localItems = items
			setText(a.localPath, resolvedPath)
			fillItems(a.localList, items)
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
				a.setStatus(a.userMessage(err, "error.generic"))
				a.checkConnectionAfterError()
				return
			}
			a.remoteCurrent = cleanRemote(target)
			a.remoteAllItems = append(a.remoteAllItems[:0], items...)
			a.remoteItems = append(a.remoteItems[:0], items...)
			setText(a.remotePath, a.remoteCurrent)
			fillItems(a.remoteList, a.remoteItems)
			restoreItemSelection(a.remoteList, a.remoteItems, selected)
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
	it := a.localItems[idx]
	if it.IsDirectory {
		a.refreshLocal(filepath.Join(a.localCurrent, it.Name))
		return
	}
	if a.connected && !a.connectionBusy {
		a.addTransfer("upload", filepath.Join(a.localCurrent, it.Name), path.Join(a.remoteCurrent, it.Name), a.localCurrent)
	}
}

func (a *app) openSelectedRemote() {
	idx := selectedIndex(a.remoteList)
	if idx < 0 || idx >= len(a.remoteItems) || !a.connected || a.connectionBusy {
		return
	}
	it := a.remoteItems[idx]
	if err := security.ValidateRemoteName(it.Name); err != nil {
		a.setStatus(a.tr("error.invalid_name"))
		return
	}
	if it.IsDirectory {
		a.refreshRemote(path.Join(a.remoteCurrent, it.Name))
		return
	}
	if it.IsSymlink {
		a.setStatus(a.tr("status.skipped") + ": " + a.tr("type.link"))
		return
	}
	local, err := security.SafeLocalChild(a.localCurrent, it.Name)
	if err != nil {
		a.setStatus(a.tr("error.invalid_name"))
		return
	}
	a.addTransfer("download", local, path.Join(a.remoteCurrent, it.Name), a.localCurrent)
}

func (a *app) localMkdirAction() {
	name, ok := platform.PromptDialog("Ghost FTP — "+a.tr("common.new_folder"), a.tr("column.name")+":", a.tr("common.new_folder"))
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalMkdir(base, name)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("Ghost FTP", a.tr("common.new_folder"), a.userMessage(err, "error.generic"))
				return
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus(a.tr("common.new_folder") + ": " + name)
		})
	})
}

func (a *app) localRenameAction() {
	indices := selectedIndices(a.localList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.localItems) {
		a.setStatus(a.tr("common.rename") + ": " + a.tr("error.invalid_name"))
		return
	}
	item := a.localItems[indices[0]]
	name, ok := platform.PromptDialog("Ghost FTP — "+a.tr("common.rename"), a.tr("column.name")+":", item.Name)
	if !ok || strings.TrimSpace(name) == item.Name {
		return
	}
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalRename(base, item.Name, strings.TrimSpace(name))
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("Ghost FTP", a.tr("common.rename"), a.userMessage(err, "error.generic"))
				return
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus(a.tr("common.rename") + ": " + item.Name)
		})
	})
}

func (a *app) localDeleteAction() {
	indices := selectedIndices(a.localList)
	if len(indices) == 0 {
		a.setStatus(a.tr("common.delete") + ": " + a.tr("error.invalid_name"))
		return
	}
	items := make([]model.Item, 0, len(indices))
	for _, idx := range indices {
		if idx >= 0 && idx < len(a.localItems) {
			items = append(items, a.localItems[idx])
		}
	}
	if len(items) == 0 {
		return
	}
	if a.settings.ConfirmDelete {
		detail := items[0].Name
		if len(items) > 1 {
			detail = strconv.Itoa(len(items)) + " × " + a.tr("column.name")
		}
		if !platform.ConfirmDialog("Ghost FTP — "+a.tr("common.delete"), a.tr("common.delete")+"?", detail) {
			return
		}
	}
	base := a.localCurrent
	a.goSafe(func() {
		deleted := 0
		var errs []error
		for _, item := range items {
			if err := a.engine.LocalDelete(base, item.Name); err != nil {
				errs = append(errs, err)
				continue
			}
			deleted++
		}
		err := errors.Join(errs...)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("Ghost FTP", a.tr("common.delete"), a.userMessage(err, "error.generic"))
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus(a.tr("common.delete") + ": " + strconv.Itoa(deleted) + " • " + a.tr("status.failed") + ": " + strconv.Itoa(len(items)-deleted))
		})
	})
}

func (a *app) remoteMkdirAction() {
	if !a.connected || a.connectionBusy {
		return
	}
	name, ok := platform.PromptDialog("Ghost FTP — "+a.tr("common.new_folder"), a.tr("column.name")+":", a.tr("common.new_folder"))
	if !ok {
		return
	}
	if err := security.ValidateRemoteName(name); err != nil {
		platform.ErrorDialog("Ghost FTP", a.tr("common.new_folder"), a.userMessage(err, "error.invalid_name"))
		return
	}
	base := a.remoteCurrent
	a.runRemoteMutation(a.tr("common.new_folder"), func(ctx context.Context) error { return a.engine.RemoteMkdir(ctx, base, name) }, a.tr("common.new_folder")+": "+name)
}

func (a *app) remoteRenameAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.remoteItems) || a.connectionBusy {
		a.setStatus(a.tr("common.rename") + ": " + a.tr("error.invalid_name"))
		return
	}
	item := a.remoteItems[indices[0]]
	base := a.remoteCurrent
	name, ok := platform.PromptDialog("Ghost FTP — "+a.tr("common.rename"), a.tr("column.name")+":", item.Name)
	if !ok {
		return
	}
	if err := security.ValidateRemoteName(name); err != nil {
		platform.ErrorDialog("Ghost FTP", a.tr("common.rename"), a.userMessage(err, "error.invalid_name"))
		return
	}
	if name == item.Name {
		return
	}
	a.runRemoteMutation(a.tr("common.rename"), func(ctx context.Context) error {
		return a.engine.RemoteRename(ctx, base, item.Name, name)
	}, a.tr("common.rename")+": "+name)
}

func (a *app) remoteDeleteAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 || a.connectionBusy {
		a.setStatus(a.tr("common.delete") + ": " + a.tr("error.invalid_name"))
		return
	}
	items := make([]model.Item, 0, len(indices))
	for _, idx := range indices {
		if idx >= 0 && idx < len(a.remoteItems) {
			items = append(items, a.remoteItems[idx])
		}
	}
	if len(items) == 0 {
		return
	}
	if a.settings.ConfirmDelete {
		detail := items[0].Name
		if len(items) > 1 {
			detail = strconv.Itoa(len(items)) + " × " + a.tr("column.name")
		}
		if !platform.ConfirmDialog("Ghost FTP — "+a.tr("common.delete"), a.tr("common.delete")+"?", detail) {
			return
		}
	}
	base := a.remoteCurrent
	a.runRemoteBatchMutationWithTimeout(a.tr("common.delete"), remoteBatchTimeout(len(items)), len(items), func(ctx context.Context, index int) error {
		item := items[index]
		return a.engine.RemoteDelete(ctx, base, item.Name, item.IsDirectory)
	}, a.tr("common.delete")+": ", 0)
}

func (a *app) remoteChmodAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 || a.connectionBusy {
		a.setStatus(a.tr("common.permissions") + ": " + a.tr("error.invalid_name"))
		return
	}
	if len(indices) > 1000 {
		a.setStatus(a.tr("error.structure_large"))
		return
	}
	items := make([]model.Item, 0, len(indices))
	skippedLinks := 0
	for _, idx := range indices {
		if idx < 0 || idx >= len(a.remoteItems) {
			continue
		}
		item := a.remoteItems[idx]
		if item.IsSymlink {
			skippedLinks++
			continue
		}
		items = append(items, item)
	}
	if len(items) == 0 {
		a.setStatus(a.tr("status.skipped") + ": " + a.tr("type.link"))
		return
	}
	base := a.remoteCurrent
	mode, ok := platform.PromptDialog("Ghost FTP — "+a.tr("common.permissions"), a.tr("common.permissions")+" (644 / 755):", "644")
	if !ok {
		return
	}
	mode = strings.TrimSpace(mode)
	a.runRemoteBatchMutationWithTimeout(a.tr("common.permissions"), remoteBatchTimeout(len(items)), len(items), func(ctx context.Context, index int) error {
		return a.engine.RemoteChmod(ctx, base, items[index].Name, mode)
	}, a.tr("common.permissions")+": ", skippedLinks)
}

func remoteBatchTimeout(count int) time.Duration {
	if count < 1 {
		count = 1
	}
	t := 90*time.Second + time.Duration(count-1)*2*time.Second
	if t > 10*time.Minute {
		return 10 * time.Minute
	}
	return t
}

func (a *app) runRemoteMutation(label string, operation func(context.Context) error, success string) {
	a.runRemoteMutationWithTimeout(label, 90*time.Second, operation, success)
}

func (a *app) runRemoteMutationWithTimeout(label string, timeout time.Duration, operation func(context.Context) error, success string) {
	if !a.connected || a.connectionBusy {
		return
	}
	baseNavGeneration := a.remoteNavSeq
	connectionGeneration := a.connectionGeneration
	a.setStatus(label + "…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := operation(ctx)
		a.dispatch(func() {
			if connectionGeneration != a.connectionGeneration || !a.connected {
				return
			}
			if err != nil {
				if a.suppressExpectedDisconnectError(err) {
					return
				}
				platform.ErrorDialog("Ghost FTP — "+a.tr("section.remote"), label, a.userMessage(err, "error.generic"))
				a.setStatus(a.userMessage(err, "error.generic"))
				a.checkConnectionAfterError()
				return
			}
			a.setStatus(success)
			if baseNavGeneration == a.remoteNavSeq {
				a.refreshRemote(a.remoteCurrent)
			}
		})
	})
}

func (a *app) runRemoteBatchMutationWithTimeout(label string, timeout time.Duration, count int, operation func(context.Context, int) error, successPrefix string, skipped int) {
	if !a.connected || a.connectionBusy || count <= 0 {
		return
	}
	baseNavGeneration := a.remoteNavSeq
	connectionGeneration := a.connectionGeneration
	a.setStatus(label + "…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		result := executeBatchMutation(ctx, count, operation)
		a.dispatch(func() {
			if connectionGeneration != a.connectionGeneration || !a.connected {
				return
			}
			status := successPrefix + strconv.Itoa(result.Succeeded)
			if result.Failed > 0 {
				status += " • " + a.tr("status.failed") + ": " + strconv.Itoa(result.Failed)
			}
			if skipped > 0 {
				status += " • " + a.tr("status.skipped") + ": " + strconv.Itoa(skipped)
			}
			a.setStatus(status)
			if result.Err != nil && !a.suppressExpectedDisconnectError(result.Err) {
				platform.ErrorDialog("Ghost FTP — "+a.tr("section.remote"), label, a.userMessage(result.Err, "error.generic"))
			}
			if result.Succeeded > 0 && baseNavGeneration == a.remoteNavSeq {
				a.refreshRemote(a.remoteCurrent)
				return
			}
			if result.Err != nil && !a.suppressExpectedDisconnectError(result.Err) {
				a.checkConnectionAfterError()
			}
		})
	})
}

type selectedTreeTransfer struct {
	localPath  string
	remotePath string
}

func (a *app) queueSelection(direction string, files []api.TransferRequest, trees []selectedTreeTransfer, skipped int) {
	if len(files) == 0 && len(trees) == 0 {
		if skipped > 0 {
			a.setStatus(a.tr("status.skipped") + ": " + a.tr("type.link"))
		} else {
			a.setStatus(a.tr("section.transfers") + ": 0")
		}
		return
	}
	generation := a.connectionGeneration
	a.setStatus(a.tr("section.transfers") + "…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		queuedFiles := 0
		queuedDirs := 0
		failedSelections := 0
		var errs []error
		if len(files) > 0 {
			jobs, err := a.engine.AddTransfers(files)
			if err != nil {
				failedSelections += len(files)
				errs = append(errs, err)
			} else {
				queuedFiles += len(jobs)
			}
		}
		for _, tree := range trees {
			if ctx.Err() != nil {
				failedSelections++
				errs = append(errs, ctx.Err())
				continue
			}
			result, err := a.engine.AddTreeTransfer(ctx, direction, tree.localPath, tree.remotePath)
			if err != nil {
				failedSelections++
				errs = append(errs, err)
				continue
			}
			queuedFiles += result.Queued
			queuedDirs += result.Directories
			skipped += result.SkippedSymlinks
		}
		err := errors.Join(errs...)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil && !a.suppressExpectedDisconnectError(err) {
				platform.ErrorDialog("Ghost FTP — "+a.tr("section.transfers"), a.tr("status.failed"), a.userMessage(err, "error.generic"))
			}
			text := a.tr("section.transfers") + ": " + strconv.Itoa(queuedFiles+queuedDirs)
			if failedSelections > 0 {
				text += " • " + a.tr("status.failed") + ": " + strconv.Itoa(failedSelections)
			}
			if skipped > 0 {
				text += " • " + a.tr("status.skipped") + ": " + strconv.Itoa(skipped)
			}
			a.setStatus(text)
			a.refreshTransfers()
		})
	})
}

func (a *app) uploadSelected() {
	indices := selectedIndices(a.localList)
	if len(indices) == 0 {
		a.setStatus(a.tr("transfer.upload") + ": " + a.tr("error.invalid_name"))
		return
	}
	localBase, remoteBase := a.localCurrent, a.remoteCurrent
	files := make([]api.TransferRequest, 0, len(indices))
	trees := make([]selectedTreeTransfer, 0, len(indices))
	skipped := 0
	for _, idx := range indices {
		if idx < 0 || idx >= len(a.localItems) {
			continue
		}
		it := a.localItems[idx]
		if it.IsSymlink {
			skipped++
			continue
		}
		local := filepath.Join(localBase, it.Name)
		remotePath := path.Join(remoteBase, it.Name)
		if it.IsDirectory {
			trees = append(trees, selectedTreeTransfer{localPath: local, remotePath: remotePath})
		} else {
			files = append(files, api.TransferRequest{Direction: "upload", LocalPath: local, RemotePath: remotePath, LocalRoot: localBase})
		}
	}
	a.queueSelection("upload", files, trees, skipped)
}

func (a *app) downloadSelected() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 {
		a.setStatus(a.tr("transfer.download") + ": " + a.tr("error.invalid_name"))
		return
	}
	localBase, remoteBase := a.localCurrent, a.remoteCurrent
	files := make([]api.TransferRequest, 0, len(indices))
	trees := make([]selectedTreeTransfer, 0, len(indices))
	skipped := 0
	for _, idx := range indices {
		if idx < 0 || idx >= len(a.remoteItems) {
			continue
		}
		it := a.remoteItems[idx]
		if it.IsSymlink {
			skipped++
			continue
		}
		if err := security.ValidateRemoteName(it.Name); err != nil {
			skipped++
			continue
		}
		local, err := security.SafeLocalChild(localBase, it.Name)
		if err != nil {
			skipped++
			continue
		}
		remotePath := path.Join(remoteBase, it.Name)
		if it.IsDirectory {
			trees = append(trees, selectedTreeTransfer{localPath: local, remotePath: remotePath})
		} else {
			files = append(files, api.TransferRequest{Direction: "download", LocalPath: local, RemotePath: remotePath, LocalRoot: localBase})
		}
	}
	a.queueSelection("download", files, trees, skipped)
}

func (a *app) addTreeTransfer(direction, local, remotePath string) {
	generation := a.connectionGeneration
	a.setStatus(a.tr("section.transfers") + "…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		result, err := a.engine.AddTreeTransfer(ctx, direction, local, remotePath)
		a.dispatch(func() {
			if generation != a.connectionGeneration {
				return
			}
			if err != nil {
				if a.suppressExpectedDisconnectError(err) {
					return
				}
				platform.ErrorDialog("Ghost FTP — "+a.tr("section.transfers"), a.tr("status.failed"), a.userMessage(err, "error.generic"))
				a.setStatus(a.userMessage(err, "error.generic"))
				return
			}
			a.setStatus(a.tr("section.transfers") + ": " + strconv.Itoa(result.Queued+result.Directories))
			a.refreshTransfers()
		})
	})
}

func (a *app) checkConnectionAfterError() {
	if a.healthCheckRunning || !a.connected || a.connectionBusy {
		return
	}
	generation := a.connectionGeneration
	a.healthCheckRunning = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	a.healthCheckCancel = cancel
	a.goSafe(func() {
		err := a.engine.Probe(ctx)
		cancel()
		a.dispatch(func() {
			if generation != a.connectionGeneration || !a.connected {
				return
			}
			a.healthCheckRunning = false
			a.healthCheckCancel = nil
			if err == nil {
				return
			}
			if a.suppressExpectedDisconnectError(err) {
				return
			}

			disconnectGeneration := a.beginConnectionTransition()
			a.setConnectionBusy(true)
			a.setStatus(a.tr("error.connection_lost"))
			a.goSafe(func() {
				disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
				disconnectErr := a.engine.Disconnect(disconnectCtx)
				disconnectCancel()
				a.dispatch(func() {
					if disconnectGeneration != a.connectionGeneration {
						return
					}
					status := a.tr("error.connection_lost")
					if disconnectErr != nil {
						status = a.userMessage(disconnectErr, "error.connection_lost")
					}
					a.finishDisconnected(status)
				})
			})
		})
	})
}
