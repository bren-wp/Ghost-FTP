//go:build windows

package platform

import (
	"strconv"
	"strings"
	"sync"
	"syscall"
	"unsafe"
)

const (
	settingsIDAppearance = 4101
	settingsIDNumberBase = 4110
	settingsIDConflict   = 4120
	settingsIDConfirm    = 4121
	settingsIDApply      = 4122
	settingsIDCancel     = 4123
	settingsIDError      = 4124

	settingsCBAdd       = 0x0143
	settingsCBGet       = 0x0147
	settingsCBSet       = 0x014E
	settingsBMGetCheck  = 0x00F0
	settingsBMSetCheck  = 0x00F1
	settingsBSTChecked  = 1
	settingsEMSetSel    = 0x00B1
	settingsESNumber    = 0x2000
	settingsAutoCheck   = 0x00000003
	settingsWSBorder    = 0x00800000
	settingsWSChild     = 0x40000000
	settingsWSVisible   = 0x10000000
	settingsWSTabStop   = 0x00010000
	settingsWSVScroll   = 0x00200000
	settingsCBSDropList = 0x0003
	settingsDefButton   = 0x00000001
	settingsEtchedHorz  = 0x00000010
)

// SettingsDialogNumber describes one bounded integer preference. InvalidText is
// caller-owned so validation remains localized without coupling platform code to
// Ghost FTP's translation catalog.
type SettingsDialogNumber struct {
	Label       string
	Value       int
	Min         int
	Max         int
	InvalidText string
}

// SettingsDialogConfig contains presentation text and current values for the
// application-owned Windows settings surface. The desktop layer owns all labels
// and business meaning; this package owns only native rendering and interaction.
type SettingsDialogConfig struct {
	Title             string
	Heading           string
	Intro             string
	AppearanceLabel   string
	AppearanceOptions []string
	AppearanceIndex   int
	Numbers           []SettingsDialogNumber
	ConflictLabel     string
	ConflictOptions   []string
	ConflictIndex     int
	ConfirmDelete     string
	ConfirmDeleteOn   bool
	Footer            string
	ApplyLabel        string
	CancelLabel       string
}

type SettingsDialogResult struct {
	AppearanceIndex int
	Numbers         []int
	ConflictIndex   int
	ConfirmDelete   bool
}

type settingsDialogState struct {
	appearance uintptr
	numbers    []uintptr
	conflict   uintptr
	confirm    uintptr
	errorLabel uintptr
	config     SettingsDialogConfig
	result     SettingsDialogResult
	accepted   bool
	closed     bool
}

var (
	settingsStates         sync.Map
	settingsOnce           sync.Once
	settingsClass          = "GhostFTP.SettingsDialog"
	settingsProc           = syscall.NewCallback(settingsWndProc)
	settingsSetWindowTextW = user32.NewProc("SetWindowTextW")
)

func settingsSetText(hwnd uintptr, text string) {
	if hwnd == 0 {
		return
	}
	settingsSetWindowTextW.Call(hwnd, uintptr(unsafe.Pointer(promptWstr(text))))
}

func settingsComboIndex(hwnd uintptr, fallback int) int {
	if hwnd == 0 {
		return fallback
	}
	value, _, _ := promptSendMessageW.Call(hwnd, settingsCBGet, 0, 0)
	if int32(value) < 0 {
		return fallback
	}
	return int(value)
}

func settingsWndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	if v, ok := settingsStates.Load(hwnd); ok {
		state := v.(*settingsDialogState)
		switch message {
		case promptWMCommand:
			id := int(wParam & 0xffff)
			switch id {
			case settingsIDApply:
				values := make([]int, len(state.numbers))
				for index, edit := range state.numbers {
					field := state.config.Numbers[index]
					value, err := strconv.Atoi(strings.TrimSpace(promptText(edit)))
					if err != nil || value < field.Min || value > field.Max {
						message := field.InvalidText
						if message == "" {
							message = field.Label
						}
						settingsSetText(state.errorLabel, message)
						promptSetFocus.Call(edit)
						promptSendMessageW.Call(edit, settingsEMSetSel, 0, ^uintptr(0))
						return 0
					}
					values[index] = value
				}

				state.result.AppearanceIndex = settingsComboIndex(state.appearance, state.config.AppearanceIndex)
				state.result.Numbers = values
				state.result.ConflictIndex = settingsComboIndex(state.conflict, state.config.ConflictIndex)
				checked, _, _ := promptSendMessageW.Call(state.confirm, settingsBMGetCheck, 0, 0)
				state.result.ConfirmDelete = checked == settingsBSTChecked
				state.accepted = true
				promptDestroyWindow.Call(hwnd)
				return 0
			case settingsIDCancel:
				promptDestroyWindow.Call(hwnd)
				return 0
			}
		case premiumWMCtlColorEdit, premiumWMCtlColorListBox, premiumWMCtlColorBtn, premiumWMCtlColorStatic:
			return premiumDialogControlColor(wParam)
		case promptWMClose:
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

func normalizedSettingsIndex(index, count int) int {
	if count <= 0 {
		return 0
	}
	if index < 0 || index >= count {
		return 0
	}
	return index
}

// SettingsDialog presents all frequently changed desktop preferences in one
// bounded modal instead of forcing the user through a chain of independent
// prompts. It is DPI-aware, owner-modal and follows the active Ghost FTP theme.
func SettingsDialog(config SettingsDialogConfig) (SettingsDialogResult, bool) {
	fallback := SettingsDialogResult{
		AppearanceIndex: normalizedSettingsIndex(config.AppearanceIndex, len(config.AppearanceOptions)),
		ConflictIndex:   normalizedSettingsIndex(config.ConflictIndex, len(config.ConflictOptions)),
		ConfirmDelete:   config.ConfirmDeleteOn,
		Numbers:         make([]int, len(config.Numbers)),
	}
	for index, field := range config.Numbers {
		fallback.Numbers[index] = field.Value
	}
	if len(config.AppearanceOptions) == 0 || len(config.ConflictOptions) == 0 || len(config.Numbers) == 0 || len(config.Numbers) > 6 {
		return fallback, false
	}
	if config.ApplyLabel == "" {
		config.ApplyLabel = "OK"
	}
	if config.CancelLabel == "" {
		config.CancelLabel = "Cancel"
	}
	config.AppearanceIndex = fallback.AppearanceIndex
	config.ConflictIndex = fallback.ConflictIndex

	hinst, _, _ := promptGetModuleHandleW.Call(0)
	settingsOnce.Do(func() {
		cursor, _, _ := promptLoadCursorW.Call(0, 32512)
		wc := promptWndClassEx{
			CbSize:     uint32(unsafe.Sizeof(promptWndClassEx{})),
			WndProc:    settingsProc,
			Instance:   hinst,
			Cursor:     cursor,
			Background: premiumDialogBackgroundBrush(),
			ClassName:  promptWstr(settingsClass),
		}
		promptRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	})

	const (
		wsOverlapped = 0x00C80000
		clientWidth  = 760
		clientHeight = 600
	)
	owner := premiumDialogOwner()
	dpi := premiumDialogDPI(owner)
	windowWidth, windowHeight := premiumDialogOuterSize(clientWidth, clientHeight, wsOverlapped, 0, dpi)
	x, y := premiumDialogPosition(owner, windowWidth, windowHeight)
	hwnd, _, _ := promptCreateWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(promptWstr(settingsClass))),
		uintptr(unsafe.Pointer(promptWstr(config.Title))),
		wsOverlapped,
		uintptr(x), uintptr(y), uintptr(windowWidth), uintptr(windowHeight),
		owner, 0, hinst, 0,
	)
	if hwnd == 0 {
		return fallback, false
	}
	applyPremiumDialogWindow(hwnd)

	state := &settingsDialogState{config: config, result: fallback}
	settingsStates.Store(hwnd, state)
	defer settingsStates.Delete(hwnd)
	restoreOwner := premiumModalOwner(owner)
	defer restoreOwner()

	font := premiumDialogFontForDPI(-15, dpi, 400)
	if font != 0 {
		defer promptDeleteObject.Call(font)
	}
	headingFont := premiumDialogFontForDPI(-25, dpi, 600)
	if headingFont != 0 {
		defer promptDeleteObject.Call(headingFont)
	}
	captionFont := premiumDialogFontForDPI(-13, dpi, 400)
	if captionFont != 0 {
		defer promptDeleteObject.Call(captionFont)
	}

	scale := func(value int) uintptr { return uintptr(premiumScale(value, dpi)) }
	makeControl := func(class, text string, style uint32, x, y, w, h, id int, controlFont uintptr) uintptr {
		child, _, _ := promptCreateWindowExW.Call(
			0,
			uintptr(unsafe.Pointer(promptWstr(class))),
			uintptr(unsafe.Pointer(promptWstr(text))),
			uintptr(settingsWSChild|settingsWSVisible|style),
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

	makeControl("STATIC", config.Heading, 0, 36, 22, 688, 38, 0, headingFont)
	makeControl("STATIC", config.Intro, 0, 36, 66, 688, 34, 0, font)
	makeControl("STATIC", config.AppearanceLabel, 0, 36, 112, 688, 22, 0, captionFont)
	state.appearance = makeControl("COMBOBOX", "", settingsWSTabStop|settingsWSVScroll|settingsCBSDropList, 36, 137, 688, 240, settingsIDAppearance, font)
	for _, option := range config.AppearanceOptions {
		promptSendMessageW.Call(state.appearance, settingsCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.appearance, settingsCBSet, uintptr(config.AppearanceIndex), 0)

	const leftX = 36
	const rightX = 390
	const fieldWidth = 334
	const editWidth = 160
	for index, field := range config.Numbers {
		row := index / 2
		column := index % 2
		xPos := leftX
		if column == 1 {
			xPos = rightX
		}
		yLabel := 192 + row*72
		makeControl("STATIC", field.Label, 0, xPos, yLabel, fieldWidth, 24, 0, captionFont)
		edit := makeControl(
			"EDIT",
			strconv.Itoa(field.Value),
			settingsWSBorder|settingsWSTabStop|settingsESNumber,
			xPos, yLabel+27, editWidth, 31,
			settingsIDNumberBase+index,
			font,
		)
		if edit != 0 {
			promptSendMessageW.Call(edit, promptEMSetLimitText, 5, 0)
		}
		state.numbers = append(state.numbers, edit)
	}

	separatorY := 192 + ((len(config.Numbers)+1)/2)*72 + 4
	makeControl("STATIC", "", settingsEtchedHorz, 36, separatorY, 688, 2, 0, font)
	conflictLabelY := separatorY + 18
	makeControl("STATIC", config.ConflictLabel, 0, 36, conflictLabelY, 688, 24, 0, captionFont)
	state.conflict = makeControl("COMBOBOX", "", settingsWSTabStop|settingsWSVScroll|settingsCBSDropList, 36, conflictLabelY+27, 688, 240, settingsIDConflict, font)
	for _, option := range config.ConflictOptions {
		promptSendMessageW.Call(state.conflict, settingsCBAdd, 0, uintptr(unsafe.Pointer(promptWstr(option))))
	}
	promptSendMessageW.Call(state.conflict, settingsCBSet, uintptr(config.ConflictIndex), 0)

	confirmY := conflictLabelY + 70
	state.confirm = makeControl("BUTTON", config.ConfirmDelete, settingsWSTabStop|settingsAutoCheck, 36, confirmY, 688, 28, settingsIDConfirm, font)
	if config.ConfirmDeleteOn {
		promptSendMessageW.Call(state.confirm, settingsBMSetCheck, settingsBSTChecked, 0)
	}

	makeControl("STATIC", "", settingsEtchedHorz, 36, 500, 688, 2, 0, font)
	makeControl("STATIC", config.Footer, 0, 36, 514, 470, 38, 0, captionFont)
	state.errorLabel = makeControl("STATIC", "", 0, 36, 552, 470, 24, settingsIDError, captionFont)
	applyButton := makeControl("BUTTON", config.ApplyLabel, settingsWSTabStop|settingsDefButton, 516, 530, 98, 38, settingsIDApply, font)
	makeControl("BUTTON", config.CancelLabel, settingsWSTabStop, 624, 530, 100, 38, settingsIDCancel, font)

	if state.appearance != 0 {
		promptSetFocus.Call(state.appearance)
	} else if applyButton != 0 {
		promptSetFocus.Call(applyButton)
	}
	promptShowWindow.Call(hwnd, 5)
	promptUpdateWindow.Call(hwnd)

	var msg promptMsg
	for !state.closed {
		r, _, _ := promptGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		promptTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		promptDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	return state.result, state.accepted
}
