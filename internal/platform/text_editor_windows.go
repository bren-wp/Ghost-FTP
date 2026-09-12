//go:build windows

package platform

import (
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	textEditorWMCommand      = 0x0111
	textEditorWMClose        = 0x0010
	textEditorWMDestroy      = 0x0002
	textEditorWMKeyDown      = 0x0100
	textEditorWMSetFont      = 0x0030
	textEditorEMSetLimitText = 0x00C5
	textEditorIDEdit         = 1601
	textEditorIDSave         = 1
	textEditorIDClose        = 2
	textEditorIDReload       = 1602
	textEditorVKControl      = 0x11
)

type TextEditorDialogConfig struct {
	Title       string
	Path        string
	Text        string
	Hint        string
	SaveLabel   string
	ReloadLabel string
	CloseLabel  string
}

type TextEditorDialogResult struct {
	Action string
	Text   string
}

type textEditorState struct {
	hwnd   uintptr
	edit   uintptr
	action string
	text   string
	closed bool
}

var (
	textEditorStates       sync.Map
	textEditorOnce         sync.Once
	textEditorClass        = "GhostFTP.RemoteTextEditor"
	textEditorProc         = syscall.NewCallback(textEditorWndProc)
	textEditorGetKeyState  = user32.NewProc("GetKeyState")
	textEditorLoadCursorW  = user32.NewProc("LoadCursorW")
	textEditorRegister     = user32.NewProc("RegisterClassExW")
	textEditorCreateWindow = user32.NewProc("CreateWindowExW")
	textEditorDestroy      = user32.NewProc("DestroyWindow")
	textEditorShow         = user32.NewProc("ShowWindow")
	textEditorUpdate       = user32.NewProc("UpdateWindow")
)

func textEditorReadText(hwnd uintptr) string {
	n, _, _ := promptGetWindowTextLengthW.Call(hwnd)
	if n > 4<<20 {
		n = 4 << 20
	}
	buf := make([]uint16, int(n)+2)
	if len(buf) == 0 {
		return ""
	}
	promptGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	text := syscall.UTF16ToString(buf)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n")
}

func textEditorFinish(s *textEditorState, action string) {
	if s == nil || s.closed {
		return
	}
	s.text = textEditorReadText(s.edit)
	s.action = action
	textEditorDestroy.Call(s.hwnd)
}

func textEditorWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if value, ok := textEditorStates.Load(hwnd); ok {
		s := value.(*textEditorState)
		switch message {
		case textEditorWMCommand:
			switch int(wParam & 0xffff) {
			case textEditorIDSave:
				textEditorFinish(s, "save")
				return 0
			case textEditorIDReload:
				textEditorFinish(s, "reload")
				return 0
			case textEditorIDClose:
				textEditorFinish(s, "close")
				return 0
			}
		case premiumWMCtlColorEdit, premiumWMCtlColorListBox, premiumWMCtlColorBtn, premiumWMCtlColorStatic:
			return premiumDialogControlColor(wParam)
		case textEditorWMClose:
			textEditorFinish(s, "close")
			return 0
		case textEditorWMDestroy:
			s.closed = true
			return 0
		}
	}
	r, _, _ := promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

func textEditorShortcut(message *promptMsg, state *textEditorState) bool {
	if message == nil || state == nil || message.Message != textEditorWMKeyDown {
		return false
	}
	control, _, _ := textEditorGetKeyState.Call(textEditorVKControl)
	if int16(control&0xffff) >= 0 {
		return false
	}
	switch message.WParam {
	case 'S':
		textEditorFinish(state, "save")
		return true
	case 'R':
		textEditorFinish(state, "reload")
		return true
	}
	return false
}

