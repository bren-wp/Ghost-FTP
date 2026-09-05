//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	promptWMCreate       = 0x0001
	promptWMDestroy      = 0x0002
	promptWMClose        = 0x0010
	promptWMCommand      = 0x0111
	promptWMSetFont      = 0x0030
	promptEMSetLimitText = 0x00C5
	promptIDEdit         = 1001
	promptIDOK           = 1
	promptIDCancel       = 2
)

var (
	promptRegisterClassExW     = user32.NewProc("RegisterClassExW")
	promptCreateWindowExW      = user32.NewProc("CreateWindowExW")
	promptDefWindowProcW       = user32.NewProc("DefWindowProcW")
	promptDestroyWindow        = user32.NewProc("DestroyWindow")
	promptShowWindow           = user32.NewProc("ShowWindow")
	promptUpdateWindow         = user32.NewProc("UpdateWindow")
	promptGetMessageW          = user32.NewProc("GetMessageW")
	promptTranslateMessage     = user32.NewProc("TranslateMessage")
	promptDispatchMessageW     = user32.NewProc("DispatchMessageW")
	promptPostQuitMessage      = user32.NewProc("PostQuitMessage")
	promptGetWindowTextLengthW = user32.NewProc("GetWindowTextLengthW")
	promptGetWindowTextW       = user32.NewProc("GetWindowTextW")
	promptSendMessageW         = user32.NewProc("SendMessageW")
	promptSetFocus             = user32.NewProc("SetFocus")
	promptLoadCursorW          = user32.NewProc("LoadCursorW")
	promptGetModuleHandleW     = syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW")
	promptCreateFontW          = syscall.NewLazyDLL("gdi32.dll").NewProc("CreateFontW")
	promptDeleteObject         = syscall.NewLazyDLL("gdi32.dll").NewProc("DeleteObject")
)

type promptWndClassEx struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type promptPoint struct{ X, Y int32 }
type promptMsg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      promptPoint
	Private uint32
}

type promptState struct {
	hwnd     uintptr
	edit     uintptr
	value    string
	accepted bool
}

var (
	promptStates sync.Map
	promptOnce   sync.Once
	promptClass  = "GhostFTP.PromptDialog"
	promptProc   = syscall.NewCallback(promptWndProc)
)

func promptWstr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func promptText(hwnd uintptr) string {
	n, _, _ := promptGetWindowTextLengthW.Call(hwnd)
	buf := make([]uint16, int(n)+2)
	if len(buf) == 0 {
		return ""
	}
	promptGetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return syscall.UTF16ToString(buf)
}

func promptWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if v, ok := promptStates.Load(hwnd); ok {
		s := v.(*promptState)
		switch message {
		case promptWMCommand:
			id := int(wParam & 0xffff)
			if id == promptIDOK {
				s.value = promptText(s.edit)
				s.accepted = true
				promptDestroyWindow.Call(hwnd)
				return 0
			}
			if id == promptIDCancel {
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

// PromptDialog displays a native edit dialog using English action labels.
// Localized UI call sites should use PromptDialogWithLabels.
func PromptDialog(title, instruction, defaultValue string) (string, bool) {
	return PromptDialogWithLabels(title, instruction, defaultValue, "OK", "Cancel")
}

// PromptDialogWithLabels keeps the platform layer dependency-free while using
// the same premium native shell as Setup. Action labels remain caller-owned so
// the application can use its selected locale without a second i18n system.
func PromptDialogWithLabels(title, instruction, defaultValue, okLabel, cancelLabel string) (string, bool) {
	if okLabel == "" {
		okLabel = "OK"
	}
	if cancelLabel == "" {
		cancelLabel = "Cancel"
	}
	hinst, _, _ := promptGetModuleHandleW.Call(0)
	promptOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    promptProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: 6,
			ClassName:  promptWstr(promptClass),
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped    = 0x00C80000
		wsVisible       = 0x10000000
		wsChild         = 0x40000000
		wsTabStop       = 0x00010000
		wsBorder        = 0x00800000
		bsDefPushButton = 0x00000001
		ssEtchedHorz    = 0x00000010
	)
	const (
		windowWidth  = 600
		windowHeight = 226
	)
	x, y := premiumDialogPosition(windowWidth, windowHeight)
	state := &promptState{}
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(promptClass))),
		uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped,
		uintptr(x), uintptr(y), windowWidth, windowHeight,
		0, 0, hinst, 0,
	)
	if hwnd == 0 {
		return "", false
	}
	applyPremiumDialogWindow(hwnd)
	state.hwnd = hwnd
	promptStates.Store(hwnd, state)
	defer promptStates.Delete(hwnd)

	font := premiumDialogFont(-15, 400)
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	captionFont := premiumDialogFont(-13, 400)
	if captionFont != 0 {
		defer promptDeleteObject.Call(captionFont)
	}

	mk := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		ch, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(wsChild|wsVisible|style),
			uintptr(x), uintptr(y), uintptr(w), uintptr(h),
			hwnd, uintptr(id), hinst, 0,
		)
		if ch != 0 && controlFont != 0 {
			promptSendMessageW.Call(ch, promptWMSetFont, controlFont, 1)
		}
		return ch
	}

	mk("STATIC", instruction, 0, 28, 22, 544, 42, 0, font)
	state.edit = mk("EDIT", defaultValue, wsBorder|wsTabStop|0x0080, 28, 72, 544, 32, promptIDEdit, font)
	if state.edit != 0 {
		promptSendMessageW.Call(state.edit, promptEMSetLimitText, 1024, 0)
	}
	mk("STATIC", "", ssEtchedHorz, 28, 120, 544, 2, 0, captionFont)
	mk("STATIC", "Ghost FTP · local native dialog", 0, 28, 138, 300, 26, 0, captionFont)
	mk("BUTTON", okLabel, wsTabStop|bsDefPushButton, 374, 134, 94, 36, promptIDOK, font)
	mk("BUTTON", cancelLabel, wsTabStop, 478, 134, 94, 36, promptIDCancel, font)

	promptSetFocus.Call(state.edit)
	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)

	var m promptMsg
	for {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		promptTranslateMessage.Call(uintptr(unsafe.Pointer(&m)))
		promptDispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	return state.value, state.accepted
}
