//go:build windows

package desktop

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/security"
)

const (
	idDirectoryCompare           = 314
	idDirectoryCompareOpenBoth   = 315
	idDirectoryCompareLocalList  = 316
	idDirectoryCompareRemoteList = 317
)

type windowsDirectoryComparisonState struct {
	compareButton  uintptr
	openBothButton uintptr
	localList      uintptr
	remoteList     uintptr
	active         bool
	loading        bool
	seq            uint64
	generation     uint64
	entries        []api.DirectoryComparisonEntry
	cancel         context.CancelFunc
	localBase      string
	remoteBase     string
}

var windowsDirectoryComparisons sync.Map

func (a *app) directoryComparisonState() *windowsDirectoryComparisonState {
	if a == nil {
		return nil
	}
	if value, ok := windowsDirectoryComparisons.Load(a.hwnd); ok {
		return value.(*windowsDirectoryComparisonState)
	}
	state := &windowsDirectoryComparisonState{}
	actual, _ := windowsDirectoryComparisons.LoadOrStore(a.hwnd, state)
	return actual.(*windowsDirectoryComparisonState)
}

func (a *app) directoryComparisonActive() bool {
	state := a.directoryComparisonState()
	return state != nil && state.active
}

func (a *app) createDirectoryComparisonButton(hinst uintptr, id int, label string, accent bool) uintptr {
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr("BUTTON"))),
		uintptr(unsafe.Pointer(wstr(label))),
		uintptr(wsChild|wsVisible|wsTabStop|bsOwnerDraw),
		0, 0, 100, 32,
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
	return a.registerButton(hwnd, iconSync, label, variant)
}

func (a *app) createDirectoryComparisonList(hinst uintptr, id int) uintptr {
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
	for index, spec := range []struct {
		label string
		width int
	}{
		{a.tr("column.name"), 250},
		{a.tr("column.status"), 150},
		{a.tr("column.size"), 105},
		{a.tr("column.modified"), 145},
	} {
		a.insertColumn(hwnd, index, spec.label, spec.width)
	}
	showWindow.Call(hwnd, workspaceSWHide)
	return hwnd
}

