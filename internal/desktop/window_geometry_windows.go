//go:build windows

package desktop

import "unsafe"

const monitorDefaultToNearest = 2

type responsiveMonitorInfo struct {
	Size    uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

var monitorFromWindowResponsive = user32.NewProc("MonitorFromWindow")
var monitorFromRectResponsive = user32.NewProc("MonitorFromRect")
var getMonitorInfoWResponsive = user32.NewProc("GetMonitorInfoW")

func (a *app) logicalFromPhysical(value int32) int {
	dpi := a.dpi
	if dpi == 0 {
		dpi = 96
	}
	// Deliberately use truncation rather than a positive-only rounding offset:
	// monitor origins may be negative in multi-monitor layouts.
	return int(value) * 96 / int(dpi)
}

func (a *app) monitorWorkAreaLogical() (x, y, width, height int, ok bool) {
	if a == nil || a.hwnd == 0 {
		return 0, 0, 0, 0, false
	}
	monitor, _, _ := monitorFromWindowResponsive.Call(a.hwnd, monitorDefaultToNearest)
	if monitor == 0 {
		return 0, 0, 0, 0, false
	}
	info := responsiveMonitorInfo{Size: uint32(unsafe.Sizeof(responsiveMonitorInfo{}))}
	result, _, _ := getMonitorInfoWResponsive.Call(monitor, uintptr(unsafe.Pointer(&info)))
	if result == 0 || info.Work.Right <= info.Work.Left || info.Work.Bottom <= info.Work.Top {
		return 0, 0, 0, 0, false
	}
	x = a.logicalFromPhysical(info.Work.Left)
	y = a.logicalFromPhysical(info.Work.Top)
	right := a.logicalFromPhysical(info.Work.Right)
	bottom := a.logicalFromPhysical(info.Work.Bottom)
	return x, y, right - x, bottom - y, true
}

func (a *app) responsiveWindowBounds() (x, y, width, height int) {
	workX, workY, workWidth, workHeight, ok := a.monitorWorkAreaLogical()
	if !ok {
		screenWRaw, _, _ := getSystemMetrics.Call(smCxScreen)
		screenHRaw, _, _ := getSystemMetrics.Call(smCyScreen)
		if screenWRaw == 0 || screenHRaw == 0 {
			return 40, 30, premiumStartWidth, premiumStartHeight
		}
		workWidth = a.unscale(int(screenWRaw))
		workHeight = a.unscale(int(screenHRaw))
	}
	bounds := responsiveWindowBoundsForWorkArea(workX, workY, workWidth, workHeight)
	return bounds.X, bounds.Y, bounds.Width, bounds.Height
}

func (a *app) responsiveMinTrackSize() (width, height int) {
	_, _, workWidth, workHeight, ok := a.monitorWorkAreaLogical()
	if !ok {
		screenWRaw, _, _ := getSystemMetrics.Call(smCxScreen)
		screenHRaw, _, _ := getSystemMetrics.Call(smCyScreen)
		if screenWRaw != 0 {
			workWidth = a.unscale(int(screenWRaw))
		}
		if screenHRaw != 0 {
			workHeight = a.unscale(int(screenHRaw))
		}
	}
	return responsiveMinimumTrackSize(workWidth, workHeight)
}

func (a *app) clampSuggestedWindowRectToWorkArea(value rect) rect {
	if a == nil {
		return value
	}
	// WM_DPICHANGED supplies a rectangle for the destination monitor. Resolve
	// that rectangle directly rather than the window's pre-move HWND position,
	// otherwise a cross-monitor DPI transition can be clamped back to the old
	// monitor before MoveWindow applies the suggested bounds.
	monitor, _, _ := monitorFromRectResponsive.Call(uintptr(unsafe.Pointer(&value)), monitorDefaultToNearest)
	if monitor == 0 {
		return value
	}
	info := responsiveMonitorInfo{Size: uint32(unsafe.Sizeof(responsiveMonitorInfo{}))}
	result, _, _ := getMonitorInfoWResponsive.Call(monitor, uintptr(unsafe.Pointer(&info)))
	if result == 0 || info.Work.Right <= info.Work.Left || info.Work.Bottom <= info.Work.Top {
		return value
	}

	width := value.Right - value.Left
	height := value.Bottom - value.Top
	workWidth := info.Work.Right - info.Work.Left
	workHeight := info.Work.Bottom - info.Work.Top
	if width > workWidth {
		width = workWidth
	}
	if height > workHeight {
		height = workHeight
	}
	if value.Left < info.Work.Left {
		value.Left = info.Work.Left
	}
	if value.Top < info.Work.Top {
		value.Top = info.Work.Top
	}
	if value.Left+width > info.Work.Right {
		value.Left = info.Work.Right - width
	}
	if value.Top+height > info.Work.Bottom {
		value.Top = info.Work.Bottom - height
	}
	value.Right = value.Left + width
	value.Bottom = value.Top + height
	return value
}