func TextEditorDialog(config TextEditorDialogConfig) TextEditorDialogResult {
	if config.SaveLabel == "" {
		config.SaveLabel = "Save"
	}
	if config.ReloadLabel == "" {
		config.ReloadLabel = "Reload"
	}
	if config.CloseLabel == "" {
		config.CloseLabel = "Close"
	}

	hinst, _, _ := promptGetModuleHandleW.Call(0)
	textEditorOnce.Do(func() {
		cursor, _, _ := textEditorLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    textEditorProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: premiumDialogBackgroundBrush(),
			ClassName:  promptWstr(textEditorClass),
		}
		textEditorRegister.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped    = 0x00C80000
		wsVisible       = 0x10000000
		wsChild         = 0x40000000
		wsTabStop       = 0x00010000
		wsBorder        = 0x00800000
		wsVScroll       = 0x00200000
		wsHScroll       = 0x00100000
		esMultiline     = 0x0004
		esAutoVScroll   = 0x0040
		esAutoHScroll   = 0x0080
		esWantReturn    = 0x1000
		bsDefPushButton = 0x00000001
		ssEtchedHorz    = 0x00000010
		clientWidth     = 920
		clientHeight    = 660
	)
	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	windowWidth, windowHeight := premiumDialogOuterSize(clientWidth, clientHeight, wsOverlapped, 0, dpi)
	x, y := premiumDialogPosition(owner, windowWidth, windowHeight)
	hwnd, _, _ := textEditorCreateWindow.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(textEditorClass))),
		uintptr(unsafe.Pointer(promptWstr(config.Title))),
		wsOverlapped,
		uintptr(x), uintptr(y), uintptr(windowWidth), uintptr(windowHeight),
		owner, 0, hinst, 0,
	)
	if hwnd == 0 {
		return TextEditorDialogResult{Action: "close", Text: config.Text}
	}
	applyPremiumDialogWindow(hwnd)
	state := &textEditorState{hwnd: hwnd, action: "close", text: config.Text}
	textEditorStates.Store(hwnd, state)
	defer textEditorStates.Delete(hwnd)
	restoreOwner := premiumModalOwner(owner)
	defer restoreOwner()

	font := premiumDialogFontForDPI(-14, dpi, 400)
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	monoName := promptWstr("Consolas")
	monoHeight := int32(-premiumScale(15, dpi))
	monoFont, _, _ := promptCreateFontW.Call(
		uintptr(uint32(monoHeight)), 0, 0, 0, 400, 0, 0, 0,
		1, 0, 0, 5, 0, uintptr(unsafe.Pointer(monoName)),
	)
	if monoFont != 0 {
		defer promptDeleteObject.Call(monoFont)
	} else {
		monoFont = font
	}

	scale := func(value int) uintptr { return uintptr(premiumScale(value, dpi)) }
	mk := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		child, _, _ := textEditorCreateWindow.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(wsChild|wsVisible|style),
			scale(x), scale(y), scale(w), scale(h),
			hwnd, uintptr(id), hinst, 0,
		)
		if child != 0 && controlFont != 0 {
			promptSendMessageW.Call(child, textEditorWMSetFont, controlFont, 1)
		}
		if child != 0 {
			applyPremiumDialogControl(child, class)
		}
		return child
	}

	mk("STATIC", config.Path, 0, 24, 18, 872, 22, 0, font)
	mk("STATIC", config.Hint, 0, 24, 42, 872, 22, 0, font)
	state.edit = mk(
		"EDIT",
		strings.ReplaceAll(config.Text, "\n", "\r\n"),
		wsBorder|wsTabStop|wsVScroll|wsHScroll|esMultiline|esAutoVScroll|esAutoHScroll|esWantReturn,
		24, 72, 872, 500, textEditorIDEdit, monoFont,
	)
	if state.edit != 0 {
		promptSendMessageW.Call(state.edit, textEditorEMSetLimitText, 4<<20, 0)
	}
	mk("STATIC", "", ssEtchedHorz, 24, 588, 872, 2, 0, font)
	mk("BUTTON", config.SaveLabel, wsTabStop|bsDefPushButton, 576, 606, 100, 36, textEditorIDSave, font)
	mk("BUTTON", config.ReloadLabel, wsTabStop, 686, 606, 100, 36, textEditorIDReload, font)
	mk("BUTTON", config.CloseLabel, wsTabStop, 796, 606, 100, 36, textEditorIDClose, font)

	promptSetFocus.Call(state.edit)
	textEditorShow.Call(hwnd, 5)
	textEditorUpdate.Call(hwnd)

	var message promptMsg
	for !state.closed {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if !premiumDialogMessageAvailable(r, &message) {
			break
		}
		if textEditorShortcut(&message, state) {
			continue
		}
		handled, _, _ := premiumIsDialogMessageW.Call(hwnd, uintptr(unsafe.Pointer(&message)))
		if handled != 0 {
			continue
		}
		promptTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		promptDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
	return TextEditorDialogResult{Action: state.action, Text: state.text}
}
