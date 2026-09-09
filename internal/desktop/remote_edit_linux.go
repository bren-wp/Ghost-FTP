//go:build linux

package desktop

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/i18n"
	"github.com/bren-wp/Ghost-FTP/internal/usererror"
)

const (
	linuxRemoteEditPendingNone = iota
	linuxRemoteEditPendingOpen
	linuxRemoteEditPendingSave
	linuxRemoteEditPendingReload
)

type linuxRemoteEditState struct {
	mu sync.Mutex

	open        bool
	doc         api.RemoteEditDocument
	buffer      remoteEditBuffer
	text        string
	cursor      int
	firstLine   int
	firstColumn int
	pending     int
	message     string

	closeConfirmUntil  time.Time
	reloadConfirmUntil time.Time
}

type linuxRemoteEditLayout struct {
	panel  linuxRect
	editor linuxRect
	save   linuxRect
	reload linuxRect
	close  linuxRect
}

var linuxRemoteEditStates sync.Map

func linuxRemoteEditStateFor(u *linuxDesktop) *linuxRemoteEditState {
	if value, ok := linuxRemoteEditStates.Load(u); ok {
		return value.(*linuxRemoteEditState)
	}
	state := &linuxRemoteEditState{}
	actual, _ := linuxRemoteEditStates.LoadOrStore(u, state)
	return actual.(*linuxRemoteEditState)
}

func (u *linuxDesktop) remoteEditSelectionReady() bool {
	if u == nil || !u.connected || u.busy {
		return false
	}
	item, ok := u.selectedRemoteItem()
	if !ok {
		return false
	}
	return !item.IsDirectory && !item.IsSymlink && item.Size <= api.MaxRemoteEditBytes
}

func (u *linuxDesktop) remoteEditButtonRect() linuxRect {
	left := u.layout.download.right + 8
	width := 92
	if left+width > u.layout.remoteList.right {
		left = u.layout.remoteList.right - width
	}
	return linuxRectWH(left, u.layout.download.top, width, u.layout.download.bottom-u.layout.download.top)
}

func (u *linuxDesktop) renderRemoteEditButton() error {
	words := remoteEditWords(u.language)
	return u.drawButton(u.remoteEditButtonRect(), words.Edit, u.remoteEditSelectionReady(), false)
}

func (u *linuxDesktop) remoteEditorOpen() bool {
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.open
}

func (u *linuxDesktop) openSelectedRemoteEditor() {
	words := remoteEditWords(u.language)
	if !u.remoteEditSelectionReady() {
		u.setStatus(words.Unsupported)
		return
	}
	item, _ := u.selectedRemoteItem()
	remotePath := terminalRemotePath(u.remoteCurrent, item.Name)
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	if state.pending != linuxRemoteEditPendingNone {
		state.mu.Unlock()
		return
	}
	state.pending = linuxRemoteEditPendingOpen
	state.message = words.Opening
	state.mu.Unlock()
	u.setStatus(words.Opening)
	u.startAction(linuxActionNone, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		doc, err := u.engine.RemoteEditOpen(ctx, remotePath)
		var buffer remoteEditBuffer
		if err == nil {
			buffer, err = newRemoteEditBuffer(doc.Text)
		}
		if err == nil {
			state.mu.Lock()
			state.open = true
			state.doc = doc
			state.buffer = buffer
			state.text = buffer.Text
			state.cursor = len(buffer.Text)
			state.firstLine = 0
			state.firstColumn = 0
			state.closeConfirmUntil = time.Time{}
			state.reloadConfirmUntil = time.Time{}
			state.mu.Unlock()
		}
		return linuxUIResult{err: err}
	})
}

