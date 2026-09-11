//go:build linux

package desktop

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

type linuxBookmarkUIState struct {
	items        []model.Bookmark
	selected     int
	firstVisible int
	visibleRows  int
	list         linuxRect
	rows         linuxRect
	scrollUp     linuxRect
	scrollDown   linuxRect
	open         linuxRect
	addLocal     linuxRect
	addRemote    linuxRect
	delete       linuxRect
	close        linuxRect
}

var linuxBookmarkUIStates sync.Map

func linuxBookmarkStateFor(u *linuxDesktop) *linuxBookmarkUIState {
	if value, ok := linuxBookmarkUIStates.Load(u); ok {
		return value.(*linuxBookmarkUIState)
	}
	state := &linuxBookmarkUIState{selected: -1}
	actual, _ := linuxBookmarkUIStates.LoadOrStore(u, state)
	return actual.(*linuxBookmarkUIState)
}

func (state *linuxBookmarkUIState) ensureSelectionVisible() {
	if state == nil {
		return
	}
	rows := state.visibleRows
	if rows < 1 {
		rows = 1
	}
	if len(state.items) == 0 {
		state.selected = -1
		state.firstVisible = 0
		return
	}
	if state.selected < 0 {
		state.selected = 0
	}
	if state.selected >= len(state.items) {
		state.selected = len(state.items) - 1
	}
	maxFirst := len(state.items) - rows
	if maxFirst < 0 {
		maxFirst = 0
	}
	if state.firstVisible < 0 {
		state.firstVisible = 0
	}
	if state.firstVisible > maxFirst {
		state.firstVisible = maxFirst
	}
	if state.selected < state.firstVisible {
		state.firstVisible = state.selected
	}
	if state.selected >= state.firstVisible+rows {
		state.firstVisible = state.selected - rows + 1
	}
	if state.firstVisible > maxFirst {
		state.firstVisible = maxFirst
	}
}

func (state *linuxBookmarkUIState) scrollViewport(delta int) {
	if state == nil || len(state.items) == 0 || delta == 0 {
		return
	}
	rows := state.visibleRows
	if rows < 1 {
		rows = 1
	}
	maxFirst := len(state.items) - rows
	if maxFirst < 0 {
		maxFirst = 0
	}
	state.firstVisible += delta
	if state.firstVisible < 0 {
		state.firstVisible = 0
	}
	if state.firstVisible > maxFirst {
		state.firstVisible = maxFirst
	}
	if state.selected < state.firstVisible {
		state.selected = state.firstVisible
	}
	lastVisible := state.firstVisible + rows - 1
	if lastVisible >= len(state.items) {
		lastVisible = len(state.items) - 1
	}
	if state.selected > lastVisible {
		state.selected = lastVisible
	}
}

func (u *linuxDesktop) bookmarksHeaderRect() linuxRect {
	right := u.layout.settings.left - 8
	width := 118
	return linuxRectWH(right-width, u.layout.settings.top, width, u.layout.settings.bottom-u.layout.settings.top)
}

func (u *linuxDesktop) renderBookmarksHeaderButton() error {
	u.enforceLinuxProfileStartDirectories()
	words := bookmarkWordsForLanguage(u.language)
	return u.drawButton(u.bookmarksHeaderRect(), words.Title, !u.busy, false)
}

func (u *linuxDesktop) handleBookmarksHeaderMouse(x, y int) bool {
	if !u.bookmarksHeaderRect().contains(x, y) {
		return false
	}
	if !u.busy {
		u.openLinuxBookmarks("")
	}
	return true
}

func (u *linuxDesktop) openLinuxBookmarks(selectID string) {
	if u.busy {
		return
	}
	items, err := u.engine.Bookmarks()
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	state := linuxBookmarkStateFor(u)
	state.items = items
	state.selected = -1
	state.firstVisible = 0
	state.visibleRows = 0
	if len(items) != 0 {
		state.selected = 0
	}
	if selectID != "" {
		for index := range items {
			if items[index].ID == selectID {
				state.selected = index
				break
			}
		}
	}
	u.promptKind = linuxPromptBookmarkManager
	u.promptTitle = bookmarkWordsForLanguage(u.language).Title
	u.promptValue = ""
}

func linuxBookmarkName(pathValue string, remote bool) string {
	pathValue = strings.TrimSpace(pathValue)
	if remote {
		cleaned := cleanRemote(pathValue)
		if cleaned == "/" || cleaned == "." {
			return cleaned
		}
		name := path.Base(cleaned)
		if name != "" && name != "." && name != "/" {
			return name
		}
		return cleaned
	}
	cleaned := filepath.Clean(pathValue)
	name := filepath.Base(cleaned)
	if name != "" && name != "." {
		return name
	}
	return cleaned
}

