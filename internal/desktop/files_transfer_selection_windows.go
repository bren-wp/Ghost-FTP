//go:build windows

package desktop

import (
	"context"
	"errors"
	"path"
	"path/filepath"
	"time"

	"brendigo.com/byftp/internal/api"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
)

type selectedTreeTransfer struct {
	localPath  string
	remotePath string
}

func (a *app) queueSelection(direction string, files []api.TransferRequest, trees []selectedTreeTransfer, skipped int) {
	if len(files) == 0 && len(trees) == 0 {
		if skipped > 0 {
			a.setStatus(a.tr("transfer.links_skipped"))
		} else {
			a.setStatus(a.tr("transfer.none"))
		}
		return
	}
	generation := a.connectionGeneration
	a.setStatus(a.tr("transfer.adding_selection"))
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
				platform.ErrorDialog(a.tr("transfer.tree_title"), a.tr("transfer.partial_title"), a.userMessage(err, "transfer.partial_body"))
			}
			text := a.tr("transfer.queue_files", queuedFiles)
			if queuedDirs > 0 {
				text += a.tr("transfer.queue_dirs", queuedDirs)
			}
			if failedSelections > 0 {
				text += a.tr("transfer.queue_failed", failedSelections)
			}
			if skipped > 0 {
				text += a.tr("transfer.queue_skipped_links", skipped)
			}
			a.setStatus(text)
			a.refreshTransfers()
		})
	})
}

func (a *app) uploadSelected() {
	indices := selectedIndices(a.localList)
	if len(indices) == 0 {
		a.setStatus(a.tr("transfer.select_upload"))
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
		item := a.localItems[idx]
		if item.IsSymlink {
			skipped++
			continue
		}
		local := filepath.Join(localBase, item.Name)
		remotePath := path.Join(remoteBase, item.Name)
		if item.IsDirectory {
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
		a.setStatus(a.tr("transfer.select_download"))
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
		item := a.remoteItems[idx]
		if item.IsSymlink {
			skipped++
			continue
		}
		if err := security.ValidateRemoteName(item.Name); err != nil {
			skipped++
			continue
		}
		local, err := security.SafeLocalChild(localBase, item.Name)
		if err != nil {
			skipped++
			continue
		}
		remotePath := path.Join(remoteBase, item.Name)
		if item.IsDirectory {
			trees = append(trees, selectedTreeTransfer{localPath: local, remotePath: remotePath})
		} else {
			files = append(files, api.TransferRequest{Direction: "download", LocalPath: local, RemotePath: remotePath, LocalRoot: localBase})
		}
	}
	a.queueSelection("download", files, trees, skipped)
}

func (a *app) addTreeTransfer(direction, local, remotePath string) {
	generation := a.connectionGeneration
	a.setStatus(a.tr("transfer.tree_preparing"))
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
				platform.ErrorDialog(a.tr("transfer.tree_title"), a.tr("transfer.tree_failed_title"), a.userMessage(err, "transfer.tree_failed_body"))
				a.setStatus(a.userMessage(err, "transfer.tree_failed_body"))
				return
			}
			a.setStatus(a.tr("transfer.tree_added", result.Queued, result.Directories))
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
			if err == nil || a.suppressExpectedDisconnectError(err) {
				return
			}

			disconnectGeneration := a.beginConnectionTransition()
			a.setConnectionBusy(true)
			a.setStatus(a.tr("connection.lost_stopping"))
			a.goSafe(func() {
				disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
				disconnectErr := a.engine.Disconnect(disconnectCtx)
				disconnectCancel()
				a.dispatch(func() {
					if disconnectGeneration != a.connectionGeneration {
						return
					}
					status := a.tr("connection.lost_done")
					if disconnectErr != nil {
						status = a.userMessage(disconnectErr, "connection.lost_done")
					}
					a.finishDisconnected(status)
				})
			})
		})
	})
}
