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
	promptClass  = "Brendigo.ByFTP.PromptDialog"
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

// PromptDialog displays a small native edit dialog using English action labels.
// Localized UI call sites should use PromptDialogWithLabels.
func PromptDialog(title, instruction, defaultValue string) (string, bool) {
	return PromptDialogWithLabels(title, instruction, defaultValue, "OK", "Cancel")
}

// PromptDialogWithLabels displays a native edit dialog whose action labels are
// supplied by the caller. Keeping the labels outside the platform package lets
// the application use one selected runtime locale without introducing a second
// localization system in the Win32 helper layer.
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

	const wsOverlapped = 0x00C80000
	const wsVisible = 0x10000000
	const wsChild = 0x40000000
	const wsTabStop = 0x00010000
	const wsBorder = 0x00800000
	state := &promptState{}
	hwnd, _, _ := promptCreateWindowExW.Call(
		0, uintptr(unsafe.Pointer(promptWstr(promptClass))), uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped|wsVisible, 420, 280, 520, 185, 0, 0, hinst, 0,
	)
	if hwnd == 0 {
		return "", false
	}
	state.hwnd = hwnd
	promptStates.Store(hwnd, state)
	defer promptStates.Delete(hwnd)

	fontHeight := int32(-15)
	font, _, _ := promptCreateFontW.Call(uintptr(uint32(fontHeight)), 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(promptWstr("Segoe UI"))))
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	mk := func(class, text string, style uint32, x, y, w, h, id int) uintptr {
		ch, _, _ := promptCreateWindowExW.Call(0, uintptr(unsafe.Pointer(promptWstr(class))), uintptr(unsafe.Pointer(promptWstr(text))), uintptr(wsChild|wsVisible|style), uintptr(x), uintptr(y), uintptr(w), uintptr(h), hwnd, uintptr(id), hinst, 0)
		if ch != 0 && font != 0 {
			promptSendMessageW.Call(ch, promptWMSetFont, font, 1)
		}
		return ch
	}
	mk("STATIC", instruction, 0, 18, 16, 470, 40, 0)
	state.edit = mk("EDIT", defaultValue, wsBorder|wsTabStop|0x0080, 18, 62, 470, 28, promptIDEdit)
	if state.edit != 0 {
		promptSendMessageW.Call(state.edit, promptEMSetLimitText, 1024, 0)
	}
	mk("BUTTON", okLabel, wsTabStop, 310, 108, 86, 30, promptIDOK)
	mk("BUTTON", cancelLabel, wsTabStop, 402, 108, 86, 30, promptIDCancel)
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
