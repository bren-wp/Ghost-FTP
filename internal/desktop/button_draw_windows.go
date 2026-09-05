//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

func buttonColors(v buttonVariant, pressed, disabled bool) (bg, border, fg uintptr) {
	if disabled {
		return rgb(20, 27, 36), rgb(38, 49, 63), rgb(100, 116, 139)
	}
	switch v {
	case buttonAccent:
		if pressed {
			return rgb(2, 132, 199), rgb(125, 211, 252), rgb(248, 250, 252)
		}
		return rgb(3, 105, 161), rgb(56, 189, 248), rgb(248, 250, 252)
	case buttonDanger:
		if pressed {
			return rgb(127, 29, 29), rgb(248, 113, 113), rgb(255, 247, 247)
		}
		return rgb(101, 28, 38), rgb(239, 68, 68), rgb(255, 241, 242)
	case buttonSubtle:
		if pressed {
			return rgb(30, 41, 54), rgb(71, 85, 105), textColor()
		}
		return rgb(15, 23, 32), rgb(42, 55, 72), textColor()
	default:
		if pressed {
			return rgb(36, 49, 64), rgb(100, 116, 139), textColor()
		}
		return rgb(24, 33, 45), rgb(55, 70, 89), textColor()
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
	roundRect.Call(dis.HDC, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), 10, 10)
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
	content.Left += 12
	content.Right -= 12
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
			content.Left += 32
		}
	}
	if visual.Label != "" {
		old, _, _ := selectObject.Call(dis.HDC, a.font)
		drawText(dis.HDC, visual.Label, &content, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(dis.HDC, old)
	}
	if dis.ItemState&odsFocus != 0 && !disabled {
		focus := r
		focus.Left += 4
		focus.Top += 4
		focus.Right -= 4
		focus.Bottom -= 4
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
