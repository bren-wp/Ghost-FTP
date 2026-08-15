//go:build windows

package desktop

import (
	"brendigo.com/byftp/internal/api"
	"brendigo.com/byftp/internal/model"
	"brendigo.com/byftp/internal/platform"
	"brendigo.com/byftp/internal/security"
	"brendigo.com/byftp/internal/usererror"
	"context"
	"errors"
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
		platform.ErrorDialog("ByFTP", "Odabir mape nije uspio", usererror.Message(err, "Mapu trenutačno nije moguće odabrati."))
		return
	}
	if p != "" {
		a.refreshLocal(p)
	}
}

func (a *app) refreshLocal(p string) {
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
				return
			}
			a.localCurrent = resolvedPath
			a.localItems = items
			setText(a.localPath, resolvedPath)
			fillItems(a.localList, items)
		})
	})
}

func (a *app) refreshRemote(p string) {
	if !a.connected {
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
	if a.remoteNavCancel != nil {
		a.remoteNavCancel()
	}
	a.remoteNavSeq++
	seq := a.remoteNavSeq
	target := p
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	a.remoteNavCancel = cancel
	a.goSafe(func() {
		defer cancel()
		items, err := a.engine.RemoteList(ctx, target)
		a.dispatch(func() {
			if seq != a.remoteNavSeq || !a.connected {
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
	if a.connected {
		a.addTransfer("upload", filepath.Join(a.localCurrent, it.Name), path.Join(a.remoteCurrent, it.Name), a.localCurrent)
	}
}

func (a *app) openSelectedRemote() {
	idx := selectedIndex(a.remoteList)
	if idx < 0 || idx >= len(a.remoteItems) {
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
	local, err := security.SafeLocalChild(a.localCurrent, it.Name)
	if err != nil {
		a.setStatus("Ovu stavku nije moguće sigurno spremiti na lokalno računalo.")
		return
	}
	a.addTransfer("download", local, path.Join(a.remoteCurrent, it.Name), a.localCurrent)
}

func (a *app) localMkdirAction() {
	name, ok := platform.PromptDialog("ByFTP — nova lokalna mapa", "Naziv nove mape:", "Nova mapa")
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalMkdir(base, name)
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP", "Mapa nije stvorena", usererror.Message(err, "Mapu nije moguće stvoriti."))
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
	name, ok := platform.PromptDialog("ByFTP — preimenuj lokalno", "Novi naziv:", item.Name)
	if !ok || strings.TrimSpace(name) == item.Name {
		return
	}
	base := a.localCurrent
	a.goSafe(func() {
		err := a.engine.LocalRename(base, item.Name, strings.TrimSpace(name))
		a.dispatch(func() {
			if err != nil {
				platform.ErrorDialog("ByFTP", "Preimenovanje nije uspjelo", usererror.Message(err, "Stavku nije moguće preimenovati."))
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
		if !platform.ConfirmDialog("ByFTP — brisanje", "Obrisati odabrane lokalne stavke?", message+"\n\nOva radnja se ne može poništiti kroz ByFTP.") {
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
				platform.ErrorDialog("ByFTP", "Nisu obrisane sve stavke", usererror.Message(err, "Dio odabranih stavki nije moguće obrisati."))
			}
			a.refreshLocal(a.localCurrent)
			a.setStatus("Obrisano lokalnih stavki: " + strconv.Itoa(deleted))
		})
	})
}

func (a *app) remoteMkdirAction() {
	if !a.connected {
		return
	}
	name, ok := platform.PromptDialog("ByFTP — nova mapa na poslužitelju", "Naziv nove mape:", "Nova mapa")
	if !ok {
		return
	}
	name = strings.TrimSpace(name)
	base := a.remoteCurrent
	a.runRemoteMutation("Stvaranje mape", func(ctx context.Context) error { return a.engine.RemoteMkdir(ctx, base, name) }, "Mapa stvorena: "+name)
}

func (a *app) remoteRenameAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.remoteItems) {
		a.setStatus("Za preimenovanje odaberite točno jednu udaljenu stavku.")
		return
	}
	item := a.remoteItems[indices[0]]
	base := a.remoteCurrent
	name, ok := platform.PromptDialog("ByFTP — preimenuj na poslužitelju", "Novi naziv:", item.Name)
	if !ok || strings.TrimSpace(name) == item.Name {
		return
	}
	a.runRemoteMutation("Preimenovanje", func(ctx context.Context) error {
		return a.engine.RemoteRename(ctx, base, item.Name, strings.TrimSpace(name))
	}, "Udaljena stavka preimenovana.")
}

func (a *app) remoteDeleteAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 {
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
		if !platform.ConfirmDialog("ByFTP — brisanje na poslužitelju", "Obrisati odabrane stavke?", message+"\n\nBrisanje na poslužitelju može biti nepovratno.") {
			return
		}
	}
	base := a.remoteCurrent
	a.runRemoteMutationWithTimeout("Brisanje", remoteBatchTimeout(len(items)), func(ctx context.Context) error {
		var errs []error
		for _, item := range items {
			if err := a.engine.RemoteDelete(ctx, base, item.Name, item.IsDirectory); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, "Udaljene stavke obrisane: "+strconv.Itoa(len(items)))
}

func (a *app) remoteChmodAction() {
	indices := selectedIndices(a.remoteList)
	if len(indices) == 0 {
		a.setStatus("Odaberite jednu ili više stavki za promjenu dozvola.")
		return
	}
	if len(indices) > 1000 {
		a.setStatus("Za jednu promjenu dozvola odaberite najviše 1000 stavki.")
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
	base := a.remoteCurrent
	mode, ok := platform.PromptDialog("ByFTP — dozvole", "Dozvole za odabrane stavke, npr. 644 ili 755:", "644")
	if !ok {
		return
	}
	mode = strings.TrimSpace(mode)
	a.runRemoteMutationWithTimeout("Promjena dozvola", remoteBatchTimeout(len(items)), func(ctx context.Context) error {
		var errs []error
		for _, item := range items {
			if err := a.engine.RemoteChmod(ctx, base, item.Name, mode); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}, "Dozvole promijenjene za stavki: "+strconv.Itoa(len(items)))
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
	if !a.connected {
		return
	}
	baseGeneration := a.remoteNavSeq
	a.setStatus(label + "…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		err := operation(ctx)
		a.dispatch(func() {
			if err != nil {
				if a.suppressExpectedDisconnectError(err) {
					return
				}
				platform.ErrorDialog("ByFTP — poslužitelj", label+" nije uspjelo", usererror.Message(err, "Radnju na poslužitelju nije moguće dovršiti."))
				a.setStatus(usererror.Message(err, label+" nije uspjelo."))
				a.checkConnectionAfterError()
				return
			}
			a.setStatus(success)
			if baseGeneration == a.remoteNavSeq {
				a.refreshRemote(a.remoteCurrent)
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
	a.setStatus("Dodavanje odabranih stavki u red prijenosa…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		queuedFiles := 0
		queuedDirs := 0
		var firstErr error
		if len(files) > 0 {
			jobs, err := a.engine.AddTransfers(files)
			if err != nil {
				firstErr = err
			} else {
				queuedFiles += len(jobs)
			}
		}
		if firstErr == nil {
			for _, tree := range trees {
				result, err := a.engine.AddTreeTransfer(ctx, direction, tree.localPath, tree.remotePath)
				if err != nil {
					firstErr = err
					break
				}
				queuedFiles += result.Queued
				queuedDirs += result.Directories
				skipped += result.SkippedSymlinks
			}
		}
		a.dispatch(func() {
			if firstErr != nil {
				platform.ErrorDialog("ByFTP — prijenos", "Nisu dodane sve odabrane stavke", usererror.Message(firstErr, "Dio odabranih stavki nije moguće dodati u prijenos."))
			}
			text := "Dodano u red: " + strconv.Itoa(queuedFiles) + " datoteka"
			if queuedDirs > 0 {
				text += ", " + strconv.Itoa(queuedDirs) + " mapa"
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
	a.setStatus("Priprema prijenosa cijele mape…")
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		result, err := a.engine.AddTreeTransfer(ctx, direction, local, remotePath)
		a.dispatch(func() {
			if err != nil {
				if a.suppressExpectedDisconnectError(err) {
					return
				}
				platform.ErrorDialog("ByFTP — prijenos mape", "Mapa nije dodana u prijenos", usererror.Message(err, "Prijenos mape nije moguće pokrenuti."))
				a.setStatus(usererror.Message(err, "Prijenos mape nije moguće pokrenuti."))
				return
			}
			a.setStatus("Mapa dodana: " + strconv.Itoa(result.Queued) + " datoteka, " + strconv.Itoa(result.Directories) + " mapa.")
			a.refreshTransfers()
		})
	})
}

func (a *app) checkConnectionAfterError() {
	if a.healthCheckRunning || !a.connected {
		return
	}
	a.healthCheckRunning = true
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	a.healthCheckCancel = cancel
	a.goSafe(func() {
		err := a.engine.Probe(ctx)
		cancel()
		if err == nil {
			a.dispatch(func() {
				a.healthCheckRunning = false
				a.healthCheckCancel = nil
			})
			return
		}
		disconnectCtx, disconnectCancel := context.WithTimeout(context.Background(), 10*time.Second)
		_ = a.engine.Disconnect(disconnectCtx)
		disconnectCancel()
		a.dispatch(func() {
			a.healthCheckRunning = false
			a.healthCheckCancel = nil
			if a.remoteNavCancel != nil {
				a.remoteNavCancel()
				a.remoteNavCancel = nil
			}
			a.remoteNavSeq++
			a.setConnectionUI(false)
			clearList(a.remoteList)
			a.remoteItems = nil
			a.setStatus("Veza više nije dostupna. Aktivni prijenosi su zaustavljeni; poslužitelj i korisničko ime ostali su uneseni.")
		})
	})
}
