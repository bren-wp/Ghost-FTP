//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	decisionCardKindInfo = iota
	decisionCardKindError
	decisionCardKindConfirm
)

const (
	decisionIDYes = 6 // IDYES
	decisionIDNo  = 7 // IDNO
	decisionSSNoPrefix = 0x00000080
	decisionEtchedHorz = 0x00000010
	decisionWSChild = 0x40000000
	decisionWSVisible = 0x10000000
	decisionWSTabStop = 0x00010000
	decisionDefButton = 0x00000001
)

type decisionCardState struct {
	kind    int
	heading uintptr
	result  int
	closed  bool
}

var (
	decisionCardStates sync.Map
	decisionCardOnce   sync.Once
	decisionCardClass  = "GhostFTP.DecisionCardDialog"
	decisionCardProc   = syscall.NewCallback(decisionCardWndProc)

	decisionLabelsMu sync.RWMutex
	decisionYesLabel = "Yes"
	decisionNoLabel  = "No"
)

// SetDialogDecisionLabels keeps Yes/No actions in the application's selected
// locale without coupling the platform package to the desktop i18n catalog.
func SetDialogDecisionLabels(yesLabel, noLabel string) {
	if yesLabel == "" {
		yesLabel = "Yes"
	}
	if noLabel == "" {
		noLabel = "No"
	}
	decisionLabelsMu.Lock()
	decisionYesLabel = yesLabel
	decisionNoLabel = noLabel
	decisionLabelsMu.Unlock()
}

func dialogDecisionLabels() (string, string) {
	decisionLabelsMu.RLock()
	defer decisionLabelsMu.RUnlock()
	return decisionYesLabel, decisionNoLabel
}

func decisionCardHeadingColor(kind int) uintptr {
	theme := premiumDialogTheme()
	switch kind {
	case decisionCardKindError:
		return premiumPaletteColor(theme.Danger)
	case decisionCardKindInfo:
		return premiumPaletteColor(theme.AccentStrong)
	default:
		return premiumPaletteColor(theme.Text)
	}
}

func decisionCardWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if value, ok := decisionCardStates.Load(hwnd); ok {
		state := value.(*decisionCardState)
		switch message {
		case promptWMCommand:
			id := int(wParam & 0xffff)
			if state.kind == decisionCardKindConfirm {
				switch id {
				case decisionIDYes:
					state.result = decisionIDYes
					promptDestroyWindow.Call(hwnd)
					return 0
				case decisionIDNo, promptIDCancel:
					state.result = decisionIDNo
					promptDestroyWindow.Call(hwnd)
					return 0
				}
			} else if id == promptIDOK || id == promptIDCancel {
				state.result = promptIDOK
				promptDestroyWindow.Call(hwnd)
				return 0
			}
		case premiumWMCtlColorEdit, premiumWMCtlColorListBox, premiumWMCtlColorBtn:
			return premiumDialogControlColor(wParam)
		case premiumWMCtlColorStatic:
			if lParam == state.heading && state.heading != 0 {
				premiumSetTextColor.Call(wParam, decisionCardHeadingColor(state.kind))
				premiumSetBkColor.Call(wParam, premiumDialogSurfaceColor())
				return premiumDialogBackgroundBrush()
			}
			return premiumDialogControlColor(wParam)
		case promptWMClose:
			if state.kind == decisionCardKindConfirm {
				state.result = decisionIDNo
			} else {
				state.result = promptIDOK
			}
			promptDestroyWindow.Call(hwnd)
			return 0
		case promptWMDestroy:
			state.closed = true
			return 0
		}
	}
	r, _, _ := promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

// decisionCardDialog is the application-owned primary path for ConfirmDialog,
// InfoDialog and ErrorDialog. Returning ok=false means Win32 could not create
// the custom surface and the caller should use the stock Windows fallback.
func decisionCardDialog(title, instruction, content string, kind int) (result int, ok bool) {
	hinst, _, _ := promptGetModuleHandleW.Call(0)
	decisionCardOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    decisionCardProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: premiumDialogBackgroundBrush(),
			ClassName:  promptWstr(decisionCardClass),
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped = 0x00C80000
		clientWidth  = 680
		clientHeight = 320
	)
	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	windowWidth, windowHeight := premiumDialogOuterSize(clientWidth, clientHeight, wsOverlapped, 0, dpi)
	x, y := premiumDialogPosition(owner, windowWidth, windowHeight)
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(decisionCardClass))),
		uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped,
		uintptr(x), uintptr(y), uintptr(windowWidth), uintptr(windowHeight),
		owner, 0, hinst, 0,
	)
	if hwnd == 0 {
		return 0, false
	}
	applyPremiumDialogWindow(hwnd)

	state := &decisionCardState{kind: kind}
	if kind == decisionCardKindConfirm {
		state.result = decisionIDNo
	} else {
		state.result = promptIDOK
	}
	decisionCardStates.Store(hwnd, state)
	defer decisionCardStates.Delete(hwnd)
	restoreOwner := premiumModalOwner(owner)
	defer restoreOwner()

	bodyFont := premiumDialogFontForDPI(-15, dpi, 400)
	if bodyFont != 0 {
		defer promptDeleteObject.Call(bodyFont)
	}
	headingFont := premiumDialogFontForDPI(-22, dpi, 600)
	if headingFont != 0 {
		defer promptDeleteObject.Call(headingFont)
	}

	scale := func(value int) uintptr { return uintptr(premiumScale(value, dpi)) }
	makeControl := func(class, text string, style uint32, x, y, width, height, id int, font uintptr) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(decisionWSChild|decisionWSVisible|style),
			scale(x), scale(y), scale(width), scale(height),
			hwnd, uintptr(id), hinst, 0,
		)
		if child != 0 && font != 0 {
			promptSendMessageW.Call(child, promptWMSetFont, font, 1)
		}
		if child != 0 {
			applyPremiumDialogControl(child, class)
		}
		return child
	}

	state.heading = makeControl("STATIC", instruction, decisionSSNoPrefix, 36, 28, 608, 58, 0, headingFont)
	makeControl("STATIC", content, decisionSSNoPrefix, 36, 96, 608, 126, 0, bodyFont)
	makeControl("STATIC", "", decisionEtchedHorz, 36, 238, 608, 2, 0, bodyFont)

	if kind == decisionCardKindConfirm {
		yesLabel, noLabel := dialogDecisionLabels()
		yesButton := makeControl("BUTTON", yesLabel, decisionWSTabStop|decisionDefButton, 430, 254, 102, 38, decisionIDYes, bodyFont)
		makeControl("BUTTON", noLabel, decisionWSTabStop, 542, 254, 102, 38, decisionIDNo, bodyFont)
		if yesButton != 0 {
			promptSetFocus.Call(yesButton)
		}
	} else {
		okLabel, _ := dialogActionLabels()
		okButton := makeControl("BUTTON", okLabel, decisionWSTabStop|decisionDefButton, 542, 254, 102, 38, promptIDOK, bodyFont)
		if okButton != 0 {
			promptSetFocus.Call(okButton)
		}
	}

	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)
	premiumRunDialogLoop(hwnd, func() bool { return state.closed })
	return state.result, true
}
