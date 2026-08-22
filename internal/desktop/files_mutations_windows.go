//go:build windows

package desktop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
)

func (a *app) promptText(title, prompt, defaultValue string) (string, bool) {
	return platform.PromptDialogWithLabels(title, prompt, defaultValue, okLabel(a.languageCode()), a.tr("common.cancel"))
}

func (a *app) localMkdirAction() {
	name, ok := a.promptText(a.tr("local.mkdir.title"), a.tr("file.folder_name"), a.tr("file.default_folder"))
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalMkdir(base, name)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP", a.tr("local.mkdir.failed_title"), a.userMessage(err, "local.mkdir.failed_body"))
				return
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus(a.tr("local.mkdir.success", name))
		})
	})
}

func (a *app) localRenameAction() {
	indices := selectedIndices(a.localList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.localItems) {
		a.setStatus(a.tr("local.rename.select_one"))
		return
	}
	item := a.localItems[indices[0]]
	name, ok := a.promptText(a.tr("local.rename.title"), a.tr("file.new_name"), item.Name)
	name = strings.TrimSpace(name)
	if !ok || name == item.Name {
		return
	}
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalRename(base, item.Name, name)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP", a.tr("local.rename.failed_title"), a.userMessage(err, "local.rename.failed_body"))
				return
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus(a.tr("local.rename.success"))
		})
	})
}

func (a *app) localDeleteAction() {
	indices := selectedIndices(a.localList)
	if len(indices) == 0 {
		a.setStatus(a.tr("local.delete.select"))
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
		message := items[0].Name
		if len(items) > 1 {
			message = a.tr("selection.items", len(items))
		}
		if !platform.ConfirmDialog(a.tr("local.delete.title"), a.tr("local.delete.question"), message+"\n\n"+a.tr("local.delete.warning")) {
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
				platform.ErrorDialog("ByFTP", a.tr("local.delete.partial_title"), a.userMessage(err, "local.delete.partial_body"))
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus(a.tr("local.delete.status", deleted, len(items)-deleted))
		})
	})
}

func (a *app) remoteMkdirAction() {
	if !a.connected || a.connectionBusy {
		return
	}
	name, ok := a.promptText(a.tr("remote.mkdir.title"), a.tr("file.folder_name"), a.tr("file.default_folder"))
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	base := a.remoteCurrent
	a.runRemoteMutation(a.tr("remote.mkdir.progress"), func(ctx context.Context) error { return a.engine.RemoteMkdir(ctx, base, name) }, a.tr("remote.mkdir.success", name))
}

func (a *app) remoteRenameAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.remoteItems) || a.connectionBusy {
		a.setStatus(a.tr("remote.rename.select_one"))
		return
	}
	item := a.remoteItems[indices[0]]
	base := a.remoteCurrent
	name, ok := a.promptText(a.tr("remote.rename.title"), a.tr("file.new_name"), item.Name)
	name = strings.TrimSpace(name)
	if !ok || name == item.Name {
		return
	}
	a.runRemoteMutation(a.tr("remote.rename.progress"), func(ctx context.Context) error {
		return a.engine.RemoteRename(ctx, base, item.Name, name)
	}, a.tr("remote.rename.success"))
}

func (a *app) remoteDeleteAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 || a.connectionBusy {
		a.setStatus(a.tr("remote.delete.select"))
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
		message := items[0].Name
		if len(items) > 1 {
			message = a.tr("selection.items", len(items))
		}
		if !platform.ConfirmDialog(a.tr("remote.delete.title"), a.tr("remote.delete.question"), message+"\n\n"+a.tr("remote.delete.warning")) {
			return
		}
	}
	base := a.remoteCurrent
	a.runRemoteBatchMutationWithTimeout(a.tr("remote.delete.progress"), remoteBatchTimeout(len(items)), len(items), func(ctx context.Context, index int) error {
		item := items[index]
		return a.engine.RemoteDelete(ctx, base, item.Name, item.IsDirectory)
	}, a.tr("remote.delete.success_prefix"), 0)
}

func (a *app) remoteChmodAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 || a.connectionBusy {
		a.setStatus(a.tr("permissions.select"))
		return
	}
	if len(indices) > 1000 {
		a.setStatus(a.tr("permissions.limit"))
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
		a.setStatus(a.tr("permissions.links_only"))
		return
	}
	base := a.remoteCurrent
	mode, ok := a.promptText(a.tr("permissions.title"), a.tr("permissions.prompt"), "644")
	if !ok {
		return
	}
	mode = strings.TrimSpace(mode)
	a.runRemoteBatchMutationWithTimeout(a.tr("permissions.progress"), remoteBatchTimeout(len(items)), len(items), func(ctx context.Context, index int) error {
		return a.engine.RemoteChmod(ctx, base, items[index].Name, mode)
	}, a.tr("permissions.success_prefix"), skippedLinks)
}

func remoteBatchTimeout(count int) time.Duration {
	if count < 1 {
		count = 1
	}
	timeout := 90*time.Second + time.Duration(count-1)*2*time.Second
	if timeout > 10*time.Minute {
		return 10 * time.Minute
	}
	return timeout
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
				platform.ErrorDialog(a.tr("remote.action.title"), a.tr("remote.action.failed", label), a.userMessage(err, "remote.action.body"))
				a.setStatus(a.userMessage(err, "remote.action.failed", label))
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
				status += a.tr("batch.failed", result.Failed)
			}
			if skipped > 0 {
				status += a.tr("batch.skipped_links", skipped)
			}
			a.setStatus(status)
			if result.Err != nil && !a.suppressExpectedDisconnectError(result.Err) {
				platform.ErrorDialog(a.tr("remote.action.title"), a.tr("remote.batch.partial", label), a.userMessage(result.Err, "remote.batch.body"))
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

func batchSummary(prefix string, count int) string {
	return fmt.Sprintf("%s%d", prefix, count)
}
