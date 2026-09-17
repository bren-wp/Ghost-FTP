//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

func buttonColors(v buttonVariant, pressed, disabled bool) (bg, border, fg uintptr) {
	if disabled {
		return panelColor(), borderColor(), mutedColor()
	}

	switch v {
	case buttonAccent:
		if pressed {
			return accentStrongColor(), accentStrongColor(), onAccentColor()
		}
		return accentColor(), accentStrongColor(), onAccentColor()
	case buttonDanger:
		if activeThemeIsDark() {
			if pressed {
				return rgb(112, 31, 49), dangerColor(), rgb(255, 246, 248)
			}
			return rgb(73, 28, 43), dangerColor(), rgb(255, 236, 241)
		}
		if pressed {
			return rgb(245, 194, 199), dangerColor(), rgb(127, 29, 29)
		}
		return rgb(253, 236, 238), dangerColor(), rgb(153, 27, 27)
	case buttonSubtle:
		if pressed {
			return selectionColor(), accentColor(), textColor()
		}
		return panelColor(), borderColor(), mutedColor()
	default:
		if pressed {
			return selectionColor(), accentColor(), textColor()
		}
		return listColor(), borderColor(), textColor()
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
	contentWidth := int(content.Right - content.Left)

	// A 1024-class desktop leaves less horizontal room after the application
	// sidebar is reserved. Prefer readable action text over decorative icons:
	// wide buttons show icon + label, medium buttons show a centered label, and
	// only genuinely narrow secondary actions become intentional icon buttons.
	// This avoids accidental "Con…", "Ren…" and similar clipped controls.
	if visual.Icon != "" && visual.Label != "" {
		switch {
		case contentWidth >= 132 && a.iconFont != 0:
			iconRect := content
			iconRect.Right = iconRect.Left + 24
			old, _, _ := selectObject.Call(hdc, a.iconFont)
			drawText(hdc, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
			selectObject.Call(hdc, old)
			content.Left += 32
		case contentWidth >= 68:
			font := a.font
			if contentWidth < 104 && a.smallFont != 0 {
				font = a.smallFont
			}
			old, _, _ := selectObject.Call(hdc, font)
			drawText(hdc, visual.Label, &content, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
			selectObject.Call(hdc, old)
			return
		default:
			if a.iconFont != 0 {
				old, _, _ := selectObject.Call(hdc, a.iconFont)
				drawText(hdc, visual.Icon, &content, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
				selectObject.Call(hdc, old)
			}
			return
		}
	} else if visual.Icon != "" && a.iconFont != 0 {
		old, _, _ := selectObject.Call(hdc, a.iconFont)
		drawText(hdc, visual.Icon, &content, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
		selectObject.Call(hdc, old)
		if visual.Label == "" {
			return
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
