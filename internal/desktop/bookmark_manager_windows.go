//go:build windows

package desktop

import (
	"context"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

const (
	idBookmarks = 97

	bookmarkIDList      = 8201
	bookmarkIDOpen      = 8202
	bookmarkIDAddLocal  = 8203
	bookmarkIDAddRemote = 8204
	bookmarkIDDelete    = 8205
	bookmarkIDClose     = 8206

	bookmarkWindowStyle = 0x00C80000 // WS_CAPTION | WS_SYSMENU
)

type bookmarkManagerState struct {
	parent    *app
	hwnd      uintptr
	list      uintptr
	listBrush uintptr
	open      uintptr
	addLocal  uintptr
	addRemote uintptr
	delete    uintptr
	close     uintptr
	items     []model.Bookmark
	selected  int
	closed    bool
	openAfter *model.Bookmark
}

var (
	bookmarkManagerStates sync.Map
	bookmarkManagerOnce   sync.Once
	bookmarkManagerClass  = "GhostFTP.Bookmarks"
	bookmarkManagerProc   = syscall.NewCallback(bookmarkManagerWndProc)
)

func bookmarkManagerWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	value, ok := bookmarkManagerStates.Load(hwnd)
	if ok {
		state := value.(*bookmarkManagerState)
		switch message {
		case wmCommand:
			id := int(wParam & 0xffff)
			notify := int((wParam >> 16) & 0xffff)
			if id == bookmarkIDList && notify == siteLBNSelChange {
				state.readSelection()
				return 0
			}
			if id == bookmarkIDList && notify == siteLBNDblClk {
				state.readSelection()
				state.openCurrent()
				return 0
			}
			if notify == bnClicked {
				switch id {
				case bookmarkIDOpen:
					state.openCurrent()
					return 0
				case bookmarkIDAddLocal:
					state.addCurrent(false)
					return 0
				case bookmarkIDAddRemote:
					state.addCurrent(true)
					return 0
				case bookmarkIDDelete:
					state.deleteCurrent()
					return 0
				case bookmarkIDClose:
					destroyWindow.Call(hwnd)
					return 0
				}
			}
		case wmDrawItem:
			if lParam != 0 {
				d := drawItemFromLParam(lParam)
				if state.parent.drawButton(&d) {
					return 1
				}
			}
		case siteWMCtlColorListBox:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, listColor())
			if state.listBrush != 0 {
				return state.listBrush
			}
			return state.parent.panelBrush
		case wmCtlColorEdit, wmCtlColorBtn, wmCtlColorStatic:
			setTextColor.Call(wParam, textColor())
			setBkColor.Call(wParam, windowColor())
			return state.parent.brush
		case wmClose:
			destroyWindow.Call(hwnd)
			return 0
		case wmDestroy:
			for _, button := range []uintptr{state.open, state.addLocal, state.addRemote, state.delete, state.close} {
				delete(state.parent.buttons, button)
			}
			if state.listBrush != 0 {
				deleteObject.Call(state.listBrush)
				state.listBrush = 0
			}
			state.closed = true
			return 0
		}
	}
	result, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return result
}

func bookmarkDefaultName(bookmarkPath string, remote bool) string {
	bookmarkPath = strings.TrimSpace(bookmarkPath)
	if remote {
		cleaned := cleanRemote(bookmarkPath)
		if cleaned == "/" || cleaned == "." {
			return cleaned
		}
		if name := path.Base(cleaned); name != "" && name != "." && name != "/" {
			return name
		}
		return cleaned
	}
	cleaned := filepath.Clean(bookmarkPath)
	if name := filepath.Base(cleaned); name != "" && name != "." {
		return name
	}
	return cleaned
}

func (state *bookmarkManagerState) displayLabel(item model.Bookmark) string {
	kind := state.parent.tr("section.local")
	identity := ""
	if item.Kind == model.BookmarkKindRemote {
		kind = state.parent.tr("section.remote")
		identity = item.Username + "@" + item.Host + "  ·  "
	}
	return "[" + kind + "]  " + item.Name + "  ·  " + identity + item.Path
}

