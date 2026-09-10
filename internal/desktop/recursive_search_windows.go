//go:build windows

package desktop

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	idLocalRecursiveSearch          = 210
	idLocalRecursiveSearchNavigate  = 211
	idLocalRecursiveSearchList      = 212
	idRemoteRecursiveSearch         = 311
	idRemoteRecursiveSearchNavigate = 312
	idRemoteRecursiveSearchList     = 313

	searchLVMSetColumnW = lvmFirst + 96
)

type windowsRecursiveSearchPane struct {
	searchButton    uintptr
	navigateButton  uintptr
	list            uintptr
	active          bool
	running         bool
	seq             uint64
	generation      uint64
	results         []api.SearchResult
	cancel          context.CancelFunc
	restoreSelected map[string]struct{}
}

type windowsRecursiveSearchState struct {
	local  windowsRecursiveSearchPane
	remote windowsRecursiveSearchPane
}

var windowsRecursiveSearchStates sync.Map
var recursiveSearchGetWindowRect = user32.NewProc("GetWindowRect")
var recursiveSearchScreenToClient = user32.NewProc("ScreenToClient")

func (a *app) recursiveSearchState() *windowsRecursiveSearchState {
	if a == nil {
		return nil
	}
	if value, ok := windowsRecursiveSearchStates.Load(a.hwnd); ok {
		return value.(*windowsRecursiveSearchState)
	}
	state := &windowsRecursiveSearchState{}
	actual, _ := windowsRecursiveSearchStates.LoadOrStore(a.hwnd, state)
	return actual.(*windowsRecursiveSearchState)
}

func (a *app) recursiveSearchPane(remote bool) *windowsRecursiveSearchPane {
	state := a.recursiveSearchState()
	if state == nil {
		return nil
	}
	if remote {
		return &state.remote
	}
	return &state.local
}

func (a *app) recursiveSearchPaneActive(remote bool) bool {
	pane := a.recursiveSearchPane(remote)
	return pane != nil && pane.active
}

func (a *app) createRecursiveSearchButton(hinst uintptr, id int, label string, accent bool) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
		0, 0, 100, 28,
		a.hwnd, uintptr(id), hinst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	applyDarkControl(hwnd, "BUTTON")
	variant := buttonSubtle
	if accent {
		variant = buttonAccent
	}
	return a.registerButton(hwnd, iconSearch, label, variant)
}

func (a *app) createRecursiveSearchList(hinst uintptr, id int) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("SysListView32"))),
		0,
		uintptr(wsChild|wsTabStop|wsBorder|wsVScroll|lvsReport|lvsShowSelAlways),
		0, 0, 100, 100,
		a.hwnd, uintptr(id), hinst, 0,
	)
	if hwnd == 0 {
		return 0
	}
	if a.font != 0 {
		sendMessageW.Call(hwnd, wmSetFont, a.font, 1)
	}
	sendMessageW.Call(hwnd, lvmSetExtendedListViewStyle, 0, lvsExFullRowSelect|lvsExDoubleBuffer|lvsExGridLines)
	styleWorkspaceList(hwnd)
	attachSystemImageList(hwnd)
	showWindow.Call(hwnd, workspaceSWHide)
	return hwnd
}

func (a *app) initializeRecursiveSearchListColumns(list uintptr) {
	if list == 0 {
		return
	}
	labels := []string{a.tr("column.name"), a.tr("common.folder"), a.tr("column.size"), a.tr("column.modified")}
	widths := []int{190, 360, 96, 132}
	for index, label := range labels {
		column := lvColumn{
			Mask: lvcfText | lvcfWidth,
			Cx:   int32(a.scale(widths[index])),
			Text: wstr(label),
		}
		sendMessageW.Call(list, lvmInsertColumnW, uintptr(index), uintptr(unsafe.Pointer(&column)))
	}
}

func (a *app) ensureRecursiveSearchControls() {
	state := a.recursiveSearchState()
	if state == nil || a.hwnd == 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	words := recursiveSearchWordsForLanguage(a.languageCode())
	if state.local.searchButton == 0 {
		state.local.searchButton = a.createRecursiveSearchButton(hinst, idLocalRecursiveSearch, words.Search, false)
		state.local.navigateButton = a.createRecursiveSearchButton(hinst, idLocalRecursiveSearchNavigate, words.Navigate, true)
		state.local.list = a.createRecursiveSearchList(hinst, idLocalRecursiveSearchList)
		a.initializeRecursiveSearchListColumns(state.local.list)
	}
	if state.remote.searchButton == 0 {
		state.remote.searchButton = a.createRecursiveSearchButton(hinst, idRemoteRecursiveSearch, words.Search, false)
		state.remote.navigateButton = a.createRecursiveSearchButton(hinst, idRemoteRecursiveSearchNavigate, words.Navigate, true)
		state.remote.list = a.createRecursiveSearchList(hinst, idRemoteRecursiveSearchList)
		a.initializeRecursiveSearchListColumns(state.remote.list)
	}
	for _, pane := range []*windowsRecursiveSearchPane{&state.local, &state.remote} {
		if pane.navigateButton != 0 && !pane.active {
			showWindow.Call(pane.navigateButton, workspaceSWHide)
		}
	}
	a.updateRecursiveSearchListHeaders()
}

