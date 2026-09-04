//go:build windows

package desktop

import (
	"context"
	"errors"
	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (a *app) suppressExpectedDisconnectError(err error) bool {
	return err != nil && errors.Is(err, context.Canceled) && (a.connectionBusy || !a.connected || a.closing)
}

func (a *app) chooseLocalDirectory() {
	p, err := a.engine.ChooseDirectory()
	if err != nil {
		platform.ErrorDialog("GhostFTP", "Odabir mape nije uspio", usererror.Message(err, "Mapu trenutačno nije moguće odabrati."))
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
				a.setStatus(usererror.Message(err, "Lokalnu mapu nije moguće otvoriti."))
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
				a.setStatus(usererror.Message(err, "Mapu na poslužitelju nije moguće otvoriti."))
				a.checkConnectionAfterError()
				return
			}
			a.remoteCurrent = cleanRemote(target)
			a.remoteItems = items
			setText(a.remotePath, a.remoteCurrent)
			fillItems(a.remoteList, items)
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
		a.setStatus("Ovu stavku nije moguće sigurno otvoriti.")
		return
	}
	if it.IsDirectory {
		a.refreshRemote(path.Join(a.remoteCurrent, it.Name))
		return
	}
	if it.IsSymlink {
		a.setStatus("Simbolička poveznica nije automatski preuzeta radi sigurnosti.")
		return
	}
	local, err := security.SafeLocalChild(a.localCurrent, it.Name)
	if err != nil {
		a.setStatus("Ovu stavku nije moguće sigurno spremiti na lokalno računalo.")
		return
	}
	a.addTransfer("download", local, path.Join(a.remoteCurrent, it.Name), a.localCurrent)
}

func (a *app) localMkdirAction() {
	name, ok := platform.PromptDialog("GhostFTP — nova lokalna mapa", "Naziv nove mape:", "Nova mapa")
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalMkdir(base, name)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("GhostFTP", "Mapa nije stvorena", usererror.Message(err, "Mapu nije moguće stvoriti."))
				return
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus("Lokalna mapa stvorena: " + name)
		})
	})
}

func (a *app) localRenameAction() {
	indices := selectedIndices(a.localList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.localItems) {
		a.setStatus("Za preimenovanje odaberite točno jednu lokalnu stavku.")
		return
	}
	item := a.localItems[indices[0]]
	name, ok := platform.PromptDialog("GhostFTP — preimenuj lokalno", "Novi naziv:", item.Name)
	if !ok || strings.TrimSpace(name) == item.Name {
		return
	}
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalRename(base, item.Name, strings.TrimSpace(name))
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("GhostFTP", "Preimenovanje nije uspjelo", usererror.Message(err, "Stavku nije moguće preimenovati."))
				return
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus("Lokalna stavka preimenovana.")
		})
	})
}

func (a *app) localDeleteAction() {
	indices := selectedIndices(a.localList)
	if len(indices) == 0 {
		a.setStatus("Odaberite jednu ili više lokalnih stavki za brisanje.")
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
			message = strconv.Itoa(len(items)) + " odabranih stavki"
		}
		if !platform.ConfirmDialog("GhostFTP — brisanje", "Obrisati odabrane lokalne stavke?", message+"\n\nOva radnja se ne može poništiti kroz GhostFTP.") {
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
				platform.ErrorDialog("GhostFTP", "Nisu obrisane sve stavke", usererror.Message(err, "Dio odabranih stavki nije moguće obrisati."))
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus("Obrisano lokalnih stavki: " + strconv.Itoa(deleted) + " • neuspjelo: " + strconv.Itoa(len(items)-deleted))
		})
	})
}

func (a *app) remoteMkdirAction() {
	if !a.connected || a.connectionBusy {
		return
	}
	name, ok := platform.PromptDialog("GhostFTP — nova mapa na poslužitelju", "Naziv nove mape:", "Nova mapa")
	if !ok {
		return
	}
	if err := security.ValidateRemoteName(name); err != nil {
		platform.ErrorDialog("GhostFTP", "Mapa nije stvorena", usererror.Message(err, "Naziv udaljene mape nije valjan."))
		return
	}
	base := a.remoteCurrent
	a.runRemoteMutation("Stvaranje mape", func(ctx context.Context) error { return a.engine.RemoteMkdir(ctx, base, name) }, "Mapa stvorena: "+name)
}

func (a *app) remoteRenameAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.remoteItems) || a.connectionBusy {
		a.setStatus("Za preimenovanje odaberite točno jednu udaljenu stavku.")
		return
	}
	item := a.remoteItems[indices[0]]
	base := a.remoteCurrent
	name, ok := platform.PromptDialog("GhostFTP — preimenuj na poslužitelju", "Novi naziv:", item.Name)
	if !ok {
		return
	}
	if err := security.ValidateRemoteName(name); err != nil {
		platform.ErrorDialog("GhostFTP", "Preimenovanje nije uspjelo", usererror.Message(err, "Novi naziv udaljene stavke nije valjan."))
		return
	}
	if name == item.Name {
		return
	}
	a.runRemoteMutation("Preimenovanje", func(ctx context.Context) error {
		return a.engine.RemoteRename(ctx, base, item.Name, name)
	}, "Udaljena stavka preimenovana.")
}

