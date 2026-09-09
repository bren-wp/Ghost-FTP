//go:build windows

package desktop

import (
	"context"
	"errors"
	"path"
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const idRemoteEdit = 309

var remoteEditButtons sync.Map

func storeRemoteEditButton(a *app, hwnd uintptr) {
	if a == nil || hwnd == 0 {
		return
	}
	remoteEditButtons.Store(a, hwnd)
}

func remoteEditButton(a *app) uintptr {
	if a == nil {
		return 0
	}
	if value, ok := remoteEditButtons.Load(a); ok {
		return value.(uintptr)
	}
	return 0
}

func clearRemoteEditButton(a *app) {
	if a != nil {
		remoteEditButtons.Delete(a)
	}
}

func (a *app) remoteEditSelectionReady() bool {
	if a == nil || !a.connected || a.connectionBusy {
		return false
	}
	indices := selectedIndices(a.remoteList)
	if len(indices) != 1 || indices[0] < 0 || indices[0] >= len(a.remoteItems) {
		return false
	}
	item := a.remoteItems[indices[0]]
	return !item.IsDirectory && !item.IsSymlink && item.Size >= 0 && item.Size <= api.MaxRemoteEditBytes
}

func (a *app) remoteEditAction() {
	if !a.remoteEditSelectionReady() {
		a.setStatus(remoteEditWords(a.languageCode()).Unsupported)
		return
	}
	index := selectedIndices(a.remoteList)[0]
	item := a.remoteItems[index]
	if err := security.ValidateRemoteName(item.Name); err != nil {
		a.setStatus(a.userMessage(err, "error.invalid_name"))
		return
	}
	remotePath := path.Join(a.remoteCurrent, item.Name)
	generation := a.connectionGeneration
	words := remoteEditWords(a.languageCode())
	a.setStatus(words.Opening)
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		doc, err := a.engine.RemoteEditOpen(ctx, remotePath)
		var buffer remoteEditBuffer
		if err == nil {
			buffer, err = newRemoteEditBuffer(doc.Text)
		}
		a.dispatch(func() {
			if generation != a.connectionGeneration || !a.connected {
				a.setStatus(words.ConnectionChanged)
				return
			}
			if err != nil {
				message := a.userMessage(err, "error.generic")
				if errors.Is(err, errRemoteEditMixedNewlines) {
					message = words.MixedNewlines
				}
				platform.ErrorDialog("Ghost FTP — "+words.Title, words.Edit, message)
				a.setStatus(message)
				return
			}
			a.setStatus(words.Loaded)
			a.showRemoteTextEditor(doc, buffer, buffer.Text, generation)
		})
	})
}

func (a *app) showRemoteTextEditor(doc api.RemoteEditDocument, buffer remoteEditBuffer, text string, generation uint64) {
	if generation != a.connectionGeneration || !a.connected {
		a.setStatus(remoteEditWords(a.languageCode()).ConnectionChanged)
		return
	}
	words := remoteEditWords(a.languageCode())
	result := platform.TextEditorDialog(platform.TextEditorDialogConfig{
		Title:       "Ghost FTP — " + words.Title,
		Path:        doc.Path,
		Text:        normalizeRemoteEditorText(text),
		Hint:        words.Hint,
		SaveLabel:   words.Save,
		ReloadLabel: words.Reload,
		CloseLabel:  words.Close,
	})
	result.Text = normalizeRemoteEditorText(result.Text)
	if generation != a.connectionGeneration || !a.connected {
		a.setStatus(words.ConnectionChanged)
		return
	}

	dirty := result.Text != buffer.Text
	switch result.Action {
	case "save":
		a.saveRemoteTextEditor(doc, buffer, result.Text, generation)
	case "reload":
		if dirty && !platform.ConfirmDialog("Ghost FTP — "+words.Title, words.Reload+"?", words.DiscardReload) {
			a.showRemoteTextEditor(doc, buffer, result.Text, generation)
			return
		}
		a.reloadRemoteTextEditor(doc.Path, generation)
	default:
		if dirty && !platform.ConfirmDialog("Ghost FTP — "+words.Title, words.Close+"?", words.DiscardClose) {
			a.showRemoteTextEditor(doc, buffer, result.Text, generation)
		}
	}
}

func (a *app) saveRemoteTextEditor(doc api.RemoteEditDocument, buffer remoteEditBuffer, text string, generation uint64) {
	words := remoteEditWords(a.languageCode())
	encoded := buffer.Encode(text)
	a.setStatus(words.Saving)
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		saved, err := a.engine.RemoteEditSave(ctx, doc.Path, doc.Revision, encoded)
		var next remoteEditBuffer
		if err == nil {
			next, err = newRemoteEditBuffer(saved.Text)
		}
		a.dispatch(func() {
			if generation != a.connectionGeneration || !a.connected {
				a.setStatus(words.ConnectionChanged)
				return
			}
			if err != nil {
				message := a.userMessage(err, "error.generic")
				if errors.Is(err, api.ErrRemoteEditConflict) {
					message = words.Conflict
				} else if errors.Is(err, errRemoteEditMixedNewlines) {
					message = words.MixedNewlines
				}
				platform.ErrorDialog("Ghost FTP — "+words.Title, words.Save, message)
				a.setStatus(message)
				a.showRemoteTextEditor(doc, buffer, text, generation)
				return
			}
			a.setStatus(words.Saved)
			a.showRemoteTextEditor(saved, next, next.Text, generation)
		})
	})
}

func (a *app) reloadRemoteTextEditor(remotePath string, generation uint64) {
	words := remoteEditWords(a.languageCode())
	a.setStatus(words.Reloading)
	a.goSafe(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		doc, err := a.engine.RemoteEditOpen(ctx, remotePath)
		var buffer remoteEditBuffer
		if err == nil {
			buffer, err = newRemoteEditBuffer(doc.Text)
		}
		a.dispatch(func() {
			if generation != a.connectionGeneration || !a.connected {
				a.setStatus(words.ConnectionChanged)
				return
			}
			if err != nil {
				message := a.userMessage(err, "error.generic")
				if errors.Is(err, errRemoteEditMixedNewlines) {
					message = words.MixedNewlines
				}
				platform.ErrorDialog("Ghost FTP — "+words.Title, words.Reload, message)
				a.setStatus(message)
				return
			}
			a.setStatus(words.Reloaded)
			a.showRemoteTextEditor(doc, buffer, buffer.Text, generation)
		})
	})
}
