//go:build windows

package desktop

import (
	"sync"
	"syscall"
	"unsafe"
)

const (
	nmCustomDraw       = 0xFFFFFFF4
	cddsPrepaint       = 0x00000001
	cddsItemPrepaint   = 0x00010001
	cdrfNewFont        = 0x00000002
	cdrfSkipDefault    = 0x00000004
	cdrfNotifyItemDraw = 0x00000020

	hdmFirst    = 0x1200
	hdmGetItemW = hdmFirst + 11
	hdiText     = 0x0002
	wmNCDestroy = 0x0082

	workspaceHeaderSubclassID = 0x47465450
)

var (
	setWindowSubclassHeader   = comctl32.NewProc("SetWindowSubclass")
	defSubclassProcHeader     = comctl32.NewProc("DefSubclassProc")
	fillRectHeader            = user32.NewProc("FillRect")
	workspaceListOwners       sync.Map
	workspaceListSubclassProc = syscall.NewCallback(workspaceListSubclass)
)

type nmCustomDrawStruct struct {
	Hdr        nmhdr
	DrawStage  uint32
	HDC        uintptr
	Rc         rect
	ItemSpec   uintptr
	ItemState  uint32
	ItemLParam uintptr
}

type hdItemW struct {
	Mask       uint32
	Cxy        int32
	Text       *uint16
	Bitmap     uintptr
	TextMax    int32
	Fmt        int32
	Param      uintptr
	Image      int32
	Order      int32
	FilterType uint32
	Filter     uintptr
	State      uint32
}

func customDrawFromLParam(p uintptr) nmCustomDrawStruct {
	var d nmCustomDrawStruct
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&d)), p, unsafe.Sizeof(d))
	}
	return d
}

func headerForList(list uintptr) uintptr {
	if list == 0 {
		return 0
	}
	h, _, _ := sendMessageW.Call(list, lvmGetHeader, 0, 0)
	return h
}

// installWorkspaceHeaderDraw subclasses the ListView because a native Header
// control sends NM_CUSTOMDRAW to its immediate ListView parent, not to Ghost
// FTP's top-level window.
func installWorkspaceHeaderDraw(a *app, list uintptr) {
	if a == nil || list == 0 {
		return
	}
	if err := setWindowSubclassHeader.Find(); err != nil {
		return
	}
	workspaceListOwners.Store(list, a)
	if ok, _, _ := setWindowSubclassHeader.Call(list, workspaceListSubclassProc, workspaceHeaderSubclassID, 0); ok == 0 {
		workspaceListOwners.Delete(list)
	}
}

func workspaceListSubclass(hwnd, message, wParam, lParam, _ uintptr, _ uintptr) uintptr {
	if uint32(message) == wmNotify && lParam != 0 {
		h := nmhdrFromLParam(lParam)
		if h.Code == nmCustomDraw && h.HwndFrom == headerForList(hwnd) {
			if owner, ok := workspaceListOwners.Load(hwnd); ok {
				return owner.(*app).drawWorkspaceHeader(lParam)
			}
		}
	}
	if uint32(message) == wmNCDestroy {
		// Common Controls removes installed subclasses as part of window
		// destruction. Only our owner lookup needs explicit cleanup here; avoiding
		// a callback self-reference also keeps package initialization acyclic.
		workspaceListOwners.Delete(hwnd)
	}
	result, _, _ := defSubclassProcHeader.Call(hwnd, message, wParam, lParam)
	return result
}

func (a *app) isWorkspaceHeader(hwnd uintptr) bool {
	if a == nil || hwnd == 0 {
		return false
	}
	return hwnd == headerForList(a.localList) || hwnd == headerForList(a.remoteList) || hwnd == headerForList(a.transferList)
}

func headerItemText(header, itemIndex uintptr, buffer *[256]uint16) (int, bool) {
	if header == 0 || buffer == nil {
		return 0, false
	}
	item := hdItemW{
		Mask:    hdiText,
		Text:    &buffer[0],
		TextMax: int32(len(buffer)),
	}
	if ok, _, _ := sendMessageW.Call(header, hdmGetItemW, itemIndex, uintptr(unsafe.Pointer(&item))); ok == 0 {
		return 0, false
	}
	for i, ch := range buffer {
		if ch == 0 {
			return i, true
		}
	}
	return len(buffer), true
}

// drawWorkspaceHeader owns both the background and text pixels so header
// contrast remains deterministic even when the host Windows account uses a
// light system theme while Ghost FTP is dark.
func (a *app) drawWorkspaceHeader(lParam uintptr) uintptr {
	d := customDrawFromLParam(lParam)
	switch d.DrawStage {
	case cddsPrepaint:
		return cdrfNotifyItemDraw
	case cddsItemPrepaint:
		if a.panelBrush != 0 {
			fillRectHeader.Call(d.HDC, uintptr(unsafe.Pointer(&d.Rc)), a.panelBrush)
		}

		var text [256]uint16
		length, ok := headerItemText(d.Hdr.HwndFrom, d.ItemSpec, &text)
		if ok && length > 0 {
			textRect := d.Rc
			textRect.Left += int32(a.scale(8))
			textRect.Right -= int32(a.scale(6))
			setTextColor.Call(d.HDC, textColor())
			setBkMode.Call(d.HDC, transparentBkMode)
			oldFont := uintptr(0)
			if a.smallFont != 0 {
				oldFont, _, _ = selectObject.Call(d.HDC, a.smallFont)
			}
			drawTextW.Call(
				d.HDC,
				uintptr(unsafe.Pointer(&text[0])),
				uintptr(length),
				uintptr(unsafe.Pointer(&textRect)),
				dtLeft|dtVCenter|dtSingleLine|dtEndEllipsis|dtNoPrefix,
			)
			if oldFont != 0 {
				selectObject.Call(d.HDC, oldFont)
			}
		}
		return cdrfSkipDefault
	default:
		return 0
	}
}
