//go:build windows

package desktop

import (
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/itemlist"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	idLocalFilter  = 209
	idRemoteFilter = 309
)

type windowsFileFilterState struct {
	localButton      uintptr
	remoteButton     uintptr
	localAll         []model.Item
	remoteAll        []model.Item
	localQuery       string
	remoteQuery      string
	remoteGeneration uint64
}

var windowsFileFilters sync.Map

func (a *app) fileFilterState() *windowsFileFilterState {
	if a == nil {
		return nil
	}
	if value, ok := windowsFileFilters.Load(a.hwnd); ok {
		if state, ok := value.(*windowsFileFilterState); ok {
			return state
		}
	}
	state := &windowsFileFilterState{}
	actual, _ := windowsFileFilters.LoadOrStore(a.hwnd, state)
	if existing, ok := actual.(*windowsFileFilterState); ok {
		return existing
	}
	return state
}

func (a *app) ensureFileFilterControls() {
	state := a.fileFilterState()
	if state == nil || a.hwnd == 0 {
		return
	}
	create := func(id int) uintptr {
		hinst, _, _ := getModuleHandleW.Call(0)
		label := fileFilterWordsForLanguage(a.languageCode()).Cue
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr("BUTTON"))),
			uintptr(unsafe.Pointer(wstr(label))),
			uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
			0, 0, 1, 1,
			a.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd == 0 {
			return 0
		}
		if a.font != 0 {
			sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
		}
		applyDarkControl(hwnd, "BUTTON")
		return a.registerButton(hwnd, iconSearch, label, buttonSubtle)
	}
	if state.localButton == 0 {
		state.localButton = create(idLocalFilter)
	}
	if state.remoteButton == 0 {
		state.remoteButton = create(idRemoteFilter)
	}
	a.updateFileFilterControls()
}

func (a *app) updateFileFilterControls() {
	state := a.fileFilterState()
	if state == nil {
		return
	}
	words := fileFilterWordsForLanguage(a.languageCode())
	label := func(query string, visible, total int) string {
		if strings.TrimSpace(query) == "" {
			return words.Cue
		}
		return words.Cue + "  ·  " + strconv.Itoa(visible) + "/" + strconv.Itoa(total)
	}
	if state.localButton != 0 {
		a.setButtonLabel(state.localButton, label(state.localQuery, len(a.localItems), len(state.localAll)))
		setControlEnabled(state.localButton, !a.closing)
	}
	if state.remoteButton != 0 {
		remoteTotal := 0
		if state.remoteGeneration == a.connectionGeneration {
			remoteTotal = len(state.remoteAll)
		}
		a.setButtonLabel(state.remoteButton, label(state.remoteQuery, len(a.remoteItems), remoteTotal))
		setControlEnabled(state.remoteButton, a.connected && !a.connectionBusy && !a.closing)
	}
}

// layoutFileFilterControls inserts one compact filter row between mutation
// actions and each file list. It is derived from the final post-sidebar control
// rectangles and is idempotent: repeated state/layout passes always recompute the
// same top edge rather than progressively shrinking the lists.
func (a *app) layoutFileFilterControls() {
	state := a.fileFilterState()
	if state == nil || state.localButton == 0 || state.remoteButton == 0 {
		return
	}
	layoutPane := func(button, action, list uintptr) {
		actionRect, actionOK := a.sidebarLogicalRect(action)
		listRect, listOK := a.sidebarLogicalRect(list)
		if !actionOK || !listOK {
			return
		}
		filterTop := int(actionRect.Bottom) + 6
		filterHeight := 28
		listTop := filterTop + filterHeight + 6
		bottom := int(listRect.Bottom)
		if bottom-listTop < 72 {
			return
		}
		left := int(listRect.Left)
		width := int(listRect.Right - listRect.Left)
		a.move(button, left, filterTop, width, filterHeight)
		a.move(list, left, listTop, width, bottom-listTop)
	}
	layoutPane(state.localButton, a.localDelete, a.localList)
	layoutPane(state.remoteButton, a.remoteChmod, a.remoteList)
}

// acceptFileFilterSnapshot records an authoritative directory result and returns
// the independent visible slice used by row-indexed actions. Sorting may mutate
// the visible slice without ever reordering or losing the source snapshot.
func (a *app) acceptFileFilterSnapshot(list uintptr, items []model.Item) []model.Item {
	state := a.fileFilterState()
	if state == nil {
		return append([]model.Item(nil), items...)
	}
	source := append([]model.Item(nil), items...)
	if list == a.localList {
		state.localAll = source
		visible := itemlist.Filter(state.localAll, state.localQuery)
		a.localItems = visible
		return visible
	}
	if list == a.remoteList {
		state.remoteAll = source
		state.remoteGeneration = a.connectionGeneration
		visible := itemlist.Filter(state.remoteAll, state.remoteQuery)
		a.remoteItems = visible
		return visible
	}
	return source
}

func (a *app) applyFileFilter(list uintptr, query string) {
	state := a.fileFilterState()
	if state == nil {
		return
	}
	query = strings.TrimSpace(query)
	var source []model.Item
	var visible *[]model.Item
	if list == a.localList {
		state.localQuery = query
		source = state.localAll
		visible = &a.localItems
	} else if list == a.remoteList {
		state.remoteQuery = query
		if state.remoteGeneration == a.connectionGeneration {
			source = state.remoteAll
		}
		visible = &a.remoteItems
	} else {
		return
	}

	selected := selectedItemNames(list, *visible)
	*visible = itemlist.Filter(source, query)
	a.sortFileItems(list, *visible)
	a.fillItemList(list, *visible)
	restoreItemSelection(list, *visible, selected)
	a.updateFileFilterControls()
	a.updateActionControls()
}

func (a *app) openFileFilter(list uintptr) {
	state := a.fileFilterState()
	if state == nil {
		return
	}
	if list == a.remoteList && (!a.connected || a.connectionBusy) {
		return
	}
	words := fileFilterWordsForLanguage(a.languageCode())
	query := state.localQuery
	section := a.tr("section.local")
	if list == a.remoteList {
		query = state.remoteQuery
		section = a.tr("section.remote")
	}
	next, ok := platform.PromptDialog("Ghost FTP — "+section, words.Cue, query)
	if !ok {
		return
	}
	a.applyFileFilter(list, next)
}

func (a *app) localFilterAction()  { a.openFileFilter(a.localList) }
func (a *app) remoteFilterAction() { a.openFileFilter(a.remoteList) }
