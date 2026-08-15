//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

func buttonColors(v buttonVariant, pressed, disabled bool) (bg, border, fg uintptr) {
	if disabled {
		return rgb(25, 30, 38), rgb(43, 50, 62), rgb(103, 113, 126)
	}
	switch v {
	case buttonAccent:
		if pressed {
			return rgb(25, 132, 184), rgb(76, 205, 250), rgb(255, 255, 255)
		}
		return rgb(22, 116, 165), rgb(47, 181, 235), rgb(255, 255, 255)
	case buttonDanger:
		if pressed {
			return rgb(126, 47, 57), rgb(239, 103, 113), rgb(255, 244, 245)
		}
		return rgb(104, 40, 49), rgb(201, 78, 89), rgb(255, 235, 237)
	case buttonSubtle:
		if pressed {
			return rgb(34, 41, 51), rgb(68, 80, 95), textColor()
		}
		return rgb(23, 29, 37), rgb(45, 54, 67), textColor()
	default:
		if pressed {
			return rgb(43, 51, 63), rgb(78, 91, 108), textColor()
		}
		return rgb(31, 38, 48), rgb(58, 69, 84), textColor()
	}
}

func (a *app) drawButton(dis *drawItemStruct) bool {
	if dis == nil || dis.HwndItem == 0 {
		return false
	}
	visual, ok := a.buttons[dis.HwndItem]
	if !ok {
		return false
	}
	pressed := dis.ItemState&odsSelected != 0
	disabled := dis.ItemState&odsDisabled != 0
	bg, border, fg := buttonColors(visual.Variant, pressed, disabled)
	brush, _, _ := createSolidBrush.Call(bg)
	pen, _, _ := createPen.Call(psSolid, 1, border)
	oldBrush, _, _ := selectObject.Call(dis.HDC, brush)
	oldPen, _, _ := selectObject.Call(dis.HDC, pen)
	r := dis.RcItem
	roundRect.Call(dis.HDC, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), 8, 8)
	selectObject.Call(dis.HDC, oldBrush)
	selectObject.Call(dis.HDC, oldPen)
	if brush != 0 {
		deleteObject.Call(brush)
	}
	if pen != 0 {
		deleteObject.Call(pen)
	}

	setBkMode.Call(dis.HDC, transparentBkMode)
	setTextColor.Call(dis.HDC, fg)
	content := r
	content.Left += 10
	content.Right -= 10
	if pressed {
		content.Top++
		content.Bottom++
	}

	if visual.Icon != "" && a.iconFont != 0 {
		iconRect := content
		if visual.Label == "" {
			old, _, _ := selectObject.Call(dis.HDC, a.iconFont)
			drawText(dis.HDC, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
			selectObject.Call(dis.HDC, old)
		} else {
			iconRect.Right = iconRect.Left + 24
			old, _, _ := selectObject.Call(dis.HDC, a.iconFont)
			drawText(dis.HDC, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
			selectObject.Call(dis.HDC, old)
			content.Left += 30
		}
	}
	if visual.Label != "" {
		old, _, _ := selectObject.Call(dis.HDC, a.font)
		drawText(dis.HDC, visual.Label, &content, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(dis.HDC, old)
	}
	if dis.ItemState&odsFocus != 0 && !disabled {
		focus := r
		focus.Left += 3
		focus.Top += 3
		focus.Right -= 3
		focus.Bottom -= 3
		drawFocusRect.Call(dis.HDC, uintptr(unsafe.Pointer(&focus)))
	}
	return true
}

func drawText(hdc uintptr, text string, r *rect, flags uint32) {
	if text == "" || r == nil {
		return
	}
	buf := syscall.StringToUTF16(text)
	if len(buf) == 0 {
		return
	}
	drawTextW.Call(hdc, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)-1), uintptr(unsafe.Pointer(r)), uintptr(flags))
}