func (a *app) ensureDirectoryComparisonControls() {
	state := a.directoryComparisonState()
	if state == nil || a.hwnd == 0 {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	words := directoryCompareWordsForLanguage(a.languageCode())
	if state.compareButton == 0 {
		state.compareButton = a.createDirectoryComparisonButton(hinst, idDirectoryCompare, words.Compare, false)
		state.openBothButton = a.createDirectoryComparisonButton(hinst, idDirectoryCompareOpenBoth, words.OpenBoth, true)
		state.localList = a.createDirectoryComparisonList(hinst, idDirectoryCompareLocalList)
		state.remoteList = a.createDirectoryComparisonList(hinst, idDirectoryCompareRemoteList)
	}
}

func (a *app) comparisonCanStart() bool {
	return a != nil && a.connected && !a.connectionBusy && !a.closing &&
		!a.recursiveSearchPaneActive(false) && !a.recursiveSearchPaneActive(true)
}

func (a *app) layoutDirectoryComparisonControls() {
	a.ensureDirectoryComparisonControls()
	state := a.directoryComparisonState()
	if state == nil || state.compareButton == 0 {
		return
	}
	localRect, localOK := recursiveSearchClientRect(a.hwnd, a.localList)
	remoteRect, remoteOK := recursiveSearchClientRect(a.hwnd, a.remoteList)
	centerRect, centerOK := recursiveSearchClientRect(a.hwnd, a.upload)
	if !localOK || !remoteOK || !centerOK {
		return
	}
	buttonH := a.scale(32)
	gap := a.scale(8)
	moveWindow.Call(state.compareButton, uintptr(centerRect.Left), uintptr(localRect.Top), uintptr(centerRect.Right-centerRect.Left), uintptr(buttonH), 1)
	moveWindow.Call(state.openBothButton, uintptr(centerRect.Left), uintptr(int(localRect.Top)+buttonH+gap), uintptr(centerRect.Right-centerRect.Left), uintptr(buttonH), 1)
	moveWindow.Call(state.localList, uintptr(localRect.Left), uintptr(localRect.Top), uintptr(localRect.Right-localRect.Left), uintptr(localRect.Bottom-localRect.Top), 1)
	moveWindow.Call(state.remoteList, uintptr(remoteRect.Left), uintptr(remoteRect.Top), uintptr(remoteRect.Right-remoteRect.Left), uintptr(remoteRect.Bottom-remoteRect.Top), 1)
	a.updateDirectoryComparisonControls()
}

func (a *app) comparisonAuxiliaryControls(remote bool) []uintptr {
	filter := a.filterButtonForPane(remote)
	search := a.recursiveSearchPane(remote)
	if search == nil {
		return []uintptr{filter}
	}
	return []uintptr{filter, search.searchButton, search.navigateButton, search.list}
}

func (a *app) updateDirectoryComparisonControls() {
	state := a.directoryComparisonState()
	if state == nil || state.compareButton == 0 {
		return
	}
	words := directoryCompareWordsForLanguage(a.languageCode())
	if !state.active {
		a.setButtonLabel(state.compareButton, words.Compare)
		showControls(true, state.compareButton, a.localList, a.remoteList, a.upload, a.download)
		showControls(false, state.openBothButton, state.localList, state.remoteList)
		setControlEnabled(state.compareButton, a.comparisonCanStart())
		return
	}

	a.setButtonLabel(state.compareButton, words.Close)
	a.setButtonLabel(state.openBothButton, words.OpenBoth)
	showControls(true, state.compareButton, state.openBothButton, state.localList, state.remoteList)
	showControls(false, a.localList, a.remoteList, a.upload, a.download)
	showControls(false, a.comparisonAuxiliaryControls(false)...)
	showControls(false, a.comparisonAuxiliaryControls(true)...)
	setControlEnabled(state.compareButton, true)
	setControlEnabled(state.openBothButton, !state.loading && len(state.entries) > 0)
}

func (a *app) comparisonStatusLabel(status api.DirectoryComparisonStatus) string {
	words := directoryCompareWordsForLanguage(a.languageCode())
	switch status {
	case api.DirectorySame:
		return words.Same
	case api.DirectoryLocalOnly:
		return words.LocalOnly
	case api.DirectoryRemoteOnly:
		return words.RemoteOnly
	case api.DirectoryNewerLocal:
		return words.NewerLocal
	case api.DirectoryNewerRemote:
		return words.NewerRemote
	case api.DirectoryConflict:
		return words.Conflict
	default:
		return words.Unknown
	}
}

func comparisonSideValues(entry api.DirectoryComparisonEntry, remote bool) (string, string) {
	item := entry.Local
	has := entry.HasLocal
	if remote {
		item = entry.Remote
		has = entry.HasRemote
	}
	if !has {
		return "", ""
	}
	return formatSize(item.Size, item.IsDirectory), formatTime(item.Modified)
}

func (a *app) renderDirectoryComparisonRows() {
	state := a.directoryComparisonState()
	if state == nil {
		return
	}
	clearList(state.localList)
	clearList(state.remoteList)
	firstSync := -1
	for row, entry := range state.entries {
		status := a.comparisonStatusLabel(entry.Status)
		localSize, localTime := comparisonSideValues(entry, false)
		remoteSize, remoteTime := comparisonSideValues(entry, true)
		localImage := int32(-1)
		if entry.HasLocal {
			localImage = systemIconIndex(entry.Local.Name, entry.Local.IsDirectory)
		}
		remoteImage := int32(-1)
		if entry.HasRemote {
			remoteImage = systemIconIndex(entry.Remote.Name, entry.Remote.IsDirectory)
		}
		insertListRowWithImage(state.localList, row, []string{entry.Name, status, localSize, localTime}, localImage)
		insertListRowWithImage(state.remoteList, row, []string{entry.Name, status, remoteSize, remoteTime}, remoteImage)
		if firstSync < 0 {
			if _, ok := a.engine.SynchronizedDirectoryName(state.entries, entry.Name); ok {
				firstSync = row
			}
		}
	}
	if firstSync >= 0 {
		setListRowSelected(state.localList, firstSync, true)
		setListRowSelected(state.remoteList, firstSync, true)
	} else if len(state.entries) > 0 {
		setListRowSelected(state.localList, 0, true)
		setListRowSelected(state.remoteList, 0, true)
	}
}

func (a *app) directoryComparisonCommand() {
	if a.directoryComparisonActive() {
		a.closeDirectoryComparison()
		return
	}
	a.startDirectoryComparison()
}

func (a *app) startDirectoryComparison() {
	if !a.comparisonCanStart() {
		return
	}
	state := a.directoryComparisonState()
	if state == nil {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	state.seq++
	seq := state.seq
	state.active = true
	state.loading = true
	state.entries = nil
	state.generation = a.connectionGeneration
	state.localBase = a.localCurrent
	state.remoteBase = a.remoteCurrent
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	state.cancel = cancel
	words := directoryCompareWordsForLanguage(a.languageCode())
	a.setStatus(words.Disclosure)
	a.layoutDirectoryComparisonControls()
	a.updateActionControls()

	localBase := state.localBase
	remoteBase := state.remoteBase
	generation := state.generation
	a.goSafe(func() {
		defer cancel()
		localResolved, localItems, err := a.engine.LocalList(ctx, localBase)
		if err == nil {
			var remoteItems []apiItemAlias
			_ = remoteItems
		}
		serverItems, remoteErr := a.engine.RemoteList(ctx, remoteBase)
		if err == nil {
			err = remoteErr
		}
		var entries []api.DirectoryComparisonEntry
		if err == nil {
			entries, err = a.engine.CompareDirectoryItems(localItems, serverItems, api.DirectoryComparisonOptions{})
		}
		a.dispatch(func() {
			current := a.directoryComparisonState()
			if current == nil || !current.active || current.seq != seq || generation != a.connectionGeneration || a.localCurrent != localBase || a.remoteCurrent != remoteBase {
				return
			}
			current.loading = false
			current.cancel = nil
			if err != nil {
				a.setStatus(a.userMessage(err, "error.generic"))
				a.updateDirectoryComparisonControls()
				return
			}
			current.localBase = localResolved
			current.entries = entries
			fillItems(a.localList, localItems)
			fillItems(a.remoteList, serverItems)
			a.renderDirectoryComparisonRows()
			a.setStatus(fmt.Sprintf("%s · %d", words.Ready, len(entries)))
			a.updateDirectoryComparisonControls()
			a.updateActionControls()
		})
	})
}

func (a *app) closeDirectoryComparison() {
	state := a.directoryComparisonState()
	if state == nil || !state.active {
		return
	}
	if state.cancel != nil {
		state.cancel()
	}
	state.seq++
	state.active = false
	state.loading = false
	state.cancel = nil
	state.entries = nil
	showControls(false, state.localList, state.remoteList, state.openBothButton)
	a.setStatus(directoryCompareWordsForLanguage(a.languageCode()).Close)
	a.updateActionControls()
}

func (a *app) selectedDirectoryComparisonEntry() (api.DirectoryComparisonEntry, bool) {
	state := a.directoryComparisonState()
	if state == nil || !state.active || state.loading {
		return api.DirectoryComparisonEntry{}, false
	}
	index := selectedIndex(state.localList)
	if index < 0 || index >= len(state.entries) {
		index = selectedIndex(state.remoteList)
	}
	if index < 0 || index >= len(state.entries) {
		return api.DirectoryComparisonEntry{}, false
	}
	return state.entries[index], true
}

func (a *app) openComparedDirectoryBoth() {
	state := a.directoryComparisonState()
	entry, ok := a.selectedDirectoryComparisonEntry()
	words := directoryCompareWordsForLanguage(a.languageCode())
	if state == nil || !ok {
		return
	}
	name, ok := a.engine.SynchronizedDirectoryName(state.entries, entry.Name)
	if !ok {
		a.setStatus(words.Unavailable)
		return
	}
	localTarget, err := security.SafeLocalChild(state.localBase, name)
	if err != nil {
		a.setStatus(a.userMessage(err, "error.generic"))
		return
	}
	if err := security.ValidateRemoteName(name); err != nil {
		a.setStatus(a.userMessage(err, "error.invalid_name"))
		return
	}
	remoteTarget := path.Clean(path.Join(state.remoteBase, name))
	if err := security.ValidateRemotePath(remoteTarget); err != nil {
		a.setStatus(a.userMessage(err, "error.generic"))
		return
	}

	if state.cancel != nil {
		state.cancel()
	}
	state.seq++
	seq := state.seq
	state.loading = true
	generation := a.connectionGeneration
	localBase := state.localBase
	remoteBase := state.remoteBase
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	state.cancel = cancel
	a.setStatus(words.OpenBoth + "…")
	a.updateDirectoryComparisonControls()

	a.goSafe(func() {
		defer cancel()
		localResolved, localItems, listErr := a.engine.LocalList(ctx, localTarget)
		var remoteItems []apiItemAlias
		_ = remoteItems
		serverItems, remoteErr := a.engine.RemoteList(ctx, remoteTarget)
		if listErr == nil {
			listErr = remoteErr
		}
		var entries []api.DirectoryComparisonEntry
		if listErr == nil {
			entries, listErr = a.engine.CompareDirectoryItems(localItems, serverItems, api.DirectoryComparisonOptions{})
		}
		a.dispatch(func() {
			current := a.directoryComparisonState()
			if current == nil || !current.active || current.seq != seq || generation != a.connectionGeneration || current.localBase != localBase || current.remoteBase != remoteBase {
				return
			}
			current.loading = false
			current.cancel = nil
			if listErr != nil {
				a.setStatus(a.userMessage(listErr, "error.generic"))
				a.updateDirectoryComparisonControls()
				return
			}
			if filter := a.fileFilterState(); filter != nil {
				filter.localQuery = ""
				filter.remoteQuery = ""
			}
			a.localCurrent = localResolved
			a.remoteCurrent = remoteTarget
			setText(a.localPath, a.localCurrent)
			setText(a.remotePath, a.remoteCurrent)
			fillItems(a.localList, localItems)
			fillItems(a.remoteList, serverItems)
			current.localBase = a.localCurrent
			current.remoteBase = a.remoteCurrent
			current.entries = entries
			a.renderDirectoryComparisonRows()
			a.setStatus(fmt.Sprintf("%s · %d", words.Ready, len(entries)))
			a.updateDirectoryComparisonControls()
			a.updateActionControls()
		})
	})
}

// apiItemAlias is intentionally impossible to instantiate usefully. It keeps
// accidental shadow declarations out of future edits; directory comparison uses
// the concrete model slices returned by Engine list calls above.
type apiItemAlias = struct{}

func directoryComparisonTrim(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
}