func (state *bookmarkManagerState) loadItems(selectID string) error {
	items, err := state.parent.engine.Bookmarks()
	if err != nil {
		return err
	}
	state.items = items
	sendMessageW.Call(state.list, siteLBResetContent, 0, 0)
	selected := -1
	for index, item := range items {
		label := state.displayLabel(item)
		sendMessageW.Call(state.list, siteLBAddString, 0, uintptr(unsafe.Pointer(wstr(label))))
		if item.ID == selectID {
			selected = index
		}
	}
	state.selected = selected
	if selected >= 0 {
		sendMessageW.Call(state.list, siteLBSetCurSel, uintptr(selected), 0)
	}
	state.updateButtons()
	return nil
}

func (state *bookmarkManagerState) readSelection() {
	index, _, _ := sendMessageW.Call(state.list, siteLBGetCurSel, 0, 0)
	if int32(index) < 0 || int(index) >= len(state.items) {
		state.selected = -1
	} else {
		state.selected = int(index)
	}
	state.updateButtons()
}

func (state *bookmarkManagerState) updateButtons() {
	hasSelection := state.selected >= 0 && state.selected < len(state.items)
	openEnabled := hasSelection && !state.parent.connectionBusy
	if openEnabled && state.items[state.selected].Kind == model.BookmarkKindRemote {
		cfg, connected := state.parent.engine.ActiveConnection()
		openEnabled = connected && state.parent.connected && config.RemoteBookmarkMatchesAccount(state.items[state.selected], cfg)
	}
	setControlEnabled(state.open, openEnabled)
	setControlEnabled(state.delete, hasSelection)
	setControlEnabled(state.addLocal, strings.TrimSpace(state.parent.localCurrent) != "")
	setControlEnabled(state.addRemote, state.parent.connected && !state.parent.connectionBusy && strings.TrimSpace(state.parent.remoteCurrent) != "")
}

func (state *bookmarkManagerState) addCurrent(remote bool) {
	words := bookmarkWordsForLanguage(state.parent.languageCode())
	bookmarkPath := state.parent.localCurrent
	title := words.AddLocal
	if remote {
		if !state.parent.connected || state.parent.connectionBusy {
			return
		}
		bookmarkPath = state.parent.remoteCurrent
		title = words.AddRemote
	}
	bookmarkPath = strings.TrimSpace(bookmarkPath)
	if bookmarkPath == "" {
		return
	}
	name, ok := platform.PromptDialogWithLabels(
		"Ghost FTP — "+words.Title,
		words.Name+":",
		bookmarkDefaultName(bookmarkPath, remote),
		okLabel(state.parent.languageCode()),
		state.parent.tr("common.cancel"),
	)
	if !ok || strings.TrimSpace(name) == "" {
		return
	}
	var (
		saved model.Bookmark
		err   error
	)
	if remote {
		saved, err = state.parent.engine.SaveRemoteBookmark("", name, bookmarkPath)
	} else {
		saved, err = state.parent.engine.SaveLocalBookmark("", name, bookmarkPath)
	}
	if err != nil {
		platform.ErrorDialog("Ghost FTP — "+words.Title, title, state.parent.userMessage(err, "error.generic"))
		return
	}
	if err := state.loadItems(saved.ID); err != nil {
		platform.ErrorDialog("Ghost FTP — "+words.Title, title, state.parent.userMessage(err, "error.generic"))
		return
	}
	state.parent.setStatus(words.Saved)
}

func (state *bookmarkManagerState) deleteCurrent() {
	if state.selected < 0 || state.selected >= len(state.items) {
		return
	}
	item := state.items[state.selected]
	words := bookmarkWordsForLanguage(state.parent.languageCode())
	if !platform.ConfirmDialog("Ghost FTP — "+words.Title, words.Delete+"?", item.Name+"\n"+item.Path) {
		return
	}
	if err := state.parent.engine.RemoveBookmark(item.ID); err != nil {
		platform.ErrorDialog("Ghost FTP — "+words.Title, words.Delete, state.parent.userMessage(err, "error.generic"))
		return
	}
	if err := state.loadItems(""); err != nil {
		platform.ErrorDialog("Ghost FTP — "+words.Title, words.Delete, state.parent.userMessage(err, "error.generic"))
		return
	}
	state.parent.setStatus(words.Removed)
}

