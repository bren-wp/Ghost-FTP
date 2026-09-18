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

type linuxModalFocusState struct {
	focus int
}

var linuxFileFilters sync.Map
var linuxModalFocus sync.Map

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
	// The master Files workspace gives the ordinary directory list the full pane.
	// Reserve the compact control row only while recursive search or directory
	// comparison is actively replacing the normal list with modal results.
	if u.directoryComparisonActiveLinux() || u.recursiveSearchActive(remote) {
		base.top += 34
		if base.top > base.bottom {
			base.top = base.bottom
		}
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

func (u *linuxDesktop) parkLinuxModalBackgroundFocus() bool {
	if u.focus < 0 || u.focus >= linuxFieldCount {
		return false
	}
	linuxModalFocus.LoadOrStore(u, linuxModalFocusState{focus: u.focus})
	u.focus = linuxFieldCount
	return true
}

func (u *linuxDesktop) isolateLinuxModalBackgroundHitTargets() {
	// Recursive search and directory comparison deliberately replace normal
	// directory rows with modal result snapshots. Editable fields are checked
	// before modal workspace handlers, so their hit targets must be removed after
	// those fields have been painted for the current frame.
	u.layout.protocol = linuxRect{}
	u.layout.host = linuxRect{}
	u.layout.port = linuxRect{}
	u.layout.user = linuxRect{}
	u.layout.password = linuxRect{}
	u.layout.key = linuxRect{}
	u.layout.passphrase = linuxRect{}
	u.layout.localPath = linuxRect{}
	u.layout.remotePath = linuxRect{}
}

func (u *linuxDesktop) restoreLinuxModalBackgroundFocus() bool {
	value, ok := linuxModalFocus.LoadAndDelete(u)
	if !ok || u.focus != linuxFieldCount {
		return false
	}
	state, ok := value.(linuxModalFocusState)
	if !ok || state.focus < 0 || state.focus >= linuxFieldCount {
		return false
	}
	u.focus = state.focus
	return true
}

func (u *linuxDesktop) redrawLinuxEditableFields() error {
	// Modal focus is parked/restored after the ordinary field paint has already
	// occurred. Repaint only editable fields immediately so the visible focus
	// border always matches the keyboard owner in the same X11 frame.
	fields := []struct {
		index int
		hint  string
	}{
		{linuxFieldProtocol, u.tr("terminal.protocol")},
		{linuxFieldHost, u.tr("terminal.server")},
		{linuxFieldPort, u.tr("terminal.port")},
		{linuxFieldUser, u.tr("terminal.username")},
		{linuxFieldPassword, u.tr("terminal.password")},
		{linuxFieldKey, u.tr("cue.private_key")},
		{linuxFieldPassphrase, u.tr("cue.passphrase")},
		{linuxFieldLocalPath, u.tr("column.local")},
		{linuxFieldRemotePath, u.tr("column.remote")},
	}
	for _, field := range fields {
		if err := u.drawField(field.index, field.hint); err != nil {
			return err
		}
	}
	return nil
}

func (u *linuxDesktop) renderFileFilterControls() error {
	// Empty linuxUIResult notifications are used only to wake the established UI
	// loop after recursive-search or comparison work. Reconcile on the UI
	// goroutine before the modal controls are painted.
	u.reconcileRecursiveSearchState()
	u.reconcileDirectoryComparisonLinux()
	comparisonActive := u.directoryComparisonActiveLinux()
	searchActive := u.recursiveSearchActive(false) || u.recursiveSearchActive(true)
	if comparisonActive || searchActive {
		if u.parkLinuxModalBackgroundFocus() {
			if err := u.redrawLinuxEditableFields(); err != nil {
				return err
			}
		}
		u.isolateLinuxModalBackgroundHitTargets()
	} else if u.restoreLinuxModalBackgroundFocus() {
		if err := u.redrawLinuxEditableFields(); err != nil {
			return err
		}
	}
	if comparisonActive {
		return u.renderDirectoryComparisonControlsLinux()
	}
	if searchActive {
		for _, remote := range []bool{false, true} {
			if u.recursiveSearchActive(remote) {
				return u.renderRecursiveSearchControls(remote)
			}
		}
	}
	// Filter, sort, recursive-search and comparison entry points are available
	// from the master More menu. Keep the normal Files surface visually 1:1
	// with the supplied reference instead of exposing permanent technical rows.
	return nil
}

func (u *linuxDesktop) handleFileFilterMouse(x, y int) bool {
	if u.directoryComparisonActiveLinux() {
		return u.handleDirectoryComparisonMouseLinux(x, y)
	}
	if u.recursiveSearchActive(false) || u.recursiveSearchActive(true) {
		return u.handleRecursiveSearchMouse(x, y)
	}
	return false
}

func (u *linuxDesktop) openFileFilterPrompt(remote bool) {
	if u.directoryComparisonActiveLinux() {
		return
	}
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
			u.remoteItems = u.sortLinuxFileItems(true, items)
		} else {
			u.localItems = u.sortLinuxFileItems(false, items)
		}
		return
	}
	source := append([]model.Item(nil), items...)
	if remote {
		state.remoteAll = source
		u.remoteItems = u.sortLinuxFileItems(true, itemlist.Filter(state.remoteAll, state.remoteQuery))
		return
	}
	state.localAll = source
	u.localItems = u.sortLinuxFileItems(false, itemlist.Filter(state.localAll, state.localQuery))
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
	if u.directoryComparisonActiveLinux() {
		return
	}
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
		u.remoteItems = u.sortLinuxFileItems(true, itemlist.Filter(state.remoteAll, query))
		u.selectedRemote = restoreLinuxSelection(u.remoteItems, selectedName)
		u.setStatus(fmt.Sprintf("%s · %d/%d", fileFilterWordsForLanguage(u.language).Cue, len(u.remoteItems), len(state.remoteAll)))
		return
	}
	selectedName := selectedLinuxItemName(u.localItems, u.selectedLocal)
	state.localQuery = query
	u.localItems = u.sortLinuxFileItems(false, itemlist.Filter(state.localAll, query))
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
