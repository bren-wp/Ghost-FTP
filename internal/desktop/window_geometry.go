package desktop

const preferredWindowHorizontalMargin = 48
const preferredWindowVerticalMargin = 72

type logicalWindowBounds struct {
	X      int
	Y      int
	Width  int
	Height int
}

func responsiveAxisSize(workSize, preferredSize, minimumSize, margin int) int {
	if workSize <= 0 {
		return preferredSize
	}
	// On constrained displays, accessibility wins over the normal breathing
	// room: use the whole work area rather than forcing the canonical minimum
	// beyond a physical screen edge.
	if workSize < minimumSize {
		return workSize
	}
	available := workSize - margin
	if available < minimumSize {
		available = minimumSize
	}
	if preferredSize > available {
		return available
	}
	return preferredSize
}

func responsiveWindowBoundsForWorkArea(workX, workY, workWidth, workHeight int) logicalWindowBounds {
	width := responsiveAxisSize(workWidth, premiumStartWidth, premiumMinWidth, preferredWindowHorizontalMargin)
	height := responsiveAxisSize(workHeight, premiumStartHeight, premiumMinHeight, preferredWindowVerticalMargin)
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return logicalWindowBounds{
		X:      workX + (workWidth-width)/2,
		Y:      workY + (workHeight-height)/2,
		Width:  width,
		Height: height,
	}
}

func responsiveMinimumTrackSize(workWidth, workHeight int) (width, height int) {
	width = premiumMinWidth
	height = premiumMinHeight
	if workWidth > 0 && width > workWidth {
		width = workWidth
	}
	if workHeight > 0 && height > workHeight {
		height = workHeight
	}
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return
}
