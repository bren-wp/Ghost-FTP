//go:build windows

package desktop

import "unsafe"

const (
	siteLBSOwnerDrawFixed = 0x0010
	siteLBSHasStrings     = 0x0040
	siteODTListBox        = 2
)

func applySiteManagerNavigationTheme(list uintptr) {
	if list == 0 {
		return
	}
	theme := siteManagerNavigationThemeClass(activeThemeIsDark())
	if theme == "" {
		// Classic Light must use the normal native Explorer/list treatment. The
		// previous unconditional DarkMode_Explorer assignment left the primary
		// light appearance with mixed native rendering.
		setWindowTheme.Call(list, 0, 0)
		return
	}
	setWindowTheme.Call(list, uintptr(unsafe.Pointer(wstr(theme))), 0)
}

func (state *siteManagerState) measureNavigationItem(lParam uintptr) bool {
	if state == nil || state.parent == nil || lParam == 0 {
		return false
	}
	item := measureItemFromLParam(lParam)
	if item.CtlType != siteODTListBox || item.CtlID != siteIDList {
		return false
	}
	height := state.parent.scale(48)
	if height < 40 {
		height = 40
	}
	item.ItemHeight = uint32(height)
	measureItemToLParam(lParam, item)
	return true
}

func fillSiteManagerRect(hdc uintptr, r *rect, color uintptr) {
	if hdc == 0 || r == nil {
		return
	}
	brush, _, _ := createSolidBrush.Call(color)
	if brush == 0 {
		return
	}
	fillRectW.Call(hdc, uintptr(unsafe.Pointer(r)), brush)
	deleteObject.Call(brush)
}

func (state *siteManagerState) drawNavigationItem(d *drawItemStruct) bool {
	if state == nil || state.parent == nil || d == nil || d.CtlType != siteODTListBox || d.CtlID != siteIDList {
		return false
	}

	selected := d.ItemState&odsSelected != 0
	background := listColor()
	if selected {
		background = selectionColor()
	}
	fillSiteManagerRect(d.HDC, &d.RcItem, background)

	// A narrow accent rail makes the active server obvious without introducing
	// another button, checkbox or duplicated navigation state.
	if selected {
		rail := d.RcItem
		rail.Right = rail.Left + int32(state.parent.scale(4))
		fillSiteManagerRect(d.HDC, &rail, accentColor())
	}

	if d.ItemID == ^uint32(0) {
		return true
	}
	rows := siteManagerNavigationRows(state.parent.tr("profile.quick"), state.profiles)
	index := int(d.ItemID)
	if index < 0 || index >= len(rows) {
		return true
	}
	row := rows[index]

	setBkMode.Call(d.HDC, transparentBkMode)
	primary := d.RcItem
	primary.Left += int32(state.parent.scale(14))
	primary.Right -= int32(state.parent.scale(8))

	if row.Secondary == "" {
		setTextColor.Call(d.HDC, textColor())
		old, _, _ := selectObject.Call(d.HDC, state.parent.font)
		drawText(d.HDC, row.Primary, &primary, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(d.HDC, old)
	} else {
		primary.Top += int32(state.parent.scale(5))
		primary.Bottom = primary.Top + int32(state.parent.scale(20))
		setTextColor.Call(d.HDC, textColor())
		old, _, _ := selectObject.Call(d.HDC, state.parent.font)
		drawText(d.HDC, row.Primary, &primary, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(d.HDC, old)

		secondary := d.RcItem
		secondary.Left = primary.Left
		secondary.Right = primary.Right
		secondary.Top = primary.Bottom
		secondary.Bottom -= int32(state.parent.scale(4))
		setTextColor.Call(d.HDC, mutedColor())
		old, _, _ = selectObject.Call(d.HDC, state.parent.smallFont)
		drawText(d.HDC, row.Secondary, &secondary, dtLeft|dtVCenter|dtSingleLine|dtNoPrefix|dtEndEllipsis)
		selectObject.Call(d.HDC, old)
	}

	if d.ItemState&odsFocus != 0 {
		focus := d.RcItem
		inset := int32(state.parent.scale(3))
		focus.Left += inset
		focus.Top += inset
		focus.Right -= inset
		focus.Bottom -= inset
		drawFocusRect.Call(d.HDC, uintptr(unsafe.Pointer(&focus)))
	}
	return true
}
