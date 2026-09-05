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
	languageClass  = "GhostFTP.OptionDialog"
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

// SelectOptionDialog shows a bounded native Windows selector with no framework
// dependency. The visual shell is shared by Setup and small option flows while
// the caller still owns all actual validation/security behavior.
func SelectOptionDialog(title, instruction, footer, acceptLabel, cancelLabel string, options []string, defaultIndex int) (int, bool) {
	if len(options) == 0 {
		return 0, false
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}
	if acceptLabel == "" {
		acceptLabel = "Continue"
	}
	if cancelLabel == "" {
		cancelLabel = "Cancel"
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
		wsOverlapped   = 0x00C80000
		wsChild        = 0x40000000
		wsVisible      = 0x10000000
		wsTabStop      = 0x00010000
		wsVScroll      = 0x00200000
		cbsDropdown    = 0x0003
		bsDefPushButton = 0x00000001
		ssEtchedHorz   = 0x00000010
	)
	const (
		windowWidth  = 680
		windowHeight = 342
	)
	x, y := premiumDialogPosition(windowWidth, windowHeight)
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(languageClass))),
		uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped,
		uintptr(x), uintptr(y), windowWidth, windowHeight,
		0, 0, hinst, 0,
	)
	if hwnd == 0 {
		return defaultIndex, false
	}
	applyPremiumDialogWindow(hwnd)

	state := &languageDialogState{selected: defaultIndex}
	languageStates.Store(hwnd, state)
	defer languageStates.Delete(hwnd)

	font, _, _ := promptCreateFontW.Call(uintptr(uint32(int32(-16))), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(promptWstr("Segoe UI"))))
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	headerFont, _, _ := promptCreateFontW.Call(uintptr(uint32(int32(-26))), 0, 0, 0, 600, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(promptWstr("Segoe UI"))))
	if headerFont != 0 {
		defer promptDeleteObject.Call(headerFont)
	}
	captionFont, _, _ := promptCreateFontW.Call(uintptr(uint32(int32(-14))), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(promptWstr("Segoe UI"))))
	if captionFont != 0 {
		defer promptDeleteObject.Call(captionFont)
	}

	makeControl := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(wsChild|wsVisible|style),
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			hwnd, uintptr(id), hinst, 0,
		)
		if child != 0 && controlFont != 0 {
			promptSendMessageW.Call(child, promptWMSetFont, controlFont, 1)
		}
		return child
	}

	makeControl("STATIC", "Ghost FTP", 0, 38, 26, 602, 38, 0, headerFont)
	makeControl("STATIC", "Secure Windows setup", 0, 40, 66, 600, 24, 0, captionFont)
	makeControl("STATIC", instruction, 0, 40, 104, 600, 42, 0, font)

	state.combo = makeControl("COMBOBOX", "", wsTabStop|wsVScroll|cbsDropdown, 40, 154, 600, 270, languageIDCombo, font)
	if state.combo == 0 {
		promptDestroyWindow.Call(hwnd)
		return defaultIndex, false
	}
	for _, option := range options {
		promptSendMessageW.Call(state.combo, languageCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.combo, languageCBSet, uintptr(defaultIndex), 0)

	makeControl("STATIC", "", ssEtchedHorz, 40, 207, 600, 2, 0, font)
	makeControl("STATIC", footer, 0, 40, 222, 386, 38, 0, captionFont)
	makeControl("BUTTON", acceptLabel, wsTabStop|bsDefPushButton, 438, 224, 98, 38, languageIDInstall, font)
	makeControl("BUTTON", cancelLabel, wsTabStop, 544, 224, 96, 38, languageIDCancel, font)

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

// SelectLanguageDialog is the installer-facing wrapper. It intentionally
// accepts display strings so platform code stays independent from i18n data.
func SelectLanguageDialog(title, instruction string, options []string, defaultIndex int) (int, bool) {
	return SelectOptionDialog(
		title,
		instruction,
		"Private by design  ·  No telemetry  ·  FTP / FTPS / SFTP",
		"Continue",
		"Cancel",
		options,
		defaultIndex,
	)
}