func (a *app) updateRecursiveSearchListHeaders() {
	state := a.recursiveSearchState()
	if state == nil {
		return
	}
	labels := []string{a.tr("column.name"), a.tr("common.folder"), a.tr("column.size"), a.tr("column.modified")}
	widths := []int{190, 360, 96, 132}
	for _, list := range []uintptr{state.local.list, state.remote.list} {
		if list == 0 {
			continue
		}
		for index, label := range labels {
			column := lvColumn{
				Mask: lvcfText | lvcfWidth,
				Cx:   int32(a.scale(widths[index])),
				Text: wstr(label),
			}
			sendMessageW.Call(list, searchLVMSetColumnW, uintptr(index), uintptr(unsafe.Pointer(&column)))
		}
	}
}

func (a *app) filterButtonForPane(remote bool) uintptr {
	filter := a.fileFilterState()
	if filter == nil {
		return 0
	}
	if remote {
		return filter.remoteButton
	}
	return filter.localButton
}

func (a *app) normalListForPane(remote bool) uintptr {
	if remote {
		return a.remoteList
	}
	return a.localList
}

func recursiveSearchClientRect(parent, hwnd uintptr) (rect, bool) {
	if parent == 0 || hwnd == 0 {
		return rect{}, false
	}
	var r rect
	if ok, _, _ := recursiveSearchGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return rect{}, false
	}
	topLeft := point{X: r.Left, Y: r.Top}
	bottomRight := point{X: r.Right, Y: r.Bottom}
	if ok, _, _ := recursiveSearchScreenToClient.Call(parent, uintptr(unsafe.Pointer(&topLeft))); ok == 0 {
		return rect{}, false
	}
	if ok, _, _ := recursiveSearchScreenToClient.Call(parent, uintptr(unsafe.Pointer(&bottomRight))); ok == 0 {
		return rect{}, false
	}
	return rect{Left: topLeft.X, Top: topLeft.Y, Right: bottomRight.X, Bottom: bottomRight.Y}, true
}

func (a *app) layoutRecursiveSearchControls() {
	a.ensureRecursiveSearchControls()
	state := a.recursiveSearchState()
	if state == nil {
		return
	}
	for _, remote := range []bool{false, true} {
		pane := a.recursiveSearchPane(remote)
		filterButton := a.filterButtonForPane(remote)
		normalList := a.normalListForPane(remote)
		if pane == nil || filterButton == 0 || normalList == 0 {
			continue
		}
		filterRect, ok := recursiveSearchClientRect(a.hwnd, filterButton)
		if !ok {
			continue
		}
		listRect, ok := recursiveSearchClientRect(a.hwnd, normalList)
		if !ok {
			continue
		}
		gap := a.scale(8)
		width := int(listRect.Right - listRect.Left)
		half := (width - gap) / 2
		y := int(filterRect.Top)
		height := int(filterRect.Bottom - filterRect.Top)
		left := int(listRect.Left)
		if pane.active {
			moveWindow.Call(pane.searchButton, uintptr(left), uintptr(y), uintptr(half), uintptr(height), 1)
			moveWindow.Call(pane.navigateButton, uintptr(left+half+gap), uintptr(y), uintptr(half), uintptr(height), 1)
		} else {
			moveWindow.Call(filterButton, uintptr(left), uintptr(y), uintptr(half), uintptr(height), 1)
			moveWindow.Call(pane.searchButton, uintptr(left+half+gap), uintptr(y), uintptr(half), uintptr(height), 1)
		}
		moveWindow.Call(pane.list, uintptr(listRect.Left), uintptr(listRect.Top), uintptr(listRect.Right-listRect.Left), uintptr(listRect.Bottom-listRect.Top), 1)
	}
	a.updateRecursiveSearchControls()
}

