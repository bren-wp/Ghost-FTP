//go:build linux

package desktop

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/security"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

type linuxDirectoryComparisonState struct {
	mu sync.Mutex

	seq      uint64
	active   bool
	running  bool
	selected int
	cancel   context.CancelFunc
	entries  []api.DirectoryComparisonEntry
	status   string
	dirty    bool

	localBase      string
	remoteBase     string
	localSnapshot  []model.Item
	remoteSnapshot []model.Item

	navReady       bool
	navLocalBase   string
	navRemoteBase  string
	navLocalItems  []model.Item
	navRemoteItems []model.Item
	navEntries     []api.DirectoryComparisonEntry
}

var linuxDirectoryComparisonStates sync.Map

func linuxDirectoryComparisonStateFor(u *linuxDesktop) *linuxDirectoryComparisonState {
	if value, ok := linuxDirectoryComparisonStates.Load(u); ok {
		return value.(*linuxDirectoryComparisonState)
	}
	state := &linuxDirectoryComparisonState{selected: -1}
	actual, _ := linuxDirectoryComparisonStates.LoadOrStore(u, state)
	return actual.(*linuxDirectoryComparisonState)
}

func (u *linuxDesktop) directoryComparisonActiveLinux() bool {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.active
}

func (u *linuxDesktop) directoryComparisonButtonRectLinux() linuxRect {
	left := u.layout.localList.left
	right := u.layout.upload.left - 8
	if right-left < 120 {
		right = left + 120
	}
	return linuxRectWH(left, u.layout.upload.top, right-left, u.layout.upload.bottom-u.layout.upload.top)
}

