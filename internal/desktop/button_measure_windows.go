//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

var (
	getDCButtonMeasure              = user32.NewProc("GetDC")
	releaseDCButtonMeasure          = user32.NewProc("ReleaseDC")
	getTextExtentPoint32WButtonMeasure = gdi32.NewProc("GetTextExtentPoint32W")
)

type textExtent struct {
	CX int32
	CY int32
}

// preferredButtonWidth returns a logical-pixel width that fits the full
// localized owner-drawn label, its optional icon and comfortable padding.
// The caller supplies a semantic floor so short labels still have a stable
// visual rhythm. If text measurement is unavailable, the floor is used.
func (a *app) preferredButtonWidth(hwnd uintptr, floor int) int {
	if floor < 64 {
		floor = 64
	}
	visual, ok := a.buttons[hwnd]
	if !ok || visual.Label == "" || a.hwnd == 0 {
		return floor
	}
	hdc, _, _ := getDCButtonMeasure.Call(a.hwnd)
	if hdc == 0 {
		return floor
	}
	defer releaseDCButtonMeasure.Call(a.hwnd, hdc)
	oldFont := uintptr(0)
	if a.font != 0 {
		oldFont, _, _ = selectObject.Call(hdc, a.font)
		if oldFont != 0 {
			defer selectObject.Call(hdc, oldFont)
		}
	}
	buf := syscall.StringToUTF16(visual.Label)
	if len(buf) <= 1 {
		return floor
	}
	var extent textExtent
	okCall, _, _ := getTextExtentPoint32WButtonMeasure.Call(
		hdc,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(len(buf)-1),
		uintptr(unsafe.Pointer(&extent)),
	)
	if okCall == 0 || extent.CX <= 0 {
		return floor
	}
	width := a.unscale(int(extent.CX)) + 24
	if visual.Icon != "" {
		width += 30
	}
	if width < floor {
		return floor
	}
	return width
}

func totalWidths(widths []int, gap int) int {
	if len(widths) == 0 {
		return 0
	}
	total := gap * (len(widths) - 1)
	for _, width := range widths {
		total += width
	}
	return total
}
