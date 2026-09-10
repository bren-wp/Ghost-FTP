//go:build linux

package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

type linuxRecursiveSearchState struct {
	mu sync.Mutex

	seq      uint64
	active   bool
	running  bool
	remote   bool
	query    string
	results  []api.SearchResult
	selected int
	cancel   context.CancelFunc
	status   string
	dirty    bool

	restoreSelected string
	navPending      bool
	navRemote       bool
	navParent       string
	navName         string
}

var linuxRecursiveSearchStates sync.Map

func linuxRecursiveSearchStateFor(u *linuxDesktop) *linuxRecursiveSearchState {
	if value, ok := linuxRecursiveSearchStates.Load(u); ok {
		return value.(*linuxRecursiveSearchState)
	}
	state := &linuxRecursiveSearchState{selected: -1}
	actual, _ := linuxRecursiveSearchStates.LoadOrStore(u, state)
	return actual.(*linuxRecursiveSearchState)
}

func (u *linuxDesktop) recursiveSearchButtonRect(remote bool) linuxRect {
	row := u.fileFilterRowRect(remote)
	gap := 8
	width := (row.right - row.left - gap) / 2
	return linuxRectWH(row.left+width+gap, row.top, width, row.bottom-row.top)
}

func (u *linuxDesktop) recursiveSearchActive(remote bool) bool {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.active && state.remote == remote
}

func (u *linuxDesktop) openRecursiveSearchPrompt(remote bool) {
	if u.busy || u.recursiveSearchActive(false) || u.recursiveSearchActive(true) {
		return
	}
	if remote && !u.connected {
		return
	}
	words := recursiveSearchWordsForLanguage(u.language)
	disclosure := words.LocalDisclosure
	kind := linuxPromptLocalRecursiveSearch
	section := u.tr("section.local")
	if remote {
		disclosure = words.RemoteDisclosure
		kind = linuxPromptRemoteRecursiveSearch
		section = u.tr("section.remote")
	}
	u.setStatus(disclosure)
	u.openPrompt(kind, section+" · "+words.Search, "")
}

func (u *linuxDesktop) startRecursiveSearch(remote bool, query string) {
	query = strings.TrimSpace(query)
	if query == "" || u.busy || (remote && !u.connected) {
		return
	}
	state := linuxRecursiveSearchStateFor(u)
	words := recursiveSearchWordsForLanguage(u.language)
	ctx, cancel := context.WithCancel(context.Background())

	restoreSelected := selectedLinuxItemName(u.localItems, u.selectedLocal)
	root := u.localCurrent
	if remote {
		restoreSelected = selectedLinuxItemName(u.remoteItems, u.selectedRemote)
		root = u.remoteCurrent
	}

	state.mu.Lock()
	if state.cancel != nil {
		state.cancel()
	}
	state.seq++
	seq := state.seq
	state.active = true
	state.running = true
	state.remote = remote
	state.query = query
	state.results = nil
	state.selected = -1
	state.cancel = cancel
	state.status = words.Searching
	state.dirty = true
	state.restoreSelected = restoreSelected
	state.navPending = false
	state.navParent = ""
	state.navName = ""
	state.mu.Unlock()

	u.busy = true
	if remote {
		u.selectedRemote = -1
		u.setStatus(words.RemoteDisclosure + " " + words.Searching)
	} else {
		u.selectedLocal = -1
		u.setStatus(words.LocalDisclosure + " " + words.Searching)
	}

	emit := func(batch []api.SearchResult) error {
		state.mu.Lock()
		if state.seq != seq || !state.active {
			state.mu.Unlock()
			return context.Canceled
		}
		state.results = append(state.results, batch...)
		if state.selected < 0 && len(state.results) > 0 {
			state.selected = 0
		}
		state.status = fmt.Sprintf("%s · %d", words.Searching, len(state.results))
		state.dirty = true
		state.mu.Unlock()
		u.resultCh <- linuxUIResult{action: linuxActionNone}
		return nil
	}

	go func() {
		var (
			stats api.SearchStats
			err   error
		)
		if remote {
			stats, err = u.engine.SearchRemoteRecursive(ctx, root, api.SearchOptions{Query: query}, emit)
		} else {
			stats, _, err = u.engine.SearchLocalRecursive(ctx, root, api.SearchOptions{Query: query}, emit)
		}

		state.mu.Lock()
		if state.seq == seq {
			state.running = false
			state.cancel = nil
			switch {
			case err != nil && context.Cause(ctx) != nil:
				state.status = words.Cancelled
			case err != nil:
				state.status = usererror.MessageFor(u.language, err, words.Cancelled)
			case stats.Results == 0:
				state.status = words.NoResults
			case stats.StopReason != "":
				state.status = fmt.Sprintf("%s · %d · %s", words.Done, stats.Results, stats.StopReason)
			default:
				state.status = fmt.Sprintf("%s · %d", words.Done, stats.Results)
			}
			state.dirty = true
		}
		state.mu.Unlock()
		u.resultCh <- linuxUIResult{action: linuxActionNone}
	}()
}

