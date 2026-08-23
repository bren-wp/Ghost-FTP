//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	languageIDCombo   = 2101
	languageIDInstall = 2102
	languageIDCancel  = 2103
	languageCBAdd     = 0x0143
	languageCBGet     = 0x0147
	languageCBSet     = 0x014E
)

type languageDialogState struct {
	combo    uintptr
	selected int
	accepted bool
}

var (
	languageStates sync.Map
	languageOnce   sync.Once
	languageClass  = "ByFTP.LanguageDialog"
	languageProc   = syscall.NewCallback(languageWndProc)
)

func languageWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if v, ok := languageStates.Load(hwnd); ok {
		state := v.(*languageDialogState)
		switch message {
		case promptWMCommand:
			id := int(wParam & 0xffff)
			switch id {
			case languageIDInstall:
				index, _, _ := promptSendMessageW.Call(state.combo, languageCBGet, 0, 0)
				if int32(index) >= 0 {
					state.selected = int(index)
				}
				state.accepted = true
				promptDestroyWindow.Call(hwnd)
				return 0
			case languageIDCancel:
				promptDestroyWindow.Call(hwnd)
				return 0
			}
		case promptWMClose:
			promptDestroyWindow.Call(hwnd)
			return 0
		case promptWMDestroy:
			promptPostQuitMessage.Call(0)
			return 0
		}
	}
	r, _, _ := promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

// SelectLanguageDialog shows a compact native Windows setup-language picker.
// It intentionally accepts display strings from the installer so platform code
// stays independent from the application's localization package.
func SelectLanguageDialog(title, instruction string, options []string, defaultIndex int) (int, bool) {
	if len(options) == 0 {
		return 0, false
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}

	hinst, _, _ := promptGetModuleHandleW.Call(0)
	languageOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    languageProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: 6,
			ClassName:  promptWstr(languageClass),
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped = 0x00C80000
		wsVisible    = 0x10000000
		wsChild      = 0x40000000
		wsTabStop    = 0x00010000
		wsVScroll    = 0x00200000
		cbsDropdown  = 0x0003
	)

	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(languageClass))),
		uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped|wsVisible,
		420, 250, 560, 230,
		0, 0, hinst, 0,
	)
	if hwnd == 0 {
		return defaultIndex, false
	}
	state := &languageDialogState{selected: defaultIndex}
	languageStates.Store(hwnd, state)
	defer languageStates.Delete(hwnd)

	// CreateFontW accepts a signed LONG height. syscall.Proc.Call takes uintptr,
	// so preserve the signed 32-bit Win32 representation at runtime instead of
	// attempting an invalid constant conversion from -16 to uint32.
	fontHeight := int32(-16)
	font, _, _ := promptCreateFontW.Call(uintptr(uint32(fontHeight)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(promptWstr("Segoe UI"))))
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	makeControl := func(class, text string, style uint32, x, y, w, h, id int) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0, uintptr(unsafe.Pointer(promptWstr(class))), uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(wsChild|wsVisible|style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), hwnd, uintptr(id), hinst, 0,
		)
		if child != 0 && font != 0 {
			promptSendMessageW.Call(child, promptWMSetFont, font, 1)
		}
		return child
	}

	makeControl("STATIC", instruction, 0, 24, 22, 500, 42, 0)
	state.combo = makeControl("COMBOBOX", "", wsTabStop|wsVScroll|cbsDropdown, 24, 70, 500, 260, languageIDCombo)
	if state.combo == 0 {
		promptDestroyWindow.Call(hwnd)
		return defaultIndex, false
	}
	for _, option := range options {
		promptSendMessageW.Call(state.combo, languageCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.combo, languageCBSet, uintptr(defaultIndex), 0)
	makeControl("BUTTON", "Install", wsTabStop, 338, 132, 88, 32, languageIDInstall)
	makeControl("BUTTON", "Cancel", wsTabStop, 436, 132, 88, 32, languageIDCancel)

	promptSetFocus.Call(state.combo)
	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)

	var msg promptMsg
	for {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		promptTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		promptDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return state.selected, state.accepted
}
