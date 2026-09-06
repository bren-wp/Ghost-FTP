//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

func buttonColors(v buttonVariant, pressed, disabled bool) (bg, border, fg uintptr) {
	if disabled {
		return rgb(238, 241, 245), rgb(215, 220, 227), rgb(143, 151, 164)
	}
	switch v {
	case buttonAccent:
		if pressed {
			return accentStrongColor(), rgb(31, 75, 145), rgb(255, 255, 255)
		}
		return accentColor(), rgb(42, 91, 171), rgb(255, 255, 255)
	case buttonDanger:
		if pressed {
			return rgb(145, 28, 23), rgb(120, 20, 17), rgb(255, 255, 255)
		}
		return rgb(255, 244, 242), rgb(218, 90, 82), dangerColor()
	case buttonSubtle:
		if pressed {
			return rgb(229, 238, 250), rgb(158, 183, 218), textColor()
		}
		return panelColor(), borderColor(), rgb(61, 79, 102)
	default:
		if pressed {
			return rgb(231, 235, 240), rgb(173, 181, 191), textColor()
		}
		return rgb(250, 251, 252), borderColor(), textColor()
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
	content.Left += 10
	content.Right -= 10
	if pressed {
		content.Top++
		content.Bottom++
	}

	if visual.Vertical {
		a.drawVerticalToolbarContent(dis.HDC, content, visual)
	} else {
		a.drawHorizontalButtonContent(dis.HDC, content, visual)
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

func (a *app) drawHorizontalButtonContent(hdc uintptr, content rect, visual buttonVisual) {
	if visual.Icon != "" && a.iconFont != 0 {
		iconRect := content
		if visual.Label == "" {
			old, _, _ := selectObject.Call(hdc, a.iconFont)
			drawText(hdc, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
			selectObject.Call(hdc, old)
		} else {
			iconRect.Right = iconRect.Left + 24
			old, _, _ := selectObject.Call(hdc, a.iconFont)
			drawText(hdc, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
			selectObject.Call(hdc, old)
			content.Left += 32
		}
	}
	if visual.Label != "" {
		old, _, _ := selectObject.Call(hdc, a.font)
		drawText(hdc, visual.Label, &content, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(hdc, old)
	}
}

func (a *app) drawVerticalToolbarContent(hdc uintptr, content rect, visual buttonVisual) {
	height := content.Bottom - content.Top
	iconRect := content
	iconRect.Bottom = content.Top + height*58/100
	labelRect := content
	labelRect.Top = iconRect.Bottom - 1

	if visual.Icon != "" && a.iconFont != 0 {
		old, _, _ := selectObject.Call(hdc, a.iconFont)
		drawText(hdc, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
		selectObject.Call(hdc, old)
	}
	if visual.Label != "" {
		old, _, _ := selectObject.Call(hdc, a.smallFont)
		drawText(hdc, visual.Label, &labelRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(hdc, old)
	}
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