func (u *linuxDesktop) reconcileRecursiveSearchState() {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	active := state.active
	remote := state.remote
	results := append([]api.SearchResult(nil), state.results...)
	selected := state.selected
	status := state.status
	dirty := state.dirty
	if dirty {
		state.dirty = false
	}
	navPending := state.navPending
	navRemote := state.navRemote
	navParent := state.navParent
	navName := state.navName
	state.mu.Unlock()

	if active {
		// Search mode is intentionally modal. This keeps every normal row-indexed
		// mutation/transfer action disabled while the visible rows are snapshots
		// produced by a recursive scan rather than the authoritative directory list.
		u.busy = true
		visible := make([]model.Item, 0, len(results))
		for _, result := range results {
			item := result.Item
			item.Name = result.Path
			visible = append(visible, item)
		}
		if remote {
			u.remoteItems = visible
			u.selectedRemote = selected
		} else {
			u.localItems = visible
			u.selectedLocal = selected
		}
		if dirty && status != "" {
			u.setStatus(status)
		}
		return
	}

	if !navPending || u.busy {
		return
	}
	matchedParent := false
	if navRemote {
		matchedParent = u.remoteCurrent == navParent
	} else {
		matchedParent = filepath.Clean(u.localCurrent) == filepath.Clean(navParent)
	}
	state.mu.Lock()
	state.navPending = false
	state.navParent = ""
	state.navName = ""
	state.mu.Unlock()
	if !matchedParent {
		return
	}
	if navRemote {
		u.selectedRemote = restoreLinuxSelection(u.remoteItems, navName)
	} else {
		u.selectedLocal = restoreLinuxSelection(u.localItems, navName)
	}
}

func (u *linuxDesktop) renderRecursiveSearchControls(remote bool) error {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	if !state.active || state.remote != remote {
		state.mu.Unlock()
		return nil
	}
	running := state.running
	count := len(state.results)
	state.mu.Unlock()

	words := recursiveSearchWordsForLanguage(u.language)
	leftLabel := words.Close
	if running {
		leftLabel = words.Cancel
	}
	if err := u.drawButton(u.fileFilterControlRect(remote), leftLabel, true, false); err != nil {
		return err
	}
	navigate := fmt.Sprintf("%s · %d", words.Navigate, count)
	return u.drawButton(u.recursiveSearchButtonRect(remote), navigate, !running && count > 0, true)
}

func (u *linuxDesktop) renderRecursiveSearchButton(remote bool) error {
	words := recursiveSearchWordsForLanguage(u.language)
	enabled := !u.busy && (!remote || u.connected)
	return u.drawButton(u.recursiveSearchButtonRect(remote), words.Search, enabled, false)
}