func directoryComparisonStatusLabelLinux(words directoryCompareWords, status api.DirectoryComparisonStatus) string {
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

func directoryComparisonVisibleLinux(entries []api.DirectoryComparisonEntry, remote bool, words directoryCompareWords) []model.Item {
	visible := make([]model.Item, 0, len(entries))
	for _, entry := range entries {
		item := entry.Local
		has := entry.HasLocal
		if remote {
			item = entry.Remote
			has = entry.HasRemote
		}
		if !has {
			item = model.Item{Name: entry.Name}
		}
		item.Name = fmt.Sprintf("%s  [%s]", entry.Name, directoryComparisonStatusLabelLinux(words, entry.Status))
		visible = append(visible, item)
	}
	return visible
}

func (u *linuxDesktop) reconcileDirectoryComparisonLinux() {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	active := state.active
	running := state.running
	selected := state.selected
	entries := append([]api.DirectoryComparisonEntry(nil), state.entries...)
	status := state.status
	dirty := state.dirty
	localBase := state.localBase
	remoteBase := state.remoteBase
	navReady := state.navReady
	navLocalBase := state.navLocalBase
	navRemoteBase := state.navRemoteBase
	navLocalItems := append([]model.Item(nil), state.navLocalItems...)
	navRemoteItems := append([]model.Item(nil), state.navRemoteItems...)
	navEntries := append([]api.DirectoryComparisonEntry(nil), state.navEntries...)
	if dirty {
		state.dirty = false
	}
	if navReady {
		state.navReady = false
		state.localBase = navLocalBase
		state.remoteBase = navRemoteBase
		state.localSnapshot = append([]model.Item(nil), navLocalItems...)
		state.remoteSnapshot = append([]model.Item(nil), navRemoteItems...)
		state.entries = append([]api.DirectoryComparisonEntry(nil), navEntries...)
		state.selected = -1
		state.navLocalItems = nil
		state.navRemoteItems = nil
		state.navEntries = nil
		localBase = navLocalBase
		remoteBase = navRemoteBase
		entries = navEntries
		selected = -1
	}
	state.mu.Unlock()

	if dirty && status != "" {
		u.setStatus(status)
	}
	if !active {
		return
	}

	// Comparison mode is modal. Keeping busy set prevents every ordinary file,
	// connection and transfer action from operating on comparison rows.
	u.busy = true
	u.localCurrent = localBase
	u.remoteCurrent = remoteBase
	if navReady {
		if filter := u.fileFilterState(); filter != nil {
			filter.localQuery = ""
			filter.remoteQuery = ""
		}
	}
	words := directoryCompareWordsForLanguage(u.language)
	u.localItems = directoryComparisonVisibleLinux(entries, false, words)
	u.remoteItems = directoryComparisonVisibleLinux(entries, true, words)
	u.selectedLocal = selected
	u.selectedRemote = selected
	if running && status == "" {
		u.setStatus(words.Disclosure)
	}
}

func (u *linuxDesktop) renderDirectoryComparisonButtonLinux() error {
	words := directoryCompareWordsForLanguage(u.language)
	enabled := u.connected && !u.busy && !u.recursiveSearchActive(false) && !u.recursiveSearchActive(true)
	return u.drawButton(u.directoryComparisonButtonRectLinux(), words.Compare, enabled, false)
}

func (u *linuxDesktop) renderDirectoryComparisonControlsLinux() error {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	running := state.running
	count := len(state.entries)
	selected := state.selected
	entries := append([]api.DirectoryComparisonEntry(nil), state.entries...)
	state.mu.Unlock()

	words := directoryCompareWordsForLanguage(u.language)
	openBoth := false
	if !running && selected >= 0 && selected < len(entries) {
		_, openBoth = u.engine.SynchronizedDirectoryName(entries, entries[selected].Name)
	}
	if err := u.drawButton(u.fileFilterControlRect(false), words.Close, true, false); err != nil {
		return err
	}
	if err := u.drawButton(u.recursiveSearchButtonRect(false), words.OpenBoth, openBoth, true); err != nil {
		return err
	}
	if err := u.drawButton(u.fileFilterControlRect(true), words.Compare, !running, false); err != nil {
		return err
	}
	summary := fmt.Sprintf("%s · %d", words.Ready, count)
	if running {
		summary = words.Disclosure
	}
	if err := u.drawButton(u.recursiveSearchButtonRect(true), summary, false, false); err != nil {
		return err
	}
	return u.drawButton(u.directoryComparisonButtonRectLinux(), words.Close, true, false)
}

func (u *linuxDesktop) startDirectoryComparisonLinux() {
	if u == nil || u.engine == nil || u.busy || !u.connected || u.recursiveSearchActive(false) || u.recursiveSearchActive(true) {
		return
	}
	state := linuxDirectoryComparisonStateFor(u)
	words := directoryCompareWordsForLanguage(u.language)
	localBase := u.localCurrent
	remoteBase := u.remoteCurrent
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)

	state.mu.Lock()
	if state.cancel != nil {
		state.cancel()
	}
	state.seq++
	seq := state.seq
	state.active = true
	state.running = true
	state.selected = -1
	state.cancel = cancel
	state.entries = nil
	state.status = words.Disclosure
	state.dirty = true
	state.localBase = localBase
	state.remoteBase = remoteBase
	state.mu.Unlock()
	u.busy = true
	u.setStatus(words.Disclosure)

	go func() {
		defer cancel()
		resolvedLocal, localItems, err := u.engine.LocalList(ctx, localBase)
		if err == nil {
			localBase = resolvedLocal
		}
		var remoteItems []model.Item
		if err == nil {
			remoteItems, err = u.engine.RemoteList(ctx, remoteBase)
		}
		var entries []api.DirectoryComparisonEntry
		if err == nil {
			entries, err = u.engine.CompareDirectoryItems(localItems, remoteItems, api.DirectoryComparisonOptions{})
		}

		state.mu.Lock()
		if state.seq == seq && state.active {
			state.running = false
			state.cancel = nil
			if err != nil {
				state.active = false
				state.status = usererror.MessageFor(u.language, err, words.Unavailable)
				state.dirty = true
			} else {
				state.localBase = localBase
				state.remoteBase = remoteBase
				state.localSnapshot = append([]model.Item(nil), localItems...)
				state.remoteSnapshot = append([]model.Item(nil), remoteItems...)
				state.entries = append([]api.DirectoryComparisonEntry(nil), entries...)
				state.selected = -1
				for i, entry := range entries {
					if _, ok := u.engine.SynchronizedDirectoryName(entries, entry.Name); ok {
						state.selected = i
						break
					}
				}
				state.status = fmt.Sprintf("%s · %d", words.Ready, len(entries))
				state.dirty = true
			}
		}
		state.mu.Unlock()
		u.resultCh <- linuxUIResult{action: linuxActionNone}
	}()
}

func (u *linuxDesktop) closeDirectoryComparisonLinux() {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	if !state.active {
		state.mu.Unlock()
		return
	}
	cancel := state.cancel
	localBase := state.localBase
	remoteBase := state.remoteBase
	localItems := append([]model.Item(nil), state.localSnapshot...)
	remoteItems := append([]model.Item(nil), state.remoteSnapshot...)
	state.seq++
	state.active = false
	state.running = false
	state.selected = -1
	state.cancel = nil
	state.entries = nil
	state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	u.busy = false
	u.localCurrent = localBase
	u.remoteCurrent = remoteBase
	u.acceptLinuxFileFilterSnapshot(false, localItems)
	u.acceptLinuxFileFilterSnapshot(true, remoteItems)
	u.selectedLocal = -1
	u.selectedRemote = -1
	u.setStatus(directoryCompareWordsForLanguage(u.language).Close)
}