func (u *linuxDesktop) handleRemoteEditResult(result linuxUIResult) bool {
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	pending := state.pending
	if pending == linuxRemoteEditPendingNone {
		state.mu.Unlock()
		return false
	}
	state.pending = linuxRemoteEditPendingNone
	state.mu.Unlock()

	words := remoteEditWords(u.language)
	if result.err != nil {
		message := usererror.MessageFor(u.language, result.err, i18n.T(u.language, "error.generic"))
		if errors.Is(result.err, api.ErrRemoteEditConflict) {
			message = words.Conflict
		} else if errors.Is(result.err, errRemoteEditMixedNewlines) {
			message = words.MixedNewlines
		}
		state.mu.Lock()
		state.message = message
		if pending == linuxRemoteEditPendingOpen {
			state.open = false
		}
		state.mu.Unlock()
		u.setStatus(message)
		return true
	}

	message := words.Loaded
	switch pending {
	case linuxRemoteEditPendingSave:
		message = words.Saved
	case linuxRemoteEditPendingReload:
		message = words.Reloaded
	}
	state.mu.Lock()
	state.message = message
	state.mu.Unlock()
	u.setStatus(message)
	return true
}

func (u *linuxDesktop) remoteEditSave() {
	if u.busy || !u.connected {
		return
	}
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	if !state.open || state.pending != linuxRemoteEditPendingNone {
		state.mu.Unlock()
		return
	}
	doc := state.doc
	buffer := state.buffer
	text := state.text
	cursor := state.cursor
	if text == buffer.Text {
		state.message = remoteEditWords(u.language).Saved
		state.mu.Unlock()
		return
	}
	state.pending = linuxRemoteEditPendingSave
	state.message = remoteEditWords(u.language).Saving
	state.mu.Unlock()

	words := remoteEditWords(u.language)
	u.setStatus(words.Saving)
	encoded := buffer.Encode(text)
	u.startAction(linuxActionNone, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		saved, err := u.engine.RemoteEditSave(ctx, doc.Path, doc.Revision, encoded)
		var next remoteEditBuffer
		if err == nil {
			next, err = newRemoteEditBuffer(saved.Text)
		}
		if err == nil {
			state.mu.Lock()
			state.doc = saved
			state.buffer = next
			state.text = next.Text
			if cursor > len(next.Text) {
				cursor = len(next.Text)
			}
			state.cursor = remoteEditClampCursor(next.Text, cursor)
			state.closeConfirmUntil = time.Time{}
			state.reloadConfirmUntil = time.Time{}
			state.mu.Unlock()
		}
		return linuxUIResult{err: err}
	})
}

func (u *linuxDesktop) remoteEditReload() {
	if u.busy || !u.connected {
		return
	}
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	if !state.open || state.pending != linuxRemoteEditPendingNone {
		state.mu.Unlock()
		return
	}
	words := remoteEditWords(u.language)
	now := time.Now()
	if state.text != state.buffer.Text && now.After(state.reloadConfirmUntil) {
		state.reloadConfirmUntil = now.Add(8 * time.Second)
		state.message = words.DiscardReload
		state.mu.Unlock()
		u.setStatus(words.DiscardReload)
		return
	}
	remotePath := state.doc.Path
	state.pending = linuxRemoteEditPendingReload
	state.message = words.Reloading
	state.reloadConfirmUntil = time.Time{}
	state.mu.Unlock()
	u.setStatus(words.Reloading)
	u.startAction(linuxActionNone, func() linuxUIResult {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		doc, err := u.engine.RemoteEditOpen(ctx, remotePath)
		var buffer remoteEditBuffer
		if err == nil {
			buffer, err = newRemoteEditBuffer(doc.Text)
		}
		if err == nil {
			state.mu.Lock()
			state.doc = doc
			state.buffer = buffer
			state.text = buffer.Text
			state.cursor = len(buffer.Text)
			state.firstLine = 0
			state.firstColumn = 0
			state.closeConfirmUntil = time.Time{}
			state.mu.Unlock()
		}
		return linuxUIResult{err: err}
	})
}

func (u *linuxDesktop) remoteEditClose() {
	if u.busy {
		return
	}
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	if !state.open {
		state.mu.Unlock()
		return
	}
	words := remoteEditWords(u.language)
	now := time.Now()
	if state.text != state.buffer.Text && now.After(state.closeConfirmUntil) {
		state.closeConfirmUntil = now.Add(8 * time.Second)
		state.message = words.DiscardClose
		state.mu.Unlock()
		u.setStatus(words.DiscardClose)
		return
	}
	state.open = false
	state.doc = api.RemoteEditDocument{}
	state.buffer = remoteEditBuffer{}
	state.text = ""
	state.cursor = 0
	state.firstLine = 0
	state.firstColumn = 0
	state.message = ""
	state.closeConfirmUntil = time.Time{}
	state.reloadConfirmUntil = time.Time{}
	state.mu.Unlock()
	u.setStatus(words.Close)
}