func (a *app) updateRecursiveSearchControls() {
	state := a.recursiveSearchState()
	if state == nil {
		return
	}
	words := recursiveSearchWordsForLanguage(a.languageCode())
	for _, remote := range []bool{false, true} {
		pane := a.recursiveSearchPane(remote)
		if pane == nil || pane.searchButton == 0 {
			continue
		}
		filterButton := a.filterButtonForPane(remote)
		normalList := a.normalListForPane(remote)
		if !pane.active {
			a.setButtonLabel(pane.searchButton, words.Search)
			showControls(true, filterButton, pane.searchButton, normalList)
			showControls(false, pane.navigateButton, pane.list)
			setControlEnabled(pane.searchButton, !a.closing && (!remote || (a.connected && !a.connectionBusy)))
			continue
		}
		label := words.Close
		if pane.running {
			label = words.Cancel
		}
		a.setButtonLabel(pane.searchButton, label)
		a.setButtonLabel(pane.navigateButton, fmt.Sprintf("%s · %d", words.Navigate, len(pane.results)))
		showControls(false, filterButton, normalList)
		showControls(true, pane.searchButton, pane.navigateButton, pane.list)
		setControlEnabled(pane.searchButton, true)
		setControlEnabled(pane.navigateButton, !pane.running && len(pane.results) > 0)
	}
}

func (a *app) renderRecursiveSearchRows(remote bool) {
	pane := a.recursiveSearchPane(remote)
	if pane == nil || pane.list == 0 {
		return
	}
	selected := selectedIndex(pane.list)
	clearList(pane.list)
	for row, result := range pane.results {
		item := result.Item
		insertListRowWithImage(pane.list, row, []string{
			item.Name,
			result.Parent,
			formatSize(item.Size, item.IsDirectory),
			formatTime(item.Modified),
		}, systemIconIndex(item.Name, item.IsDirectory))
	}
	if len(pane.results) == 0 {
		return
	}
	if selected < 0 || selected >= len(pane.results) {
		selected = 0
	}
	setListRowSelected(pane.list, selected, true)
}

func (a *app) openRecursiveSearch(remote bool) {
	if a.closing || a.recursiveSearchPaneActive(remote) {
		return
	}
	if remote && (!a.connected || a.connectionBusy) {
		return
	}
	words := recursiveSearchWordsForLanguage(a.languageCode())
	disclosure := words.LocalDisclosure
	section := a.tr("section.local")
	if remote {
		disclosure = words.RemoteDisclosure
		section = a.tr("section.remote")
	}
	a.setStatus(disclosure)
	query, ok := platform.PromptDialog("Ghost FTP — "+section, disclosure+"\r\n\r\n"+words.Search, "")
	query = strings.TrimSpace(query)
	if !ok || query == "" {
		return
	}
	a.startRecursiveSearch(remote, query)
}

func (a *app) startRecursiveSearch(remote bool, query string) {
	pane := a.recursiveSearchPane(remote)
	if pane == nil {
		return
	}
	if pane.cancel != nil {
		pane.cancel()
	}
	pane.seq++
	seq := pane.seq
	pane.active = true
	pane.running = true
	pane.results = nil
	if remote {
		pane.restoreSelected = selectedItemNames(a.remoteList, a.remoteItems)
		pane.generation = a.connectionGeneration
	} else {
		pane.restoreSelected = selectedItemNames(a.localList, a.localItems)
	}
	ctx, cancel := context.WithCancel(context.Background())
	pane.cancel = cancel
	words := recursiveSearchWordsForLanguage(a.languageCode())
	root := a.localCurrent
	if remote {
		root = a.remoteCurrent
	}
	a.setStatus(words.Searching)
	a.updateActionControls()
	a.layoutRecursiveSearchControls()

	emit := func(batch []api.SearchResult) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		copyBatch := append([]api.SearchResult(nil), batch...)
		a.dispatch(func() {
			current := a.recursiveSearchPane(remote)
			if current == nil || !current.active || current.seq != seq {
				return
			}
			if remote && current.generation != a.connectionGeneration {
				return
			}
			current.results = append(current.results, copyBatch...)
			a.renderRecursiveSearchRows(remote)
			a.setStatus(fmt.Sprintf("%s · %d", words.Searching, len(current.results)))
			a.updateRecursiveSearchControls()
		})
		return nil
	}

	a.goSafe(func() {
		var (
			stats api.SearchStats
			err   error
		)
		if remote {
			stats, err = a.engine.SearchRemoteRecursive(ctx, root, api.SearchOptions{Query: query}, emit)
		} else {
			stats, _, err = a.engine.SearchLocalRecursive(ctx, root, api.SearchOptions{Query: query}, emit)
		}
		a.dispatch(func() {
			current := a.recursiveSearchPane(remote)
			if current == nil || !current.active || current.seq != seq {
				return
			}
			if remote && current.generation != a.connectionGeneration {
				current.running = false
				current.cancel = nil
				a.setStatus(words.Cancelled)
				a.updateRecursiveSearchControls()
				return
			}
			current.running = false
			current.cancel = nil
			switch {
			case err != nil && ctx.Err() != nil:
				a.setStatus(words.Cancelled)
			case err != nil:
				a.setStatus(a.userMessage(err, "error.generic"))
			case stats.Results == 0:
				a.setStatus(words.NoResults)
			case stats.StopReason != "":
				a.setStatus(fmt.Sprintf("%s · %d · %s", words.Done, stats.Results, stats.StopReason))
			default:
				a.setStatus(fmt.Sprintf("%s · %d", words.Done, stats.Results))
			}
			a.updateRecursiveSearchControls()
			a.updateActionControls()
		})
	})
}

