//go:build windows

package desktop

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"syscall"
	"unsafe"

	"github.com/bren-wp/Ghost-FTP/internal/api"
	"github.com/bren-wp/Ghost-FTP/internal/brand"
	"github.com/bren-wp/Ghost-FTP/internal/config"
	"github.com/bren-wp/Ghost-FTP/internal/model"
	"github.com/bren-wp/Ghost-FTP/internal/platform"
)

type app struct {
	hwnd                                 uintptr
	engine                               *api.Engine
	version                              string
	font, titleFont, smallFont, iconFont uintptr
	dpi                                  uint32
	brush, panelBrush                    uintptr

	brandTitle, brandSubtitle, connectionBadge, sectionLocal, sectionRemote, sectionTransfers uintptr
	profilesCombo, languageCombo, saveProfile, removeProfile, settingsBtn, aboutBtn           uintptr
	protocol, host, port, user, pass                                                          uintptr
	keyPath, chooseKey, passphrase                                                            uintptr
	connect, disconnect                                                                       uintptr
	localPath, localUp, localRefresh, localChoose, localList                                  uintptr
	localMkdir, localRename, localDelete                                                      uintptr
	remotePath, remoteUp, remoteRefresh, remoteList                                           uintptr
	remoteMkdir, remoteRename, remoteDelete, remoteChmod                                      uintptr
	upload, download                                                                          uintptr
	transferList, pauseQueue, resumeQueue, cancelJob, retryJob, clearQueue                    uintptr
	status, statusVersion, transferSummary                                                    uintptr
	buttons                                                                                   map[uintptr]buttonVisual

	siteManagerBtn uintptr

	mu                   sync.Mutex
	dispatchQ            []func()
	localItems           []model.Item
	remoteItems          []model.Item
	transferJobs         []model.TransferJob
	profiles             []model.PublicProfile
	settings             model.Settings
	localCurrent         string
	remoteCurrent        string
	selectedProfileID    string
	connected            bool
	connectionBusy       bool
	connectionGeneration uint64
	healthCheckRunning   bool
	connectionCancel     context.CancelFunc
	healthCheckCancel    context.CancelFunc
	localNavCancel       context.CancelFunc
	remoteNavCancel      context.CancelFunc
	transferSeq          int64
	seenDone             map[string]bool
	queuePaused          bool
	closing              bool
	localNavSeq          uint64
	remoteNavSeq         uint64
	localSortColumn      int
	remoteSortColumn     int
	localSortDescending  bool
	remoteSortDescending bool
}

var apps sync.Map
var wndProcPtr = syscall.NewCallback(wndProc)