func (u *linuxDesktop) remoteEditLayout() linuxRemoteEditLayout {
	width := min(980, u.width-64)
	if width < 520 {
		width = 520
	}
	height := min(700, u.height-64)
	if height < 360 {
		height = 360
	}
	left := (u.width - width) / 2
	top := (u.height - height) / 2
	panel := linuxRectWH(left, top, width, height)
	footerY := panel.bottom - 58
	return linuxRemoteEditLayout{
		panel:  panel,
		editor: linuxRect{left: panel.left + 20, top: panel.top + 76, right: panel.right - 20, bottom: footerY - 34},
		save:   linuxRectWH(panel.right-332, footerY+10, 96, 32),
		reload: linuxRectWH(panel.right-226, footerY+10, 96, 32),
		close:  linuxRectWH(panel.right-120, footerY+10, 96, 32),
	}
}

func remoteEditCursorLineColumn(text string, cursor int) (line, column, lineStart int) {
	cursor = remoteEditClampCursor(text, cursor)
	lineStart = strings.LastIndex(text[:cursor], "\n") + 1
	line = strings.Count(text[:lineStart], "\n")
	column = utf8.RuneCountInString(text[lineStart:cursor])
	return line, column, lineStart
}

func remoteEditLineBounds(text string, line int) (start, end int, ok bool) {
	if line < 0 {
		return 0, 0, false
	}
	start = 0
	for current := 0; current < line; current++ {
		relative := strings.IndexByte(text[start:], '\n')
		if relative < 0 {
			return 0, 0, false
		}
		start += relative + 1
	}
	if relative := strings.IndexByte(text[start:], '\n'); relative >= 0 {
		end = start + relative
	} else {
		end = len(text)
	}
	return start, end, true
}

func remoteEditByteAtColumn(line string, column int) int {
	if column <= 0 {
		return 0
	}
	position := 0
	for i := 0; i < column && position < len(line); i++ {
		_, size := utf8.DecodeRuneInString(line[position:])
		if size <= 0 {
			break
		}
		position += size
	}
	return position
}

func remoteEditClampCursor(text string, cursor int) int {
	if cursor < 0 {
		return 0
	}
	if cursor > len(text) {
		cursor = len(text)
	}
	for cursor > 0 && cursor < len(text) && !utf8.RuneStart(text[cursor]) {
		cursor--
	}
	return cursor
}

func remoteEditVisibleSegment(line string, firstColumn, count int) string {
	if count <= 0 {
		return ""
	}
	runes := []rune(line)
	if firstColumn < 0 {
		firstColumn = 0
	}
	if firstColumn >= len(runes) {
		return ""
	}
	end := firstColumn + count
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[firstColumn:end])
}

