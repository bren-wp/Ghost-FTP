//go:build linux

package desktop

import (
	"fmt"
	"strings"
	"sync"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
)

type linuxFileFilterState struct {
	localAll    []model.Item
	remoteAll   []model.Item
	localQuery  string
	remoteQuery string
}

var linuxFileFilters sync.Map

func (u *linuxDesktop) fileFilterState() *linuxFileFilterState {
	if u == nil {
		return nil
	}
	if value, ok := linuxFileFilters.Load(u); ok {
		if state, ok := value.(*linuxFileFilterState); ok {
			return state
		}
	}
	state := &linuxFileFilterState{}
	actual, _ := linuxFileFilters.LoadOrStore(u, state)
	if existing, ok := actual.(*linuxFileFilterState); ok {
		return existing
	}
	return state
}

func (u *linuxDesktop) fileFilterRowRect(remote bool) linuxRect {
	base := u.layout.localList
	if remote {
		base = u.layout.remoteList
	}
	return linuxRectWH(base.left, base.top, base.right-base.left, 28)
}

func (u *linuxDesktop) fileFilterControlRect(remote bool) linuxRect {
	row := u.fileFilterRowRect(remote)
	gap := 8
	width := (row.right - row.left - gap) / 2
	return linuxRectWH(row.left, row.top, width, row.bottom-row.top)
}

func (u *linuxDesktop) fileFilterListRect(remote bool) linuxRect {
	base := u.layout.localList
	if remote {
		base = u.layout.remoteList
	}
	base.top += 34
	if base.top > base.bottom {
		base.top = base.bottom
	}
	return base
}

func (u *linuxDesktop) fileFilterLabel(remote bool) string {
	state := u.fileFilterState()
	words := fileFilterWordsForLanguage(u.language)
	if state == nil {
		return words.Cue
	}
	query := state.localQuery
	visible, total := len(u.localItems), len(state.localAll)
	if remote {
		query = state.remoteQuery
		visible, total = len(u.remoteItems), len(state.remoteAll)
	}
	if strings.TrimSpace(query) == "" {
		return words.Cue
	}
	return fmt.Sprintf("%s  ·  %d/%d", words.Cue, visible, total)
}

func (u *linuxDesktop) renderFileFilterControls() error {
	// Empty linuxUIResult notifications are used only to wake the established UI
	// loop after recursive-search batches. Reconcile here, on the UI goroutine,
	// before normal controls are drawn so search mode remains modal and normal
	// row-indexed mutation controls never become enabled between batches.
	u.reconcileRecursiveSearchState()
	for _, remote := range []bool{false, true} {
		if u.recursiveSearchActive(remote) {
			if err := u.renderRecursiveSearchControls(remote); err != nil {
				return err
			}
			continue
		}
		filterEnabled := !u.busy && (!remote || u.connected)
		if err := u.drawButton(u.fileFilterControlRect(remote), u.fileFilterLabel(remote), filterEnabled, false); err != nil {
			return err
		}
		if err := u.renderRecursiveSearchButton(remote); err != nil {
			return err
		}
	}
	return nil
}

func (u *linuxDesktop) handleFileFilterMouse(x, y int) bool {
	if u.handleRecursiveSearchMouse(x, y) {
		return true
	}
	if u.fileFilterControlRect(false).contains(x, y) {
		if !u.busy {
			u.openFileFilterPrompt(false)
		}
		return true
	}
	if u.fileFilterControlRect(true).contains(x, y) {
		if u.connected && !u.busy {
			u.openFileFilterPrompt(true)
		}
		return true
	}
	return false
}

func (u *linuxDesktop) openFileFilterPrompt(remote bool) {
	state := u.fileFilterState()
	if state == nil {
		return
	}
	words := fileFilterWordsForLanguage(u.language)
	kind := linuxPromptLocalFilter
	query := state.localQuery
	section := u.tr("section.local")
	if remote {
		kind = linuxPromptRemoteFilter
		query = state.remoteQuery
		section = u.tr("section.remote")
	}
	u.openPrompt(kind, section+" · "+words.Cue, query)
}

func (u *linuxDesktop) acceptLinuxFileFilterSnapshot(remote bool, items []model.Item) {
	state := u.fileFilterState()
	if state == nil {
		if remote {
			u.remoteItems = append([]model.Item(nil), items...)
		} else {
			u.localItems = append([]model.Item(nil), items...)
		}
		return
	}
	source := append([]model.Item(nil), items...)
	if remote {
		state.remoteAll = source
		u.remoteItems = itemlist.Filter(state.remoteAll, state.remoteQuery)
		return
	}
	state.localAll = source
	u.localItems = itemlist.Filter(state.localAll, state.localQuery)
}

func selectedLinuxItemName(items []model.Item, selected int) string {
	if selected < 0 || selected >= len(items) {
		return ""
	}
	return items[selected].Name
}

func restoreLinuxSelection(items []model.Item, name string) int {
	if name == "" {
		return -1
	}
	for index := range items {
		if items[index].Name == name {
			return index
		}
	}
	return -1
}

func (u *linuxDesktop) applyLinuxFileFilter(remote bool, query string) {
	state := u.fileFilterState()
	if state == nil {
		return
	}
	query = strings.TrimSpace(query)
	if remote {
		if !u.connected {
			return
		}
		selectedName := selectedLinuxItemName(u.remoteItems, u.selectedRemote)
		state.remoteQuery = query
		u.remoteItems = itemlist.Filter(state.remoteAll, query)
		u.selectedRemote = restoreLinuxSelection(u.remoteItems, selectedName)
		u.setStatus(fmt.Sprintf("%s · %d/%d", fileFilterWordsForLanguage(u.language).Cue, len(u.remoteItems), len(state.remoteAll)))
		return
	}
	selectedName := selectedLinuxItemName(u.localItems, u.selectedLocal)
	state.localQuery = query
	u.localItems = itemlist.Filter(state.localAll, query)
	u.selectedLocal = restoreLinuxSelection(u.localItems, selectedName)
	u.setStatus(fmt.Sprintf("%s · %d/%d", fileFilterWordsForLanguage(u.language).Cue, len(u.localItems), len(state.localAll)))
}

func (u *linuxDesktop) clearLinuxRemoteFilterSource() {
	state := u.fileFilterState()
	if state != nil {
		state.remoteAll = nil
	}
	u.remoteItems = nil
	u.selectedRemote = -1
}