func (u *linuxDesktop) openComparedDirectoryBothLinux() {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	if !state.active || state.running || state.selected < 0 || state.selected >= len(state.entries) {
		state.mu.Unlock()
		return
	}
	entries := append([]api.DirectoryComparisonEntry(nil), state.entries...)
	entry := entries[state.selected]
	localBase := state.localBase
	remoteBase := state.remoteBase
	state.mu.Unlock()

	name, ok := u.engine.SynchronizedDirectoryName(entries, entry.Name)
	words := directoryCompareWordsForLanguage(u.language)
	if !ok {
		u.setStatus(words.Unavailable)
		return
	}
	localTarget, err := security.SafeLocalChild(localBase, name)
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, words.Unavailable))
		return
	}
	if err := security.ValidateRemoteName(name); err != nil {
		u.setStatus(words.Unavailable)
		return
	}
	remoteTarget := path.Clean(path.Join(remoteBase, name))
	if err := security.ValidateRemotePath(remoteTarget); err != nil {
		u.setStatus(words.Unavailable)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	state.mu.Lock()
	if state.cancel != nil {
		state.cancel()
	}
	state.seq++
	seq := state.seq
	state.running = true
	state.cancel = cancel
	state.status = words.Disclosure
	state.dirty = true
	state.mu.Unlock()
	u.setStatus(words.Disclosure)

	go func() {
		defer cancel()
		resolvedLocal, localItems, err := u.engine.LocalList(ctx, localTarget)
		var remoteItems []model.Item
		if err == nil {
			remoteItems, err = u.engine.RemoteList(ctx, remoteTarget)
		}
		var entries []api.DirectoryComparisonEntry
		if err == nil {
			entries, err = u.engine.CompareDirectoryItems(localItems, remoteItems, api.DirectoryComparisonOptions{})
		}

		state.mu.Lock()
		if state.seq == seq && state.active {
			state.running = false
			state.cancel = nil
			if err != nil {
				state.status = usererror.MessageFor(u.language, err, words.Unavailable)
				state.dirty = true
			} else {
				state.navReady = true
				state.navLocalBase = resolvedLocal
				state.navRemoteBase = remoteTarget
				state.navLocalItems = append([]model.Item(nil), localItems...)
				state.navRemoteItems = append([]model.Item(nil), remoteItems...)
				state.navEntries = append([]api.DirectoryComparisonEntry(nil), entries...)
				state.status = fmt.Sprintf("%s · %d", words.Ready, len(entries))
				state.dirty = true
			}
		}
		state.mu.Unlock()
		u.resultCh <- linuxUIResult{action: linuxActionNone}
	}()
}

func (u *linuxDesktop) handleDirectoryComparisonMouseLinux(x, y int) bool {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	active := state.active
	running := state.running
	count := len(state.entries)
	state.mu.Unlock()
	if !active {
		if u.directoryComparisonButtonRectLinux().contains(x, y) {
			if !u.busy && u.connected {
				u.startDirectoryComparisonLinux()
			}
			return true
		}
		return false
	}

	// Comparison is modal: swallow all ordinary workspace clicks while active.
	if u.fileFilterControlRect(false).contains(x, y) || u.directoryComparisonButtonRectLinux().contains(x, y) {
		u.closeDirectoryComparisonLinux()
		return true
	}
	if u.recursiveSearchButtonRect(false).contains(x, y) {
		if !running {
			u.openComparedDirectoryBothLinux()
		}
		return true
	}
	if u.fileFilterControlRect(true).contains(x, y) {
		if !running {
			u.closeDirectoryComparisonLinux()
			u.startDirectoryComparisonLinux()
		}
		return true
	}
	for _, remote := range []bool{false, true} {
		list := u.fileFilterListRect(remote)
		if list.contains(x, y) {
			index := u.selectRow(list, y, count)
			state.mu.Lock()
			if state.active && !state.running {
				state.selected = index
			}
			state.mu.Unlock()
			u.selectedLocal = index
			u.selectedRemote = index
			return true
		}
	}
	return true
}

func (u *linuxDesktop) disposeDirectoryComparisonLinux() {
	state := linuxDirectoryComparisonStateFor(u)
	state.mu.Lock()
	cancel := state.cancel
	state.cancel = nil
	state.active = false
	state.running = false
	state.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	linuxDirectoryComparisonStates.Delete(u)
}
