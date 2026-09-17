//go:build windows

package desktop

import (
	"strconv"
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
	case buttonNavActive:
		if pressed {
			return accentColor(), accentStrongColor(), onAccentColor()
		}
		return selectionColor(), accentColor(), accentStrongColor()
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
	roundRect.Call(dis.HDC, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(a.scale(10)), uintptr(a.scale(10)))
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
	content.Left += int32(a.scale(10))
	content.Right -= int32(a.scale(10))
	if pressed {
		content.Top++
		content.Bottom++
	}

	textContent := content
	if visual.Badge > 0 && !visual.Vertical {
		textContent.Right -= int32(a.scale(34))
	}
	if visual.Vertical {
		a.drawVerticalToolbarContent(dis.HDC, textContent, visual)
	} else {
		a.drawHorizontalButtonContent(dis.HDC, textContent, visual)
	}
	if visual.Badge > 0 {
		a.drawButtonBadge(dis.HDC, content, visual.Badge, disabled)
	}
	if dis.ItemState&odsFocus != 0 && !disabled {
		focus := r
		focus.Left += int32(a.scale(4))
		focus.Top += int32(a.scale(4))
		focus.Right -= int32(a.scale(4))
		focus.Bottom -= int32(a.scale(4))
		drawFocusRect.Call(dis.HDC, uintptr(unsafe.Pointer(&focus)))
	}
	return true
}

func (a *app) drawButtonBadge(hdc uintptr, content rect, count int, disabled bool) {
	if hdc == 0 || count <= 0 {
		return
	}
	label := strconv.Itoa(count)
	if count > 99 {
		label = "99+"
	}
	width := 22
	if len(label) >= 2 {
		width = 28
	}
	if len(label) >= 3 {
		width = 34
	}
	height := 20
	badge := content
	badge.Right -= int32(a.scale(1))
	badge.Left = badge.Right - int32(a.scale(width))
	centerY := int(badge.Top+badge.Bottom) / 2
	badge.Top = int32(centerY - a.scale(height)/2)
	badge.Bottom = badge.Top + int32(a.scale(height))

	bg := accentColor()
	fg := onAccentColor()
	if disabled {
		bg = borderColor()
		fg = mutedColor()
	}
	brush, _, _ := createSolidBrush.Call(bg)
	pen, _, _ := createPen.Call(psSolid, 1, bg)
	oldBrush, _, _ := selectObject.Call(hdc, brush)
	oldPen, _, _ := selectObject.Call(hdc, pen)
	roundRect.Call(hdc, uintptr(badge.Left), uintptr(badge.Top), uintptr(badge.Right), uintptr(badge.Bottom), uintptr(a.scale(10)), uintptr(a.scale(10)))
	selectObject.Call(hdc, oldBrush)
	selectObject.Call(hdc, oldPen)
	if brush != 0 {
		deleteObject.Call(brush)
	}
	if pen != 0 {
		deleteObject.Call(pen)
	}

	setBkMode.Call(hdc, transparentBkMode)
	setTextColor.Call(hdc, fg)
	oldFont := uintptr(0)
	if a.smallFont != 0 {
		oldFont, _, _ = selectObject.Call(hdc, a.smallFont)
	}
	drawText(hdc, label, &badge, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
	if oldFont != 0 {
		selectObject.Call(hdc, oldFont)
	}
}

func (a *app) drawHorizontalButtonContent(hdc uintptr, content rect, visual buttonVisual) {
	contentWidth := int(content.Right - content.Left)

	// Prefer icon + readable text for navigation-sized controls. Compact file
	// actions still collapse to label-only or icon-only rather than clipping.
	if visual.Icon != "" && visual.Label != "" {
		switch {
		case contentWidth >= a.scale(108) && a.iconFont != 0:
			iconRect := content
			iconRect.Right = iconRect.Left + int32(a.scale(22))
			old, _, _ := selectObject.Call(hdc, a.iconFont)
			drawText(hdc, visual.Icon, &iconRect, dtCenter|dtVCenter|dtSingleLine|dtNoPrefix)
			selectObject.Call(hdc, old)
			content.Left += int32(a.scale(30))
		case contentWidth >= a.scale(68):
			font := a.font
			if contentWidth < a.scale(104) && a.smallFont != 0 {
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
		font := a.font
		if contentWidth < a.scale(136) && a.smallFont != 0 {
			font = a.smallFont
		}
		old, _, _ := selectObject.Call(hdc, font)
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