func (u *linuxDesktop) renderRemoteEditorOverlay() error {
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	if !state.open {
		state.mu.Unlock()
		return nil
	}
	layout := u.remoteEditLayout()
	text := state.text
	cursor := remoteEditClampCursor(text, state.cursor)
	cursorLine, cursorColumn, _ := remoteEditCursorLineColumn(text, cursor)
	lineHeight := 20
	visibleRows := (layout.editor.bottom - layout.editor.top - 16) / lineHeight
	if visibleRows < 1 {
		visibleRows = 1
	}
	visibleColumns := (layout.editor.right - layout.editor.left - 66) / 7
	if visibleColumns < 8 {
		visibleColumns = 8
	}
	if cursorLine < state.firstLine {
		state.firstLine = cursorLine
	}
	if cursorLine >= state.firstLine+visibleRows {
		state.firstLine = cursorLine - visibleRows + 1
	}
	if cursorColumn < state.firstColumn {
		state.firstColumn = cursorColumn
	}
	if cursorColumn >= state.firstColumn+visibleColumns {
		state.firstColumn = cursorColumn - visibleColumns + 1
	}
	firstLine := state.firstLine
	firstColumn := state.firstColumn
	doc := state.doc
	baseline := state.buffer.Text
	message := state.message
	pending := state.pending
	state.mu.Unlock()

	if err := u.drawPanel(layout.panel); err != nil {
		return err
	}
	words := remoteEditWords(u.language)
	title := strings.ToUpper(words.Title)
	if text != baseline {
		title += "  •  *"
	}
	if err := u.x.text(layout.panel.left+20, layout.panel.top+27, title, premiumTheme.Text, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.text(layout.panel.left+20, layout.panel.top+50, linuxTrimForUI(doc.Path, max(24, (layout.panel.right-layout.panel.left-40)/7)), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	if err := u.x.fillRect(layout.editor.left, layout.editor.top, layout.editor.right-layout.editor.left, layout.editor.bottom-layout.editor.top, premiumTheme.List); err != nil {
		return err
	}
	if err := u.x.strokeRect(layout.editor.left, layout.editor.top, layout.editor.right-layout.editor.left, layout.editor.bottom-layout.editor.top, premiumTheme.Border); err != nil {
		return err
	}

	lines := strings.Split(text, "\n")
	for row := 0; row < visibleRows; row++ {
		lineIndex := firstLine + row
		if lineIndex >= len(lines) {
			break
		}
		y := layout.editor.top + 22 + row*lineHeight
		lineNumber := fmt.Sprintf("%4d", lineIndex+1)
		if err := u.x.text(layout.editor.left+8, y, lineNumber, premiumTheme.Muted, premiumTheme.List); err != nil {
			return err
		}
		segment := remoteEditVisibleSegment(lines[lineIndex], firstColumn, visibleColumns)
		if err := u.x.text(layout.editor.left+54, y, segment, premiumTheme.Text, premiumTheme.List); err != nil {
			return err
		}
	}
	if cursorLine >= firstLine && cursorLine < firstLine+visibleRows {
		caretX := layout.editor.left + 54 + (cursorColumn-firstColumn)*7
		caretY := layout.editor.top + 8 + (cursorLine-firstLine)*lineHeight
		if caretX >= layout.editor.left+54 && caretX < layout.editor.right-2 {
			if err := u.x.fillRect(caretX, caretY, 2, 16, premiumTheme.Accent); err != nil {
				return err
			}
		}
	}

	footerY := layout.editor.bottom + 24
	footerText := words.Hint
	if message != "" {
		footerText = message
	}
	if err := u.x.text(layout.panel.left+20, footerY, linuxTrimForUI(footerText, max(24, (layout.panel.right-layout.panel.left-380)/7)), premiumTheme.Muted, premiumTheme.Panel); err != nil {
		return err
	}
	enabled := !u.busy && pending == linuxRemoteEditPendingNone
	if err := u.drawButton(layout.save, words.Save, enabled, true); err != nil {
		return err
	}
	if err := u.drawButton(layout.reload, words.Reload, enabled, false); err != nil {
		return err
	}
	return u.drawButton(layout.close, words.Close, enabled, false)
}

func (u *linuxDesktop) handleRemoteEditorMouse(x, y int) bool {
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	open := state.open
	state.mu.Unlock()
	if !open {
		return false
	}
	layout := u.remoteEditLayout()
	if layout.save.contains(x, y) {
		u.remoteEditSave()
		return true
	}
	if layout.reload.contains(x, y) {
		u.remoteEditReload()
		return true
	}
	if layout.close.contains(x, y) {
		u.remoteEditClose()
		return true
	}
	if layout.editor.contains(x, y) && !u.busy {
		state.mu.Lock()
		lineHeight := 20
		line := state.firstLine + max(0, (y-layout.editor.top-8)/lineHeight)
		column := state.firstColumn + max(0, (x-layout.editor.left-54)/7)
		if start, end, ok := remoteEditLineBounds(state.text, line); ok {
			state.cursor = start + remoteEditByteAtColumn(state.text[start:end], column)
			state.cursor = remoteEditClampCursor(state.text, state.cursor)
		}
		state.mu.Unlock()
	}
	return true
}

func linuxRemoteEditKeysymText(sym uint32) (string, bool) {
	switch {
	case sym >= 0x20 && sym <= 0x7e:
		return string(rune(sym)), true
	case sym >= 0xa0 && sym <= 0xff:
		return string(rune(sym)), true
	case sym&0xff000000 == 0x01000000:
		r := rune(sym & 0x00ffffff)
		if r >= 0x20 && utf8.ValidRune(r) {
			return string(r), true
		}
	}
	return "", false
}

func (u *linuxDesktop) remoteEditInsert(value string) {
	if value == "" || u.busy {
		return
	}
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	defer state.mu.Unlock()
	if !state.open || state.pending != linuxRemoteEditPendingNone {
		return
	}
	cursor := remoteEditClampCursor(state.text, state.cursor)
	if len(state.text)+len(value) > int(api.MaxRemoteEditBytes) {
		state.message = remoteEditWords(u.language).Unsupported
		return
	}
	state.text = state.text[:cursor] + value + state.text[cursor:]
	state.cursor = cursor + len(value)
	state.closeConfirmUntil = time.Time{}
	state.reloadConfirmUntil = time.Time{}
}

func (u *linuxDesktop) handleRemoteEditKey(sym uint32, keyState uint16) bool {
	state := linuxRemoteEditStateFor(u)
	state.mu.Lock()
	open := state.open
	state.mu.Unlock()
	if !open {
		if keyState&x11ControlMask != 0 && (sym == 'e' || sym == 'E') && !u.settingsOpen && u.promptKind == linuxPromptNone && u.pendingFingerprint == "" {
			u.openSelectedRemoteEditor()
			return true
		}
		return false
	}
	if u.busy {
		return true
	}
	if keyState&x11ControlMask != 0 {
		switch sym {
		case 's', 'S':
			u.remoteEditSave()
		case 'r', 'R':
			u.remoteEditReload()
		}
		return true
	}

	switch sym {
	case x11KeyEscape:
		u.remoteEditClose()
		return true
	case x11KeyReturn:
		u.remoteEditInsert("\n")
		return true
	case x11KeyTab:
		u.remoteEditInsert("\t")
		return true
	case x11KeyLeft:
		state.mu.Lock()
		cursor := remoteEditClampCursor(state.text, state.cursor)
		if cursor > 0 {
			_, size := utf8.DecodeLastRuneInString(state.text[:cursor])
			state.cursor = cursor - size
		}
		state.mu.Unlock()
		return true
	case x11KeyRight:
		state.mu.Lock()
		cursor := remoteEditClampCursor(state.text, state.cursor)
		if cursor < len(state.text) {
			_, size := utf8.DecodeRuneInString(state.text[cursor:])
			state.cursor = cursor + size
		}
		state.mu.Unlock()
		return true
	case x11KeyBackSpace:
		state.mu.Lock()
		cursor := remoteEditClampCursor(state.text, state.cursor)
		if cursor > 0 {
			_, size := utf8.DecodeLastRuneInString(state.text[:cursor])
			start := cursor - size
			state.text = state.text[:start] + state.text[cursor:]
			state.cursor = start
		}
		state.mu.Unlock()
		return true
	case x11KeyDelete:
		state.mu.Lock()
		cursor := remoteEditClampCursor(state.text, state.cursor)
		if cursor < len(state.text) {
			_, size := utf8.DecodeRuneInString(state.text[cursor:])
			state.text = state.text[:cursor] + state.text[cursor+size:]
		}
		state.mu.Unlock()
		return true
	case x11KeyUp, x11KeyDown:
		state.mu.Lock()
		line, column, _ := remoteEditCursorLineColumn(state.text, state.cursor)
		targetLine := line - 1
		if sym == x11KeyDown {
			targetLine = line + 1
		}
		if start, end, ok := remoteEditLineBounds(state.text, targetLine); ok {
			state.cursor = start + remoteEditByteAtColumn(state.text[start:end], column)
		}
		state.mu.Unlock()
		return true
	case 0xff50: // Home
		state.mu.Lock()
		_, _, start := remoteEditCursorLineColumn(state.text, state.cursor)
		state.cursor = start
		state.mu.Unlock()
		return true
	case 0xff57: // End
		state.mu.Lock()
		line, _, _ := remoteEditCursorLineColumn(state.text, state.cursor)
		if _, end, ok := remoteEditLineBounds(state.text, line); ok {
			state.cursor = end
		}
		state.mu.Unlock()
		return true
	}
	if text, ok := linuxRemoteEditKeysymText(sym); ok {
		u.remoteEditInsert(text)
	}
	return true
}