func (a *app) cancelRecursiveSearch(remote bool) {
	pane := a.recursiveSearchPane(remote)
	if pane == nil || !pane.active || !pane.running {
		return
	}
	if pane.cancel != nil {
		pane.cancel()
	}
	a.setStatus(recursiveSearchWordsForLanguage(a.languageCode()).Cancelled)
}

func (a *app) closeRecursiveSearch(remote bool) {
	pane := a.recursiveSearchPane(remote)
	if pane == nil || !pane.active {
		return
	}
	if pane.cancel != nil {
		pane.cancel()
	}
	pane.seq++
	pane.active = false
	pane.running = false
	pane.cancel = nil
	pane.results = nil
	selected := pane.restoreSelected
	pane.restoreSelected = nil
	showControls(true, a.normalListForPane(remote), a.filterButtonForPane(remote))
	showControls(false, pane.list, pane.navigateButton)
	if remote {
		restoreItemSelection(a.remoteList, a.remoteItems, selected)
	} else {
		restoreItemSelection(a.localList, a.localItems, selected)
	}
	a.setStatus(recursiveSearchWordsForLanguage(a.languageCode()).Close)
	a.updateActionControls()
}

func (a *app) recursiveSearchCommand(remote bool) {
	pane := a.recursiveSearchPane(remote)
	if pane != nil && pane.active {
		if pane.running {
			a.cancelRecursiveSearch(remote)
		} else {
			a.closeRecursiveSearch(remote)
		}
		return
	}
	a.openRecursiveSearch(remote)
}

func (a *app) navigateRecursiveSearch(remote bool) {
	pane := a.recursiveSearchPane(remote)
	if pane == nil || !pane.active || pane.running || len(pane.results) == 0 {
		return
	}
	index := selectedIndex(pane.list)
	if index < 0 || index >= len(pane.results) {
		return
	}
	result := pane.results[index]
	name := result.Item.Name
	parent := result.Parent
	words := recursiveSearchWordsForLanguage(a.languageCode())

	a.closeRecursiveSearch(remote)
	if filter := a.fileFilterState(); filter != nil {
		if remote {
			filter.remoteQuery = ""
		} else {
			filter.localQuery = ""
		}
	}
	a.setStatus(words.Navigate + "…")
	selected := map[string]struct{}{name: {}}

	if remote {
		if !a.connected || a.connectionBusy {
			return
		}
		if a.remoteNavCancel != nil {
			a.remoteNavCancel()
		}
		a.remoteNavSeq++
		seq := a.remoteNavSeq
		generation := a.connectionGeneration
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		a.remoteNavCancel = cancel
		a.goSafe(func() {
			defer cancel()
			items, err := a.engine.RemoteList(ctx, parent)
			a.dispatch(func() {
				if seq != a.remoteNavSeq || generation != a.connectionGeneration || !a.connected {
					return
				}
				a.remoteNavCancel = nil
				if err != nil {
					a.setStatus(a.userMessage(err, "error.generic"))
					return
				}
				a.remoteCurrent = cleanRemote(parent)
				setText(a.remotePath, a.remoteCurrent)
				fillItems(a.remoteList, items)
				restoreItemSelection(a.remoteList, a.remoteItems, selected)
				a.updateActionControls()
			})
		})
		return
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
		resolved, items, err := a.engine.LocalList(ctx, parent)
		a.dispatch(func() {
			if seq != a.localNavSeq {
				return
			}
			a.localNavCancel = nil
			if err != nil {
				a.setStatus(a.userMessage(err, "error.generic"))
				return
			}
			a.localCurrent = filepath.Clean(resolved)
			setText(a.localPath, a.localCurrent)
			fillItems(a.localList, items)
			restoreItemSelection(a.localList, a.localItems, selected)
			a.updateActionControls()
		})
	})
}