func (u *linuxDesktop) renderBookmarkManagerOverlay() error {
	state := linuxBookmarkStateFor(u)
	words := bookmarkWordsForLanguage(u.language)
	width := min(780, u.width-80)
	height := min(470, u.height-80)
	if width < 660 {
		width = 660
	}
	if height < 340 {
		height = 340
	}
	left := (u.width - width) / 2
	top := (u.height - height) / 2
	panel := linuxRectWH(left, top, width, height)
	if err := u.drawPanel(panel); err != nil {
		return err
	}
	if err := u.x.text(left+20, top+30, strings.ToUpper(words.Title), premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}

	listTop := top + 48
	listBottom := top + height - 86
	state.list = linuxRectWH(left+20, listTop, width-40, listBottom-listTop)
	if err := u.x.fillRect(state.list.left, state.list.top, state.list.right-state.list.left, state.list.bottom-state.list.top, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(state.list.left, state.list.top, state.list.right-state.list.left, state.list.bottom-state.list.top, premiumTheme.Border); err != nil {
		return err
	}
	rowH := 32
	state.visibleRows = (state.list.bottom - state.list.top - 8) / rowH
	if state.visibleRows < 1 {
		state.visibleRows = 1
	}
	state.ensureSelectionVisible()
	state.rows = state.list
	state.scrollUp = linuxRect{}
	state.scrollDown = linuxRect{}
	if len(state.items) > state.visibleRows {
		scrollWidth := 28
		state.rows.right = state.list.right - scrollWidth - 8
		state.scrollUp = linuxRectWH(state.list.right-scrollWidth-5, state.list.top+5, scrollWidth, 28)
		state.scrollDown = linuxRectWH(state.list.right-scrollWidth-5, state.list.bottom-33, scrollWidth, 28)
		if err := u.drawButton(state.scrollUp, "^", state.firstVisible > 0 && !u.busy, false); err != nil {
			return err
		}
		if err := u.drawButton(state.scrollDown, "v", state.firstVisible+state.visibleRows < len(state.items) && !u.busy, false); err != nil {
			return err
		}
	}
	if len(state.items) == 0 {
		if err := u.x.text(state.rows.left+12, state.rows.top+26, words.Empty, premiumTheme.Muted, premiumTheme.List); err != nil {
			return err
		}
	} else {
		for visualRow := 0; visualRow < state.visibleRows; visualRow++ {
			index := state.firstVisible + visualRow
			if index >= len(state.items) {
				break
			}
			item := state.items[index]
			rowTop := state.rows.top + 4 + visualRow*rowH
			background := premiumTheme.List
			if index == state.selected {
				background = premiumTheme.Selection
				if err := u.x.fillRect(state.rows.left+1, rowTop, state.rows.right-state.rows.left-2, rowH, background); err != nil {
					return err
				}
			}
			kind := u.tr("section.local")
			identity := ""
			if item.Kind == model.BookmarkKindRemote {
				kind = u.tr("section.remote")
				identity = item.Username + "@" + item.Host + " · "
			}
			line := fmt.Sprintf("%s  ·  %s  ·  %s%s", kind, item.Name, identity, item.Path)
			if err := u.x.text(state.rows.left+10, rowTop+21, linuxTrimForUI(line, max(24, (state.rows.right-state.rows.left-20)/7)), premiumTheme.Text, background); err != nil {
				return err
			}
		}
	}

	buttonTop := top + height - 58
	buttonGap := 8
	state.open = linuxRectWH(left+20, buttonTop, 104, 32)
	state.addLocal = linuxRectWH(state.open.right+buttonGap, buttonTop, 132, 32)
	state.addRemote = linuxRectWH(state.addLocal.right+buttonGap, buttonTop, 132, 32)
	state.delete = linuxRectWH(state.addRemote.right+buttonGap, buttonTop, 104, 32)
	state.close = linuxRectWH(left+width-124, buttonTop, 104, 32)
	hasSelection := state.selected >= 0 && state.selected < len(state.items)
	if err := u.drawButton(state.open, words.Open, hasSelection && !u.busy, true); err != nil {
		return err
	}
	if err := u.drawButton(state.addLocal, words.AddLocal, strings.TrimSpace(u.localCurrent) != "" && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(state.addRemote, words.AddRemote, u.connected && strings.TrimSpace(u.remoteCurrent) != "" && !u.busy, false); err != nil {
		return err
	}
	if err := u.drawButton(state.delete, words.Delete, hasSelection && !u.busy, false); err != nil {
		return err
	}
	return u.drawButton(state.close, words.Close, !u.busy, false)
}

func (u *linuxDesktop) bookmarkRowAt(x, y int) int {
	state := linuxBookmarkStateFor(u)
	if !state.rows.contains(x, y) || y < state.rows.top+4 {
		return -1
	}
	visualRow := (y - state.rows.top - 4) / 32
	if visualRow < 0 || visualRow >= state.visibleRows {
		return -1
	}
	index := state.firstVisible + visualRow
	if index < 0 || index >= len(state.items) {
		return -1
	}
	return index
}

func (u *linuxDesktop) handleBookmarkManagerMouse(x, y int) bool {
	state := linuxBookmarkStateFor(u)
	if state.scrollUp.contains(x, y) {
		if !u.busy {
			state.scrollViewport(-1)
		}
		return true
	}
	if state.scrollDown.contains(x, y) {
		if !u.busy {
			state.scrollViewport(1)
		}
		return true
	}
	if index := u.bookmarkRowAt(x, y); index >= 0 {
		state.selected = index
		state.ensureSelectionVisible()
		return true
	}
	switch {
	case state.open.contains(x, y):
		u.openSelectedLinuxBookmark()
	case state.addLocal.contains(x, y):
		if !u.busy && strings.TrimSpace(u.localCurrent) != "" {
			words := bookmarkWordsForLanguage(u.language)
			u.openPrompt(linuxPromptBookmarkLocalName, words.AddLocal+" · "+words.Name, linuxBookmarkName(u.localCurrent, false))
		}
	case state.addRemote.contains(x, y):
		if !u.busy && u.connected && strings.TrimSpace(u.remoteCurrent) != "" {
			words := bookmarkWordsForLanguage(u.language)
			u.openPrompt(linuxPromptBookmarkRemoteName, words.AddRemote+" · "+words.Name, linuxBookmarkName(u.remoteCurrent, true))
		}
	case state.delete.contains(x, y):
		u.deleteSelectedLinuxBookmark()
	case state.close.contains(x, y):
		u.closePrompt()
	default:
		return true
	}
	return true
}

func (u *linuxDesktop) handleBookmarkManagerKey(sym uint32) bool {
	state := linuxBookmarkStateFor(u)
	switch sym {
	case x11KeyEscape:
		u.closePrompt()
	case x11KeyUp:
		if state.selected > 0 {
			state.selected--
			state.ensureSelectionVisible()
		}
	case x11KeyDown:
		if state.selected+1 < len(state.items) {
			state.selected++
			state.ensureSelectionVisible()
		}
	case x11KeyReturn:
		u.openSelectedLinuxBookmark()
	case x11KeyDelete:
		u.deleteSelectedLinuxBookmark()
	}
	return true
}

func (u *linuxDesktop) saveLinuxBookmark(remote bool, name string) {
	words := bookmarkWordsForLanguage(u.language)
	var (
		saved model.Bookmark
		err   error
	)
	if remote {
		if !u.connected {
			u.setStatus(i18n.T(u.language, "error.generic"))
			return
		}
		saved, err = u.engine.SaveRemoteBookmark("", name, u.remoteCurrent)
	} else {
		saved, err = u.engine.SaveLocalBookmark("", name, u.localCurrent)
	}
	if err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	u.setStatus(words.Saved)
	u.openLinuxBookmarks(saved.ID)
}

func (u *linuxDesktop) deleteSelectedLinuxBookmark() {
	state := linuxBookmarkStateFor(u)
	if u.busy || state.selected < 0 || state.selected >= len(state.items) {
		return
	}
	item := state.items[state.selected]
	if !u.destructiveConfirmed("bookmark:"+item.ID, item.Name) {
		return
	}
	if err := u.engine.RemoveBookmark(item.ID); err != nil {
		u.setStatus(usererror.MessageFor(u.language, err, i18n.T(u.language, "error.generic")))
		return
	}
	words := bookmarkWordsForLanguage(u.language)
	u.setStatus(words.Removed)
	u.openLinuxBookmarks("")
}

func (u *linuxDesktop) openSelectedLinuxBookmark() {
	state := linuxBookmarkStateFor(u)
	if u.busy || state.selected < 0 || state.selected >= len(state.items) {
		return
	}
	bookmark := state.items[state.selected]
	if bookmark.Kind == model.BookmarkKindRemote && !u.connected {
		u.setStatus(i18n.T(u.language, "error.generic"))
		return
	}
	u.closePrompt()
	words := bookmarkWordsForLanguage(u.language)
	u.setStatus(words.Open + ": " + bookmark.Name)
	if bookmark.Kind == model.BookmarkKindLocal {
		u.startAction(linuxActionLocalRefresh, func() linuxUIResult {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			result, err := u.engine.NavigateBookmark(ctx, bookmark.ID)
			return linuxUIResult{localBase: result.LocalBase, localItems: result.Items, err: err}
		})
		return
	}
	u.startAction(linuxActionRemoteRefresh, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()
		result, err := u.engine.NavigateBookmark(ctx, bookmark.ID)
		return linuxUIResult{localBase: result.Bookmark.Path, remoteItems: result.Items, err: err}
	})
}