func (state *bookmarkManagerState) openCurrent() {
	if state.selected < 0 || state.selected >= len(state.items) || state.parent.connectionBusy {
		return
	}
	item := state.items[state.selected]
	if item.Kind == model.BookmarkKindRemote {
		cfg, connected := state.parent.engine.ActiveConnection()
		if !connected || !state.parent.connected || !config.RemoteBookmarkMatchesAccount(item, cfg) {
			return
		}
	}
	state.openAfter = &item
	destroyWindow.Call(state.hwnd)
}

func (state *bookmarkManagerState) createControls(hinst uintptr) error {
	parent := state.parent
	words := bookmarkWordsForLanguage(parent.languageCode())
	mk := func(class, text string, style uint32, x, y, width, height, id int) uintptr {
		hwnd, _, _ := createWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(wstr(class))),
			uintptr(unsafe.Pointer(wstr(text))),
			uintptr(wsChild|wsVisible|style),
			uintptr(parent.scale(x)), uintptr(parent.scale(y)), uintptr(parent.scale(width)), uintptr(parent.scale(height)),
			state.hwnd, uintptr(id), hinst, 0,
		)
		if hwnd != 0 && parent.font != 0 {
			sendMessageW.Call(hwnd, wmSetFont, parent.font, 1)
		}
		if hwnd != 0 {
			applyDarkControl(hwnd, class)
		}
		return hwnd
	}
	state.listBrush, _, _ = createSolidBrush.Call(listColor())
	if state.listBrush == 0 {
		return fmt.Errorf("bookmark manager list brush initialization failed")
	}
	state.list = mk("LISTBOX", "", wsBorder|wsTabStop|wsVScroll|siteLBSNotify|siteLBSNoIntegralHeight, 20, 20, 700, 330, bookmarkIDList)
	state.open = parent.registerButton(mk("BUTTON", words.Open, wsTabStop|bsOwnerDraw, 20, 370, 118, 34, bookmarkIDOpen), iconOpenLocal, words.Open, buttonAccent)
	state.addLocal = parent.registerButton(mk("BUTTON", words.AddLocal, wsTabStop|bsOwnerDraw, 148, 370, 138, 34, bookmarkIDAddLocal), iconSave, words.AddLocal, buttonDefault)
	state.addRemote = parent.registerButton(mk("BUTTON", words.AddRemote, wsTabStop|bsOwnerDraw, 296, 370, 138, 34, bookmarkIDAddRemote), iconSave, words.AddRemote, buttonDefault)
	state.delete = parent.registerButton(mk("BUTTON", words.Delete, wsTabStop|bsOwnerDraw, 444, 370, 118, 34, bookmarkIDDelete), iconDelete, words.Delete, buttonDanger)
	state.close = parent.registerButton(mk("BUTTON", words.Close, wsTabStop|bsOwnerDraw, 572, 370, 148, 34, bookmarkIDClose), iconCancel, words.Close, buttonSubtle)
	for _, control := range []uintptr{state.list, state.open, state.addLocal, state.addRemote, state.delete, state.close} {
		if control == 0 {
			return fmt.Errorf("bookmark manager control initialization failed")
		}
	}
	return state.loadItems("")
}

