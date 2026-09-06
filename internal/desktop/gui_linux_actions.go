//go:build linux

package desktop

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	linuxPromptNone = iota
	linuxPromptLocalMkdir
	linuxPromptLocalRename
	linuxPromptRemoteMkdir
	linuxPromptRemoteRename
	linuxPromptRemoteChmod
)

func (u *linuxDesktop) openPrompt(kind int, title, initial string) {
	if u.busy || kind == linuxPromptNone {
		return
	}
	u.promptKind = kind
	u.promptTitle = title
	u.promptValue = initial
}

func (u *linuxDesktop) closePrompt() {
	u.promptKind = linuxPromptNone
	u.promptTitle = ""
	u.promptValue = ""
}

func (u *linuxDesktop) handlePromptKey(sym uint32) bool {
	if u.promptKind == linuxPromptNone {
		return false
	}
	switch sym {
	case x11KeyEscape:
		u.closePrompt()
		u.setStatus("Operation cancelled.")
	case x11KeyReturn:
		u.submitPrompt()
	case x11KeyBackSpace, x11KeyDelete:
		if len(u.promptValue) != 0 {
			u.promptValue = u.promptValue[:len(u.promptValue)-1]
		}
	default:
		if text, ok := linuxKeysymText(sym); ok && len(u.promptValue)+len(text) <= 255 {
			u.promptValue += text
		}
	}
	return true
}