func Run(engine *api.Engine, version string) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	_, _, _ = setProcessDPI.Call(^uintptr(3))
	icc := initCommonControls{Size: uint32(unsafe.Sizeof(initCommonControls{})), ICC: 0x00000001 | 0x00004000}
	_, _, _ = initCommonControlsEx.Call(uintptr(unsafe.Pointer(&icc)))

	startupSettings, settingsErr := engine.Settings()
	if settingsErr != nil {
		startupSettings = config.DefaultSettings()
	}
	setActiveTheme(startupSettings.Appearance)
	platform.SetDialogAppearance(activeThemeIsDark())

	hinst, _, _ := getModuleHandleW.Call(0)
	cursor, _, _ := loadCursorW.Call(0, 32512)
	icon, _, _ := loadIconW.Call(hinst, 1)
	brush, _, _ := createSolidBrush.Call(windowColor())
	panelBrush, _, _ := createSolidBrush.Call(panelColor())
	className := wstr("GhostFTP.NativeWindow")
	wc := wndClassEx{
		CbSize:     uint32(unsafe.Sizeof(wndClassEx{})),
		WndProc:    wndProcPtr,
		Instance:   hinst,
		Icon:       icon,
		Cursor:     cursor,
		Background: brush,
		ClassName:  className,
		IconSm:     icon,
	}
	if r, _, err := registerClassExW.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
		if brush != 0 {
			deleteObject.Call(brush)
		}
		if panelBrush != 0 {
			deleteObject.Call(panelBrush)
		}
		return fmt.Errorf("unable to register GhostFTP window: %v", err)
	}

	a := &app{
		engine: engine, version: version, remoteCurrent: "/", seenDone: map[string]bool{},
		brush: brush, panelBrush: panelBrush, buttons: make(map[uintptr]buttonVisual),
		settings: startupSettings,
	}
	hwnd, _, err := createWindowExW.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(wstr(brand.ProductName+" "+version+" — "+brand.Company))),
		wsOverlappedWindow,
		40, 30, 1200, 780,
		0, 0, hinst, 0,
	)
	if hwnd == 0 {
		if brush != 0 {
			deleteObject.Call(brush)
		}
		if panelBrush != 0 {
			deleteObject.Call(panelBrush)
		}
		return fmt.Errorf("unable to open GhostFTP window: %v", err)
	}
	a.hwnd = hwnd
	a.dpi = windowDPI(hwnd)
	apps.Store(hwnd, a)
	x, y, w, h := a.preferredWindowBounds()
	moveWindow.Call(hwnd, uintptr(a.scale(x)), uintptr(a.scale(y)), uintptr(a.scale(w)), uintptr(a.scale(h)), 0)
	applyDarkTitleBar(hwnd)
	if err := a.createControls(hinst); err != nil {
		apps.Delete(hwnd)
		destroyWindow.Call(hwnd)
		for _, f := range []uintptr{a.font, a.titleFont, a.smallFont, a.iconFont} {
			if f != 0 {
				deleteObject.Call(f)
			}
		}
		for _, b := range []uintptr{a.brush, a.panelBrush} {
			if b != 0 {
				deleteObject.Call(b)
			}
		}
		return err
	}
	var client rect
	if r, _, _ := getClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client))); r != 0 {
		a.layout(int(client.Right-client.Left), int(client.Bottom-client.Top))
	} else {
		a.layout(a.scale(1180), a.scale(760))
	}
	a.updateActionControls()
	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)
	setTimer.Call(hwnd, 1, 1000, 0)
	accelerators := []accel{
		{FVirt: fvirtKey, Key: vkF5, Cmd: idRefreshAll},
		{FVirt: fvirtKey | fcontrol, Key: 'S', Cmd: idSaveProfile},
		{FVirt: fvirtKey | fcontrol, Key: 'L', Cmd: idFocusLocalPath},
		{FVirt: fvirtKey | fcontrol | fshiftKeyboard, Key: 'L', Cmd: idFocusRemotePath},
	}
	accelTable, _, _ := createAcceleratorTableW.Call(uintptr(unsafe.Pointer(&accelerators[0])), uintptr(len(accelerators)))
	if accelTable != 0 {
		defer destroyAcceleratorTable.Call(accelTable)
	}
	a.loadProfiles()
	if settingsErr != nil {
		a.setStatus(a.userMessage(settingsErr, "settings.load_failed"))
	} else {
		a.loadSettings()
	}
	a.refreshLocal("")
	if settingsErr == nil {
		a.setStatus(a.tr("status.ready"))
	}

	var m msg
	for {
		r, _, callErr := getMessageW.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(r) == -1 {
			if callErr != nil && callErr != syscall.Errno(0) {
				return fmt.Errorf("GhostFTP window messages are unavailable: %w", callErr)
			}
			return fmt.Errorf("GhostFTP window messages are unavailable")
		}
		if r == 0 {
			break
		}
		if accelTable != 0 {
			if handled, _, _ := translateAcceleratorW.Call(hwnd, accelTable, uintptr(unsafe.Pointer(&m))); handled != 0 {
				continue
			}
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&m)))
	}
	apps.Delete(hwnd)
	for _, f := range []uintptr{a.font, a.titleFont, a.smallFont, a.iconFont} {
		if f != 0 {
			deleteObject.Call(f)
		}
	}
	for _, b := range []uintptr{a.brush, a.panelBrush} {
		if b != 0 {
			deleteObject.Call(b)
		}
	}
	return nil
}

