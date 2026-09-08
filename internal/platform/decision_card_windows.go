//go:build windows

package platform

import (
	"strings"
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
	decisionIDYes          = 6 // IDYES
	decisionIDNo           = 7 // IDNO
	decisionSSNoPrefix     = 0x00000080
	decisionSSEditControl  = 0x00002000
	decisionEtchedHorz     = 0x00000010
	decisionWSChild        = 0x40000000
	decisionWSVisible      = 0x10000000
	decisionWSTabStop      = 0x00010000
	decisionDefButton      = 0x00000001
	decisionClientWidth    = 680
	decisionHeadingY       = 28
	decisionHeadingMinH    = 58
	decisionHeadingMaxH    = 126
	decisionBodyMinH       = 126
	decisionBodyMaxH       = 280
	decisionButtonH        = 38
	decisionBottomPadding  = 28
)

type decisionCardState struct {
	kind    int
	heading uintptr
	result  int
	closed  bool
}

type decisionCardLayout struct {
	clientWidth   int
	clientHeight  int
	headingY      int
	headingHeight int
	bodyY         int
	bodyHeight    int
	dividerY      int
	buttonY       int
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

func decisionCardTextUnits(text string) int {
	units := 0
	for _, r := range text {
		switch {
		case r == '\t':
			units += 4
		case r > 0x07ff:
			// CJK, Japanese, Korean and Indic scripts generally consume more
			// horizontal space than Latin/Cyrillic glyphs at the same font size.
			units += 2
		default:
			units++
		}
	}
	return units
}

func decisionCardEstimatedLines(text string, unitsPerLine int) int {
	if unitsPerLine < 1 {
		unitsPerLine = 1
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	paragraphs := strings.Split(text, "\n")
	lines := 0
	for _, paragraph := range paragraphs {
		units := decisionCardTextUnits(strings.TrimSpace(paragraph))
		if units == 0 {
			lines++
			continue
		}
		lines += (units + unitsPerLine - 1) / unitsPerLine
	}
	if lines < 1 {
		return 1
	}
	return lines
}

func decisionCardClamp(value, minimum, maximum int) int {
	if value < minimum {
		return minimum
	}
	if value > maximum {
		return maximum
	}
	return value
}

// decisionCardLayoutForText preserves the original compact 680x320 surface for
// short messages, but expands vertically for longer localized/security text.
// The estimator is intentionally conservative for wide-script glyphs so static
// controls receive enough height before Win32 performs its final word wrapping.
func decisionCardLayoutForText(instruction, content string) decisionCardLayout {
	headingHeight := decisionCardClamp(decisionCardEstimatedLines(instruction, 28)*31, decisionHeadingMinH, decisionHeadingMaxH)
	bodyHeight := decisionCardClamp(decisionCardEstimatedLines(content, 42)*23, decisionBodyMinH, decisionBodyMaxH)
	bodyY := decisionHeadingY + headingHeight + 10
	dividerY := bodyY + bodyHeight + 16
	buttonY := dividerY + 16
	clientHeight := buttonY + decisionButtonH + decisionBottomPadding
	return decisionCardLayout{
		clientWidth:   decisionClientWidth,
		clientHeight:  clientHeight,
		headingY:      decisionHeadingY,
		headingHeight: headingHeight,
		bodyY:         bodyY,
		bodyHeight:    bodyHeight,
		dividerY:      dividerY,
		buttonY:       buttonY,
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

	const wsOverlapped = 0x00C80000
	layout := decisionCardLayoutForText(instruction, content)
	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	windowWidth, windowHeight := premiumDialogOuterSize(layout.clientWidth, layout.clientHeight, wsOverlapped, 0, dpi)
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

	textStyle := uint32(decisionSSNoPrefix | decisionSSEditControl)
	state.heading = makeControl("STATIC", instruction, textStyle, 36, layout.headingY, 608, layout.headingHeight, 0, headingFont)
	makeControl("STATIC", content, textStyle, 36, layout.bodyY, 608, layout.bodyHeight, 0, bodyFont)
	makeControl("STATIC", "", decisionEtchedHorz, 36, layout.dividerY, 608, 2, 0, bodyFont)

	if kind == decisionCardKindConfirm {
		_, _, yesLabel, noLabel := resolvedDialogLabels()
		yesButton := makeControl("BUTTON", yesLabel, decisionWSTabStop|decisionDefButton, 430, layout.buttonY, 102, decisionButtonH, decisionIDYes, bodyFont)
		makeControl("BUTTON", noLabel, decisionWSTabStop, 542, layout.buttonY, 102, decisionButtonH, decisionIDNo, bodyFont)
		if yesButton != 0 {
			promptSetFocus.Call(yesButton)
		}
	} else {
		okLabel, _, _, _ := resolvedDialogLabels()
		okButton := makeControl("BUTTON", okLabel, decisionWSTabStop|decisionDefButton, 542, layout.buttonY, 102, decisionButtonH, promptIDOK, bodyFont)
		if okButton != 0 {
			promptSetFocus.Call(okButton)
		}
	}

	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)
	premiumRunDialogLoop(hwnd, func() bool { return state.closed })
	return state.result, true
}