func (a *app) openBookmarkManager() {
	if a == nil || a.hwnd == 0 || a.connectionBusy {
		return
	}
	hinst, _, _ := getModuleHandleW.Call(0)
	bookmarkManagerOnce.Do(func() {
		cursor, _, _ := loadCursorW.Call(0, 32512)
		icon, _, _ := loadIconW.Call(hinst, 1)
		class := wndClassEx{
			CbSize: uint32(unsafe.Sizeof(wndClassEx{})), WndProc: bookmarkManagerProc,
			Instance: hinst, Icon: icon, Cursor: cursor, Background: a.brush,
			ClassName: wstr(bookmarkManagerClass), IconSm: icon,
		}
		registerClassExW.Call(uintptr(unsafe.Pointer(&class)))
	})

	logicalW, logicalH := 760, 470
	pixelW, pixelH := a.scale(logicalW), a.scale(logicalH)
	screenW, _, _ := getSystemMetrics.Call(smCxScreen)
	screenH, _, _ := getSystemMetrics.Call(smCyScreen)
	x := max(0, (int(screenW)-pixelW)/2)
	y := max(0, (int(screenH)-pixelH)/2)
	words := bookmarkWordsForLanguage(a.languageCode())
	hwnd, _, _ := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(wstr(bookmarkManagerClass))),
		uintptr(unsafe.Pointer(wstr("Ghost FTP — "+words.Title))),
		bookmarkWindowStyle,
		uintptr(x), uintptr(y), uintptr(pixelW), uintptr(pixelH),
		a.hwnd, 0, hinst, 0,
	)
	if hwnd == 0 {
		platform.ErrorDialog("Ghost FTP", words.Title, a.tr("error.generic"))
		return
	}
	state := &bookmarkManagerState{parent: a, hwnd: hwnd, selected: -1}
	bookmarkManagerStates.Store(hwnd, state)
	defer bookmarkManagerStates.Delete(hwnd)
	applyDarkTitleBar(hwnd)
	if err := state.createControls(hinst); err != nil {
		destroyWindow.Call(hwnd)
		platform.ErrorDialog("Ghost FTP", words.Title, a.userMessage(err, "error.generic"))
		return
	}
	enableWindow.Call(a.hwnd, 0)
	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)
	var message msg
	for !state.closed {
		result, _, callErr := getMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(result) == -1 {
			if callErr != nil && callErr != syscall.Errno(0) {
				a.setStatus(callErr.Error())
			}
			break
		}
		if result == 0 {
			postQuitMessage.Call(0)
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&message)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	enableWindow.Call(a.hwnd, 1)
	showWindow.Call(a.hwnd, swShow)
	if state.openAfter != nil {
		a.navigateBookmark(*state.openAfter)
	}
}

func (a *app) navigateBookmark(bookmark model.Bookmark) {
	words := bookmarkWordsForLanguage(a.languageCode())
	if bookmark.Kind == model.BookmarkKindLocal {
		if a.localNavCancel != nil {
			a.localNavCancel()
		}
		a.localNavSeq++
		seq := a.localNavSeq
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		a.localNavCancel = cancel
		a.goSafe(func() {
			defer cancel()
			result, err := a.engine.NavigateBookmark(ctx, bookmark.ID)
			a.dispatch(func() {
				if seq != a.localNavSeq {
					return
				}
				a.localNavCancel = nil
				if err != nil {
					a.setStatus(a.userMessage(err, "error.generic"))
					return
				}
				a.localCurrent = result.LocalBase
				a.localItems = append(a.localItems[:0], result.Items...)
				setText(a.localPath, a.localCurrent)
				fillItems(a.localList, a.localItems)
				a.updateActionControls()
				a.setStatus(words.Open + ": " + result.Bookmark.Name)
			})
		})
		return
	}
	if bookmark.Kind != model.BookmarkKindRemote || !a.connected || a.connectionBusy {
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
		result, err := a.engine.NavigateBookmark(ctx, bookmark.ID)
		a.dispatch(func() {
			if seq != a.remoteNavSeq || generation != a.connectionGeneration || !a.connected {
				return
			}
			a.remoteNavCancel = nil
			if err != nil {
				if a.suppressExpectedDisconnectError(err) {
					return
				}
				a.setStatus(a.userMessage(err, "error.generic"))
				a.checkConnectionAfterError()
				return
			}
			a.remoteCurrent = cleanRemote(result.Bookmark.Path)
			a.remoteItems = append(a.remoteItems[:0], result.Items...)
			setText(a.remotePath, a.remoteCurrent)
			fillItems(a.remoteList, a.remoteItems)
			a.updateActionControls()
			a.setStatus(words.Open + ": " + result.Bookmark.Name)
		})
	})
}