func wndProc(hwnd uintptr, message uint32, wParam, lParam uintptr) uintptr {
	v, ok := apps.Load(hwnd)
	if !ok {
		r, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
		return r
	}
	a := v.(*app)
	switch message {
	case wmGetMinMaxInfo:
		if lParam != 0 {
			info := minMaxInfoFromLParam(lParam)
			info.MinTrackSize.X = int32(a.scale(premiumMinWidth))
			info.MinTrackSize.Y = int32(a.scale(premiumMinHeight))
			minMaxInfoToLParam(lParam, info)
		}
		return 0
	case wmSize:
		w := int(lParam & 0xffff)
		h := int((lParam >> 16) & 0xffff)
		a.reflowWorkspace(w, h)
		return 0
	case wmDpiChanged:
		newDPI := uint32((wParam >> 16) & 0xffff)
		if newDPI == 0 {
			newDPI = 96
		}
		a.applyDPI(newDPI)
		if lParam != 0 {
			r := rectFromLParam(lParam)
			moveWindow.Call(hwnd, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right-r.Left), uintptr(r.Bottom-r.Top), 1)
		}
		return 0
	case wmCommand:
		id := int(wParam & 0xffff)
		notify := int((wParam >> 16) & 0xffff)
		if id == idProtocol && notify == cbnSelChange {
			a.syncDefaultPort()
			a.updateProtocolControls()
			a.refineWorkspaceLayout()
			return 0
		}
		if id == idProfiles && notify == cbnSelChange {
			a.selectProfile()
			return 0
		}
		if id == idLanguage && notify == cbnSelChange {
			a.changeLanguageFromUI()
			a.refineWorkspaceLayout()
			return 0
		}
		if notify == bnClicked || notify == acceleratorCommandNotification {
			a.command(id)
			return 0
		}
	case wmDrawItem:
		if lParam != 0 {
			d := drawItemFromLParam(lParam)
			if a.drawButton(&d) {
				return 1
			}
		}
	case wmNotify:
		if lParam != 0 {
			h := nmhdrFromLParam(lParam)
			if h.Code == nmCustomDraw && a.isWorkspaceHeader(h.HwndFrom) {
				return a.drawWorkspaceHeader(lParam)
			}
			if h.Code == lvnColumnClick && (h.HwndFrom == a.localList || h.HwndFrom == a.remoteList) {
				n := nmListViewFromLParam(lParam)
				a.handleFileColumnClick(h.HwndFrom, int(n.SubItem))
				return 0
			}
			if h.Code == lvnKeyDown && (h.HwndFrom == a.localList || h.HwndFrom == a.remoteList) {
				n := nmLVKeyDownFromLParam(lParam)
				if a.handleFileListKey(h.HwndFrom, n.VKey) {
					return 0
				}
			}
			if h.Code == lvnItemChanged && (h.HwndFrom == a.localList || h.HwndFrom == a.remoteList || h.HwndFrom == a.transferList) {
				a.updateActionControls()
				return 0
			}
			if h.Code == nmDblClk {
				if h.HwndFrom == a.localList {
					a.openSelectedLocal()
				}
				if h.HwndFrom == a.remoteList {
					a.openSelectedRemote()
				}
				return 0
			}
		}
	case wmTimer:
		if wParam == 1 {
			a.refreshTransfers()
		}
		return 0
	case wmCtlColorEdit, wmCtlColorBtn, wmCtlColorStatic:
		color := textColor()
		if lParam == a.brandTitle {
			color = textColor()
		} else if lParam == a.brandSubtitle || lParam == a.sectionLocal || lParam == a.sectionRemote || lParam == a.sectionTransfers || lParam == a.status {
			color = mutedColor()
		} else if lParam == a.connectionBadge {
			if a.connected {
				color = successColor()
			} else {
				color = mutedColor()
			}
		}
		setTextColor.Call(wParam, color)
		if message == wmCtlColorStatic && a.panelBrush != 0 {
			setBkColor.Call(wParam, panelColor())
			return a.panelBrush
		}
		setBkColor.Call(wParam, windowColor())
		return a.brush
	case wmAppDispatch:
		a.runDispatch()
		return 0
	case wmClose:
		if a.connected || a.connectionBusy || a.hasActiveTransfers() {
			if !platform.ConfirmDialog("Ghost FTP", closeQuestion(a.languageCode()), closeBody(a.languageCode())) {
				return 0
			}
		}
		destroyWindow.Call(hwnd)
		return 0
	case wmDestroy:
		a.cancelConnectionAttempt()
		a.cancelHealthCheck()
		if a.localNavCancel != nil {
			a.localNavCancel()
			a.localNavCancel = nil
		}
		if a.remoteNavCancel != nil {
			a.remoteNavCancel()
			a.remoteNavCancel = nil
		}
		if diagnostics := a.sidebarDiagnosticsButton(); diagnostics != 0 {
			delete(a.buttons, diagnostics)
		}
		sidebarDiagnostics.Delete(a.hwnd)
		a.mu.Lock()
		a.closing = true
		a.mu.Unlock()
		killTimer.Call(hwnd, 1)
		postQuitMessage.Call(0)
		return 0
	}
	r, _, _ := defWindowProcW.Call(hwnd, uintptr(message), wParam, lParam)
	return r
}
