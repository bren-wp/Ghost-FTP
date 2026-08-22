//go:build windows

package desktop

func buttonFlowRows(widths []int, gap, available int) int {
	if len(widths) == 0 || available <= 0 {
		return 0
	}
	rows := 1
	used := 0
	for _, width := range widths {
		if width > available {
			width = available
		}
		need := width
		if used > 0 {
			need += gap
		}
		if used > 0 && used+need > available {
			rows++
			used = width
			continue
		}
		used += need
	}
	return rows
}

// placeButtonFlow places complete-label buttons from left to right and wraps
// only between buttons. It returns the number of rows used.
func (a *app) placeButtonFlow(handles []uintptr, widths []int, x, y, available, gap, rowH int) int {
	if len(handles) == 0 || len(handles) != len(widths) || available <= 0 {
		return 0
	}
	row := 0
	used := 0
	for index, hwnd := range handles {
		width := widths[index]
		if width > available {
			width = available
		}
		need := width
		if used > 0 {
			need += gap
		}
		if used > 0 && used+need > available {
			row++
			used = 0
			need = width
		}
		buttonX := x + used
		if used > 0 {
			buttonX += gap
			used += gap
		}
		a.move(hwnd, buttonX, y+row*(rowH+gap), width, rowH)
		used += width
	}
	return row + 1
}
