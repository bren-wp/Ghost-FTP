//go:build windows

package desktop

var redrawWindowW = user32.NewProc("RedrawWindow")

const (
	rdwInvalidate  = 0x0001
	rdwAllChildren = 0x0080
	rdwUpdateNow   = 0x0100
)

// reflowWorkspace batches resize work into one non-erasing redraw. Moving many
// native child controls with immediate repaint used to produce visible flashes
// during startup/maximize/interactive resize on some Windows systems.
func (a *app) reflowWorkspace(width, height int) {
	if a == nil || a.hwnd == 0 || width <= 0 || height <= 0 {
		return
	}
	sendMessageW.Call(a.hwnd, wmSetRedraw, 0, 0)
	a.layout(width, height)
	a.refineWorkspaceLayout()
	sendMessageW.Call(a.hwnd, wmSetRedraw, 1, 0)
	redrawWindowW.Call(a.hwnd, 0, 0, rdwInvalidate|rdwAllChildren|rdwUpdateNow)
}