func (u *linuxDesktop) handleRecursiveSearchMouse(x, y int) bool {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	active := state.active
	remote := state.remote
	running := state.running
	count := len(state.results)
	state.mu.Unlock()
	if !active {
		for _, paneRemote := range []bool{false, true} {
			if u.recursiveSearchButtonRect(paneRemote).contains(x, y) {
				if !u.busy && (!paneRemote || u.connected) {
					u.openRecursiveSearchPrompt(paneRemote)
				}
				return true
			}
		}
		return false
	}

	if u.fileFilterControlRect(remote).contains(x, y) {
		if running {
			u.cancelRecursiveSearch()
		} else {
			u.closeRecursiveSearch()
		}
		return true
	}
	if u.recursiveSearchButtonRect(remote).contains(x, y) {
		if !running && count > 0 {
			u.navigateRecursiveSearchResult()
		}
		return true
	}
	list := u.fileFilterListRect(remote)
	if list.contains(x, y) {
		index := u.selectRow(list, y, count)
		state.mu.Lock()
		if state.active && state.remote == remote {
			state.selected = index
		}
		state.mu.Unlock()
		if remote {
			u.selectedRemote = index
		} else {
			u.selectedLocal = index
		}
		return true
	}
	return false
}

func (u *linuxDesktop) cancelRecursiveSearch() {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	if !state.active || !state.running {
		state.mu.Unlock()
		return
	}
	cancel := state.cancel
	state.status = recursiveSearchWordsForLanguage(u.language).Cancelled
	state.dirty = true
	state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (u *linuxDesktop) restoreRecursiveSearchPane(remote bool, selectedName string) {
	filter := u.fileFilterState()
	if filter == nil {
		return
	}
	if remote {
		u.remoteItems = itemlist.Filter(filter.remoteAll, filter.remoteQuery)
		u.selectedRemote = restoreLinuxSelection(u.remoteItems, selectedName)
		return
	}
	u.localItems = itemlist.Filter(filter.localAll, filter.localQuery)
	u.selectedLocal = restoreLinuxSelection(u.localItems, selectedName)
}

func (u *linuxDesktop) closeRecursiveSearch() {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	if !state.active {
		state.mu.Unlock()
		return
	}
	remote := state.remote
	restoreSelected := state.restoreSelected
	cancel := state.cancel
	state.seq++
	state.active = false
	state.running = false
	state.cancel = nil
	state.results = nil
	state.selected = -1
	state.dirty = false
	state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	u.busy = false
	u.restoreRecursiveSearchPane(remote, restoreSelected)
	u.setStatus(recursiveSearchWordsForLanguage(u.language).Close)
}

func (u *linuxDesktop) navigateRecursiveSearchResult() {
	state := linuxRecursiveSearchStateFor(u)
	state.mu.Lock()
	if !state.active || state.running || state.selected < 0 || state.selected >= len(state.results) {
		state.mu.Unlock()
		return
	}
	result := state.results[state.selected]
	remote := state.remote
	cancel := state.cancel
	state.seq++
	state.active = false
	state.running = false
	state.cancel = nil
	state.results = nil
	state.selected = -1
	state.navPending = true
	state.navRemote = remote
	state.navParent = result.Parent
	state.navName = result.Item.Name
	state.mu.Unlock()
	if cancel != nil {
		cancel()
	}

	// Clear only the target pane's current-folder filter so the freshly re-listed
	// result cannot be hidden by an older view filter. The recursive result itself
	// is never copied into the authoritative directory snapshot.
	if filter := u.fileFilterState(); filter != nil {
		if remote {
			filter.remoteQuery = ""
		} else {
			filter.localQuery = ""
		}
	}
	u.busy = false
	u.setStatus(recursiveSearchWordsForLanguage(u.language).Navigate + "…")
	if remote {
		if u.connected {
			u.refreshRemote(result.Parent)
		}
		return
	}
	u.refreshLocal(result.Parent)
}
