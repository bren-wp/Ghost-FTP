package desktop

// fileColumnWidths returns logical-pixel widths that fit inside the actual
// ListView client width. The previous Windows layout estimated pane width from
// the top-level window, which ignored the application sidebar and could push the
// final Permissions column outside the visible remote pane.
func fileColumnWidths(listWidth int, remote bool) []int {
	if listWidth < 1 {
		listWidth = 1
	}

	// Leave room for ListView borders and the vertical scrollbar so the width sum
	// never creates a horizontal scrollbar at the standard compact workspace.
	available := listWidth - 18
	if available < 1 {
		available = 1
	}

	if remote {
		typeW, sizeW, modifiedW, permissionsW := 82, 92, 132, 112
		if available < 520 {
			// Compact widths are deliberately based on header readability rather than
			// the old wide-pane defaults. This keeps Permissions fully visible while
			// preserving as much filename space as the real pane permits.
			typeW, sizeW, modifiedW, permissionsW = 48, 48, 74, 84
		}
		fixed := typeW + sizeW + modifiedW + permissionsW
		nameW := available - fixed
		if nameW < 1 {
			nameW = 1
		}
		return []int{nameW, typeW, sizeW, modifiedW, permissionsW}
	}

	typeW, sizeW, modifiedW := 82, 92, 132
	if available < 520 {
		typeW, sizeW, modifiedW = 60, 60, 86
	}
	fixed := typeW + sizeW + modifiedW
	nameW := available - fixed
	if nameW < 1 {
		nameW = 1
	}
	return []int{nameW, typeW, sizeW, modifiedW}
}