func (a *app) remoteDeleteAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 || a.connectionBusy {
		a.setStatus("Odaberite jednu ili više udaljenih stavki za brisanje.")
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
			message = strconv.Itoa(len(items)) + " odabranih stavki"
		}
		if !platform.ConfirmDialog("GhostFTP — brisanje na poslužitelju", "Obrisati odabrane stavke?", message+"\n\nBrisanje na poslužitelju može biti nepovratno.") {
			return
		}
	}
	base := a.remoteCurrent
	a.runRemoteBatchMutationWithTimeout("Brisanje", remoteBatchTimeout(len(items)), len(items), func(ctx context.Context, index int) error {
		item := items[index]
		return a.engine.RemoteDelete(ctx, base, item.Name, item.IsDirectory)
	}, "Obrisano udaljenih stavki: ", 0)
}

func (a *app) remoteChmodAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 || a.connectionBusy {
		a.setStatus("Odaberite jednu ili više stavki za promjenu dozvola.")
		return
	}
	if len(indices) > 1000 {
		a.setStatus("Za jednu promjenu dozvola odaberite najviše 1000 stavki.")
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
		a.setStatus("Dozvole nisu mijenjane: simboličke poveznice preskaču se radi sigurnosti.")
		return
	}
	base := a.remoteCurrent
	mode, ok := platform.PromptDialog("GhostFTP — dozvole", "Dozvole za odabrane stavke, npr. 644 ili 755:", "644")
	if !ok {
		return
	}
	mode = strings.TrimSpace(mode)
	a.runRemoteBatchMutationWithTimeout("Promjena dozvola", remoteBatchTimeout(len(items)), len(items), func(ctx context.Context, index int) error {
		return a.engine.RemoteChmod(ctx, base, items[index].Name, mode)
	}, "Dozvole promijenjene za stavki: ", skippedLinks)
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
				platform.ErrorDialog("GhostFTP — poslužitelj", label+" nije uspjelo", usererror.Message(err, "Radnju na poslužitelju nije moguće dovršiti."))
				a.setStatus(usererror.Message(err, label+" nije uspjelo."))
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
				status += " • neuspjelo: " + strconv.Itoa(result.Failed)
			}
			if skipped > 0 {
				status += " • preskočeno poveznica: " + strconv.Itoa(skipped)
			}
			a.setStatus(status)
			if result.Err != nil && !a.suppressExpectedDisconnectError(result.Err) {
				platform.ErrorDialog("GhostFTP — poslužitelj", label+" nije dovršeno za sve stavke", usererror.Message(result.Err, "Dio odabranih stavki nije moguće promijeniti."))
			}
			if result.Succeeded > 0 && baseNavGeneration == a.remoteNavSeq {
				// Refresh itself proves whether the connection is still usable, so do
				// not start a redundant probe in parallel after partial success.
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
			a.setStatus("Odabrane poveznice nisu prenesene radi sigurnosti.")
		} else {
			a.setStatus("Nema stavki za prijenos.")
		}
		return
	}
	generation := a.connectionGeneration
	a.setStatus("Dodavanje odabranih stavki u red prijenosa…")
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
				platform.ErrorDialog("GhostFTP — prijenos", "Nisu dodane sve odabrane stavke", usererror.Message(err, "Dio odabranih stavki nije moguće dodati u prijenos."))
			}
			text := "Dodano u red: " + strconv.Itoa(queuedFiles) + " datoteka"
			if queuedDirs > 0 {
				text += ", " + strconv.Itoa(queuedDirs) + " mapa"
			}
			if failedSelections > 0 {
				text += " • neuspjelo odabira: " + strconv.Itoa(failedSelections)
			}
			if skipped > 0 {
				text += " • preskočeno poveznica: " + strconv.Itoa(skipped)
			}
			a.setStatus(text)
			a.refreshTransfers()
		})
	})
}

func (a *app) uploadSelected() {
	indices := selectedIndices(a.localList)
	if len(indices) == 0 {
		a.setStatus("Odaberite jednu ili više lokalnih datoteka ili mapa za prijenos.")
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
		a.setStatus("Odaberite jednu ili više datoteka ili mapa na poslužitelju za preuzimanje.")
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
	a.setStatus("Priprema prijenosa cijele mape…")
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
				platform.ErrorDialog("GhostFTP — prijenos mape", "Mapa nije dodana u prijenos", usererror.Message(err, "Prijenos mape nije moguće pokrenuti."))
				a.setStatus(usererror.Message(err, "Prijenos mape nije moguće pokrenuti."))
				return
			}
			a.setStatus("Mapa dodana: " + strconv.Itoa(result.Queued) + " datoteka, " + strconv.Itoa(result.Directories) + " mapa.")
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
			a.setStatus("Veza više nije dostupna. Zaustavljanje prijenosa…")
			a.goSafe(func() {
				disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
				disconnectErr := a.engine.Disconnect(disconnectCtx)
				disconnectCancel()
				a.dispatch(func() {
					if disconnectGeneration != a.connectionGeneration {
						return
					}
					status := "Veza više nije dostupna. Aktivni prijenosi su zaustavljeni; poslužitelj i korisničko ime ostali su uneseni."
					if disconnectErr != nil {
						status = usererror.Message(disconnectErr, status)
					}
					a.finishDisconnected(status)
				})
			})
		})
	})
}