func (u *linuxDesktop) renderPromptOverlay() error {
	if u.promptKind == linuxPromptNone {
		return nil
	}
	width := min(620, u.width-80)
	height := 158
	left := (u.width - width) / 2
	top := (u.height - height) / 2
	panel := linuxRectWH(left, top, width, height)
	if err := u.drawPanel(panel); err != nil {
		return err
	}
	if err := u.x.text(left+20, top+30, linuxTrimForUI(u.promptTitle, 72), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	u.layout.promptInput = linuxRectWH(left+20, top+52, width-40, 32)
	if err := u.x.fillRect(u.layout.promptInput.left, u.layout.promptInput.top, width-40, 32, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(u.layout.promptInput.left, u.layout.promptInput.top, width-40, 32, premiumTheme.Accent); err != nil {
		return err
	}
	shown := u.promptValue
	if shown == "" {
		shown = "Type a value"
	}
	if err := u.x.text(left+29, top+73, linuxTrimForUI(shown, max(16, (width-58)/7)), premiumTheme.Text, premiumTheme.List); err != nil {
		return err
	}
	u.layout.promptOK = linuxRectWH(left+width-224, top+108, 94, 30)
	u.layout.promptCancel = linuxRectWH(left+width-120, top+108, 94, 30)
	if err := u.drawButton(u.layout.promptOK, "Apply", strings.TrimSpace(u.promptValue) != "" && !u.busy, true); err != nil {
		return err
	}
	return u.drawButton(u.layout.promptCancel, "Cancel", !u.busy, false)
}

func (u *linuxDesktop) handlePromptMouse(x, y int) bool {
	if u.promptKind == linuxPromptNone {
		return false
	}
	if u.layout.promptOK.contains(x, y) {
		u.submitPrompt()
		return true
	}
	if u.layout.promptCancel.contains(x, y) {
		u.closePrompt()
		u.setStatus("Operation cancelled.")
		return true
	}
	return true
}

func (u *linuxDesktop) submitPrompt() {
	value := strings.TrimSpace(u.promptValue)
	if value == "" || u.busy {
		return
	}
	kind := u.promptKind
	u.closePrompt()
	switch kind {
	case linuxPromptLocalMkdir:
		if err := u.engine.LocalMkdir(u.localCurrent, value); err != nil {
			u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
			return
		}
		u.refreshLocal(u.localCurrent)
	case linuxPromptLocalRename:
		item, ok := u.selectedLocalItem()
		if !ok {
			u.setStatus("Select a local item first.")
			return
		}
		if err := u.engine.LocalRename(u.localCurrent, item.Name, value); err != nil {
			u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
			return
		}
		u.refreshLocal(u.localCurrent)
	case linuxPromptRemoteMkdir:
		u.remoteMutation("Creating server folder...", func(ctx context.Context) error {
			return u.engine.RemoteMkdir(ctx, u.remoteCurrent, value)
		})
	case linuxPromptRemoteRename:
		item, ok := u.selectedRemoteItem()
		if !ok {
			u.setStatus("Select a server item first.")
			return
		}
		oldName := item.Name
		u.remoteMutation("Renaming server item...", func(ctx context.Context) error {
			return u.engine.RemoteRename(ctx, u.remoteCurrent, oldName, value)
		})
	case linuxPromptRemoteChmod:
		item, ok := u.selectedRemoteItem()
		if !ok {
			u.setStatus("Select a server item first.")
			return
		}
		name := item.Name
		u.remoteMutation("Changing server permissions...", func(ctx context.Context) error {
			return u.engine.RemoteChmod(ctx, u.remoteCurrent, name, value)
		})
	}
}

func (u *linuxDesktop) remoteMutation(status string, mutation func(context.Context) error) {
	if !u.connected || u.busy {
		return
	}
	current := u.remoteCurrent
	u.setStatus(status)
	u.startAction(linuxActionRemoteRefresh, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := mutation(ctx); err != nil {
			return linuxUIResult{err: err}
		}
		items, err := u.engine.RemoteList(ctx, current)
		return linuxUIResult{remoteItems: items, localBase: current, err: err}
	})
}

func (u *linuxDesktop) destructiveConfirmed(key, label string) bool {
	settings, err := u.engine.Settings()
	if err == nil && !settings.ConfirmDelete {
		u.confirmKind = ""
		return true
	}
	now := time.Now()
	if u.confirmKind == key && now.Before(u.confirmUntil) {
		u.confirmKind = ""
		u.confirmUntil = time.Time{}
		return true
	}
	u.confirmKind = key
	u.confirmUntil = now.Add(8 * time.Second)
	u.setStatus("Safety confirmation: click Delete again within 8 seconds to remove " + label + ".")
	return false
}

func (u *linuxDesktop) deleteSelectedLocal() {
	item, ok := u.selectedLocalItem()
	if !ok || u.busy {
		return
	}
	key := "local:" + u.localCurrent + "\x00" + item.Name
	if !u.destructiveConfirmed(key, item.Name) {
		return
	}
	if err := u.engine.LocalDelete(u.localCurrent, item.Name); err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	u.refreshLocal(u.localCurrent)
}

func (u *linuxDesktop) deleteSelectedRemote() {
	item, ok := u.selectedRemoteItem()
	if !ok || u.busy || !u.connected {
		return
	}
	key := "remote:" + u.remoteCurrent + "\x00" + item.Name
	if !u.destructiveConfirmed(key, item.Name) {
		return
	}
	name, isDir := item.Name, item.IsDirectory
	u.remoteMutation("Deleting server item...", func(ctx context.Context) error {
		return u.engine.RemoteDelete(ctx, u.remoteCurrent, name, isDir)
	})
}

func (u *linuxDesktop) openSelectedLocalRename() {
	if item, ok := u.selectedLocalItem(); ok {
		u.openPrompt(linuxPromptLocalRename, "Rename local item", item.Name)
	}
}

func (u *linuxDesktop) openSelectedRemoteRename() {
	if item, ok := u.selectedRemoteItem(); ok {
		u.openPrompt(linuxPromptRemoteRename, "Rename server item", item.Name)
	}
}

func (u *linuxDesktop) openSelectedRemoteChmod() {
	if item, ok := u.selectedRemoteItem(); ok {
		initial := item.Permissions
		if initial == "" || len(initial) > 4 {
			initial = "0644"
		}
		u.openPrompt(linuxPromptRemoteChmod, fmt.Sprintf("Permissions for %s", item.Name), initial)
	}
}
