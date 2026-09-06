//go:build windows

package desktop

import "unsafe"

const (
	nmCustomDraw       = 0xFFFFFFF4
	cddsPrepaint       = 0x00000001
	cddsItemPrepaint   = 0x00010001
	cdrfNewFont        = 0x00000002
	cdrfNotifyItemDraw = 0x00000020
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

func (a *app) isWorkspaceHeader(hwnd uintptr) bool {
	if a == nil || hwnd == 0 {
		return false
	}
	return hwnd == headerForList(a.localList) || hwnd == headerForList(a.remoteList) || hwnd == headerForList(a.transferList)
}

func (a *app) drawWorkspaceHeader(lParam uintptr) uintptr {
	d := customDrawFromLParam(lParam)
	switch d.DrawStage {
	case cddsPrepaint:
		return cdrfNotifyItemDraw
	case cddsItemPrepaint:
		setTextColor.Call(d.HDC, textColor())
		setBkColor.Call(d.HDC, panelColor())
		setBkMode.Call(d.HDC, transparentBkMode)
		return cdrfNewFont
	default:
		return 0
	}
}
