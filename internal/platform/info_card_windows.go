//go:build windows

package platform

import (
	"sync"
	"syscall"
	"unsafe"
)

const infoCardIDClose = 3201

type infoCardState struct {
	closed bool
}

var (
	infoCardStates sync.Map
	infoCardOnce   sync.Once
	infoCardClass  = "GhostFTP.InfoCardDialog"
	infoCardProc   = syscall.NewCallback(infoCardWndProc)
)

func infoCardWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if v, ok := infoCardStates.Load(hwnd); ok {
		state := v.(*infoCardState)
		switch message {
		case promptWMCommand:
			if int(wParam&0xffff) == infoCardIDClose {
				promptDestroyWindow.Call(hwnd)
				return 0
			}
		case premiumWMCtlColorEdit, premiumWMCtlColorListBox, premiumWMCtlColorBtn, premiumWMCtlColorStatic:
			return premiumDialogControlColor(wParam)
		case promptWMClose:
			promptDestroyWindow.Call(hwnd)
			return 0
		case promptWMDestroy:
			// Only the main desktop window owns WM_QUIT. Closing About or another
			// informational card must return to the main message loop, not end it.
			state.closed = true
			return 0
		}
	}
	r, _, _ := promptDefWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}

// InfoCardDialog displays application-owned informational content in the same
// Light/Dark native shell as the rest of Ghost FTP. Unlike TaskDialog, it does
// not depend on the current Windows system theme to choose its client surface.
func InfoCardDialog(title, heading, body, closeLabel string) {
	infoCardDialog(title, heading, body, closeLabel, false)
}

// CompactInfoDialog is used for concise application-owned diagnostics/status
// content. It keeps the same themed shell without forcing short information into
// the larger About-card geometry.
func CompactInfoDialog(title, heading, body, closeLabel string) {
	infoCardDialog(title, heading, body, closeLabel, true)
}

func infoCardDialog(title, heading, body, closeLabel string, compact bool) {
	if closeLabel == "" {
		closeLabel = "OK"
	}
	hinst, _, _ := promptGetModuleHandleW.Call(0)
	infoCardOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    infoCardProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: premiumDialogBackgroundBrush(),
			ClassName:  promptWstr(infoCardClass),
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped    = 0x00C80000
		wsChild         = 0x40000000
		wsVisible       = 0x10000000
		wsTabStop       = 0x00010000
		bsDefPushButton = 0x00000001
		ssEtchedHorz    = 0x00000010
		// Keep these canonical About client dimensions explicit. Existing release
		// regression coverage protects the 760x460 multiline-heading geometry.
		windowWidth  = 760
		windowHeight = 460
	)

	clientWidth := windowWidth
	clientHeight := windowHeight
	if compact {
		clientWidth = 560
		clientHeight = 300
	}
	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	outerWidth, outerHeight := premiumDialogOuterSize(clientWidth, clientHeight, wsOverlapped, 0, dpi)
	x, y := premiumDialogPosition(owner, outerWidth, outerHeight)
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(infoCardClass))),
		uintptr(unsafe.Pointer(promptWstr(title))),
		wsOverlapped,
		uintptr(x), uintptr(y), uintptr(outerWidth), uintptr(outerHeight),
		owner, 0, hinst, 0,
	)
	if hwnd == 0 {
		return
	}
	applyPremiumDialogWindow(hwnd)
	state := &infoCardState{}
	infoCardStates.Store(hwnd, state)
	defer infoCardStates.Delete(hwnd)
	restoreOwner := premiumModalOwner(owner)
	defer restoreOwner()

	bodyFont := premiumDialogFontForDPI(-15, dpi, 400)
	if bodyFont != 0 {
		defer promptDeleteObject.Call(bodyFont)
	}
	headingHeight := int32(-24)
	if compact {
		headingHeight = -20
	}
	headingFont := premiumDialogFontForDPI(headingHeight, dpi, 600)
	if headingFont != 0 {
		defer promptDeleteObject.Call(headingFont)
	}

	scale := func(value int) uintptr { return uintptr(premiumScale(value, dpi)) }
	makeControl := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(wsChild|wsVisible|style),
			scale(x), scale(y), scale(w), scale(h),
			hwnd, uintptr(id), hinst, 0,
		)
		if child != 0 && controlFont != 0 {
			promptSendMessageW.Call(child, promptWMSetFont, controlFont, 1)
		}
		if child != 0 {
			applyPremiumDialogControl(child, class)
		}
		return child
	}

	var closeButton uintptr
	if compact {
		makeControl("STATIC", heading, 0, 32, 24, 496, 44, 0, headingFont)
		makeControl("STATIC", body, 0, 32, 82, 496, 126, 0, bodyFont)
		makeControl("STATIC", "", ssEtchedHorz, 32, 220, 496, 2, 0, bodyFont)
		closeButton = makeControl("BUTTON", closeLabel, wsTabStop|bsDefPushButton, 424, 238, 104, 38, infoCardIDClose, bodyFont)
	} else {
		// About headings vary substantially across the 24 runtime locales. Reserve
		// enough room for a wrapped three-line heading instead of clipping it.
		makeControl("STATIC", heading, 0, 40, 26, 680, 92, 0, headingFont)
		makeControl("STATIC", body, 0, 40, 132, 680, 220, 0, bodyFont)
		makeControl("STATIC", "", ssEtchedHorz, 40, 366, 680, 2, 0, bodyFont)
		closeButton = makeControl("BUTTON", closeLabel, wsTabStop|bsDefPushButton, 616, 382, 104, 38, infoCardIDClose, bodyFont)
	}
	if closeButton != 0 {
		promptSetFocus.Call(closeButton)
	}

	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)

	var message promptMsg
	for !state.closed {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&message)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		promptTranslateMessage.Call(uintptr(unsafe.Pointer(&message)))
		promptDispatchMessageW.Call(uintptr(unsafe.Pointer(&message)))
	}
}
