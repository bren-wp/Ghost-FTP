//go:build windows

package desktop

import (
	"sync"
	"unicode/utf8"
	"unsafe"
)

const (
	odtMenu            = 1
	odsGrayed          = 0x0002
	wmMeasureItem      = 0x002C
	mimBackground      = 0x00000002
	mimApplyToSubmenus = 0x80000000
)

type menuVisual struct {
	label string
	root  bool
}

var menuVisualState = struct {
	sync.RWMutex
	items map[uintptr]menuVisual
}{items: make(map[uintptr]menuVisual)}

var (
	fillRectW    = user32.NewProc("FillRect")
	setMenuInfoW = user32.NewProc("SetMenuInfo")
)

type measureItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemWidth  uint32
	ItemHeight uint32
	ItemData   uintptr
}

type menuInfo struct {
	CbSize          uint32
	FMask           uint32
	DwStyle         uint32
	CyMax           uint32
	HbrBack         uintptr
	DwContextHelpID uint32
	DwMenuData      uintptr
}

func resetMenuVisuals() {
	menuVisualState.Lock()
	menuVisualState.items = make(map[uintptr]menuVisual)
	menuVisualState.Unlock()
}

func registerMenuVisual(key uintptr, label string, root bool) {
	menuVisualState.Lock()
	menuVisualState.items[key] = menuVisual{label: label, root: root}
	menuVisualState.Unlock()
}

func menuVisualFor(key uintptr) (menuVisual, bool) {
	menuVisualState.RLock()
	v, ok := menuVisualState.items[key]
	menuVisualState.RUnlock()
	return v, ok
}

func measureItemFromLParam(p uintptr) measureItemStruct {
	var item measureItemStruct
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&item)), p, unsafe.Sizeof(item))
	}
	return item
}

func measureItemToLParam(p uintptr, item measureItemStruct) {
	if p != 0 {
		rtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&item)), unsafe.Sizeof(item))
	}
}

func (a *app) measureMenuItem(lParam uintptr) bool {
	if lParam == 0 {
		return false
	}
	item := measureItemFromLParam(lParam)
	if item.CtlType != odtMenu {
		return false
	}
	visual, ok := menuVisualFor(item.ItemData)
	if !ok {
		return false
	}
	glyphs := utf8.RuneCountInString(visual.label)
	width := 30 + glyphs*8
	height := 28
	if visual.root {
		width = 20 + glyphs*8
		height = 24
	}
	if width < 48 {
		width = 48
	}
	item.ItemWidth = uint32(width)
	item.ItemHeight = uint32(height)
	measureItemToLParam(lParam, item)
	return true
}

func (a *app) drawMenuItem(d *drawItemStruct) bool {
	if a == nil || d == nil || d.CtlType != odtMenu {
		return false
	}
	visual, ok := menuVisualFor(d.ItemData)
	if !ok {
		return false
	}
	brush := a.panelBrush
	temporary := uintptr(0)
	if d.ItemState&odsSelected != 0 {
		temporary, _, _ = createSolidBrush.Call(selectionColor())
		if temporary != 0 {
			brush = temporary
		}
	}
	if brush != 0 {
		fillRectW.Call(d.HDC, uintptr(unsafe.Pointer(&d.RcItem)), brush)
	}
	if temporary != 0 {
		deleteObject.Call(temporary)
	}
	color := textColor()
	if d.ItemState&(odsDisabled|odsGrayed) != 0 {
		color = mutedColor()
	}
	setTextColor.Call(d.HDC, color)
	setBkMode.Call(d.HDC, transparentBkMode)
	rc := d.RcItem
	if visual.root {
		rc.Left += 8
		rc.Right -= 8
	} else {
		rc.Left += 12
		rc.Right -= 12
	}
	label := wstr(visual.label)
	drawTextW.Call(d.HDC, uintptr(unsafe.Pointer(label)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
	return true
}

func applyDarkMenuBackground(menu, brush uintptr) {
	if menu == 0 || brush == 0 {
		return
	}
	info := menuInfo{
		CbSize:  uint32(unsafe.Sizeof(menuInfo{})),
		FMask:   mimBackground | mimApplyToSubmenus,
		HbrBack: brush,
	}
	setMenuInfoW.Call(menu, uintptr(unsafe.Pointer(&info)))
}
