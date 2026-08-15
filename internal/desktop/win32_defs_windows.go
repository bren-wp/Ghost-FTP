//go:build windows

package desktop

import (
	"syscall"
	"unsafe"
)

var rtlMoveMemory = syscall.NewLazyDLL("kernel32.dll").NewProc("RtlMoveMemory")

const (
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	wsChild            = 0x40000000
	wsTabStop          = 0x00010000
	wsBorder           = 0x00800000
	wsVScroll          = 0x00200000

	esAutoHScroll   = 0x0080
	esPassword      = 0x0020
	bsPushButton    = 0x00000000
	bsOwnerDraw     = 0x0000000B
	cbsDropDownList = 0x0003

	lvsReport          = 0x0001
	lvsShowSelAlways   = 0x0008
	lvsExGridLines     = 0x00000001
	lvsExFullRowSelect = 0x00000020
	lvsExDoubleBuffer  = 0x00010000
	lvsilSmall         = 1

	wmDestroy        = 0x0002
	wmDrawItem       = 0x002B
	wmClose          = 0x0010
	wmSize           = 0x0005
	wmDpiChanged     = 0x02E0
	wmSetFont        = 0x0030
	wmCommand        = 0x0111
	wmTimer          = 0x0113
	wmNotify         = 0x004E
	wmCtlColorEdit   = 0x0133
	wmCtlColorBtn    = 0x0135
	wmCtlColorStatic = 0x0138
	wmAppDispatch    = 0x8001

	swShow   = 5
	vkF5     = 0x74
	fvirtKey = 0x01
	fcontrol = 0x08

	bnClicked    = 0
	cbnSelChange = 1
	cbAddString  = 0x0143
	cbGetCurSel  = 0x0147
	cbSetCurSel  = 0x014E

	emSetCueBanner = 0x1501
	emSetLimitText = 0x00C5

	lvmFirst                    = 0x1000
	lvmSetImageList             = lvmFirst + 3
	lvmDeleteAllItems           = lvmFirst + 9
	lvmGetNextItem              = lvmFirst + 12
	lvmSetExtendedListViewStyle = lvmFirst + 54
	lvmSetColumnWidth           = lvmFirst + 30
	lvmInsertColumnW            = lvmFirst + 97
	lvmInsertItemW              = lvmFirst + 77
	lvmSetItemTextW             = lvmFirst + 116
	lvmSetBkColor               = lvmFirst + 1
	lvmSetTextBkColor           = lvmFirst + 38
	lvmSetTextColor             = lvmFirst + 36
	lvniSelected                = 0x0002
	lvcfFmt                     = 0x0001
	lvcfWidth                   = 0x0002
	lvcfText                    = 0x0004
	lvifText                    = 0x0001
	lvifImage                   = 0x0002

	shgfiSmallIcon         = 0x000000001
	shgfiUseFileAttributes = 0x000000010
	shgfiSysIconIndex      = 0x000004000
	fileAttributeDirectory = 0x00000010
	fileAttributeNormal    = 0x00000080

	nmDblClk = 0xFFFFFFFD

	odsSelected = 0x0001
	odsDisabled = 0x0004
	odsFocus    = 0x0010

	psSolid           = 0
	transparentBkMode = 1
	dtCenter          = 0x00000001
	dtVCenter         = 0x00000004
	dtSingleLine      = 0x00000020
	dtNoPrefix        = 0x00000800
	dtEndEllipsis     = 0x00008000
	dtLeft            = 0x00000000

	idProfiles      = 91
	idSaveProfile   = 92
	idRemoveProfile = 93
	idSettings      = 94
	idAbout         = 95

	idProtocol   = 101
	idHost       = 102
	idPort       = 103
	idUser       = 104
	idPass       = 105
	idConnect    = 106
	idDisconnect = 107
	idKeyPath    = 108
	idChooseKey  = 109
	idPassphrase = 110

	idLocalPath    = 201
	idLocalUp      = 202
	idLocalRefresh = 203
	idLocalList    = 204
	idLocalChoose  = 205
	idLocalMkdir   = 206
	idLocalRename  = 207
	idLocalDelete  = 208

	idRemotePath    = 301
	idRemoteUp      = 302
	idRemoteRefresh = 303
	idRemoteList    = 304
	idRemoteMkdir   = 305
	idRemoteRename  = 306
	idRemoteDelete  = 307
	idRemoteChmod   = 308

	idUpload   = 401
	idDownload = 402

	idTransferList = 501
	idPauseQueue   = 502
	idResumeQueue  = 503
	idCancelJob    = 504
	idRetryJob     = 505
	idClearQueue   = 506
	idRefreshAll   = 507

	idStatus = 601
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	comctl32 = syscall.NewLazyDLL("comctl32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	uxtheme  = syscall.NewLazyDLL("uxtheme.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	ntdll    = syscall.NewLazyDLL("ntdll.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")

	registerClassExW        = user32.NewProc("RegisterClassExW")
	createWindowExW         = user32.NewProc("CreateWindowExW")
	destroyWindow           = user32.NewProc("DestroyWindow")
	defWindowProcW          = user32.NewProc("DefWindowProcW")
	showWindow              = user32.NewProc("ShowWindow")
	updateWindow            = user32.NewProc("UpdateWindow")
	getMessageW             = user32.NewProc("GetMessageW")
	getClientRect           = user32.NewProc("GetClientRect")
	translateMessage        = user32.NewProc("TranslateMessage")
	dispatchMessageW        = user32.NewProc("DispatchMessageW")
	postQuitMessage         = user32.NewProc("PostQuitMessage")
	postMessageW            = user32.NewProc("PostMessageW")
	sendMessageW            = user32.NewProc("SendMessageW")
	moveWindow              = user32.NewProc("MoveWindow")
	setWindowTextW          = user32.NewProc("SetWindowTextW")
	getWindowTextW          = user32.NewProc("GetWindowTextW")
	getWindowTextLengthW    = user32.NewProc("GetWindowTextLengthW")
	enableWindow            = user32.NewProc("EnableWindow")
	loadCursorW             = user32.NewProc("LoadCursorW")
	loadIconW               = user32.NewProc("LoadIconW")
	setTimer                = user32.NewProc("SetTimer")
	killTimer               = user32.NewProc("KillTimer")
	setProcessDPI           = user32.NewProc("SetProcessDpiAwarenessContext")
	getDpiForWindow         = user32.NewProc("GetDpiForWindow")
	createAcceleratorTableW = user32.NewProc("CreateAcceleratorTableW")
	translateAcceleratorW   = user32.NewProc("TranslateAcceleratorW")
	destroyAcceleratorTable = user32.NewProc("DestroyAcceleratorTable")
	invalidateRect          = user32.NewProc("InvalidateRect")
	drawTextW               = user32.NewProc("DrawTextW")
	drawFocusRect           = user32.NewProc("DrawFocusRect")
	setBkColor              = gdi32.NewProc("SetBkColor")
	setTextColor            = gdi32.NewProc("SetTextColor")
	createSolidBrush        = gdi32.NewProc("CreateSolidBrush")
	createPen               = gdi32.NewProc("CreatePen")
	selectObject            = gdi32.NewProc("SelectObject")
	roundRect               = gdi32.NewProc("RoundRect")
	setBkMode               = gdi32.NewProc("SetBkMode")

	getModuleHandleW      = kernel32.NewProc("GetModuleHandleW")
	initCommonControlsEx  = comctl32.NewProc("InitCommonControlsEx")
	createFontW           = gdi32.NewProc("CreateFontW")
	deleteObject          = gdi32.NewProc("DeleteObject")
	setWindowTheme        = uxtheme.NewProc("SetWindowTheme")
	dwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	shGetFileInfoW        = shell32.NewProc("SHGetFileInfoW")
	rtlGetVersion         = ntdll.NewProc("RtlGetVersion")
)

type wndClassEx struct {
	CbSize     uint32
	Style      uint32
	WndProc    uintptr
	ClsExtra   int32
	WndExtra   int32
	Instance   uintptr
	Icon       uintptr
	Cursor     uintptr
	Background uintptr
	MenuName   *uint16
	ClassName  *uint16
	IconSm     uintptr
}

type rect struct {
	Left, Top, Right, Bottom int32
}

type drawItemStruct struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   uintptr
	HDC        uintptr
	RcItem     rect
	ItemData   uintptr
}

type rtlOsVersionInfo struct {
	Size, Major, Minor, Build, Platform uint32
	CSDVersion                          [128]uint16
}

type point struct{ X, Y int32 }
type msg struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      point
	Private uint32
}

type initCommonControls struct {
	Size uint32
	ICC  uint32
}

type accel struct {
	FVirt byte
	Pad   byte
	Key   uint16
	Cmd   uint16
}

type nmhdr struct {
	HwndFrom uintptr
	IDFrom   uintptr
	Code     uint32
}

type lvColumn struct {
	Mask      uint32
	Fmt       int32
	Cx        int32
	Text      *uint16
	TextMax   int32
	SubItem   int32
	Image     int32
	Order     int32
	CxMin     int32
	CxDefault int32
	CxIdeal   int32
}

type lvItem struct {
	Mask      uint32
	Item      int32
	SubItem   int32
	State     uint32
	StateMask uint32
	Text      *uint16
	TextMax   int32
	Image     int32
	Param     uintptr
	Indent    int32
	GroupID   int32
	Columns   uint32
	ColumnPtr *uint32
	ColFmt    *int32
	Group     int32
}

type shFileInfo struct {
	Icon        uintptr
	IconIndex   int32
	Attributes  uint32
	DisplayName [260]uint16
	TypeName    [80]uint16
}

func wstr(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func rgb(r, g, b byte) uintptr { return uintptr(r) | uintptr(g)<<8 | uintptr(b)<<16 }

func rectFromLParam(p uintptr) rect {
	var r rect
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&r)), p, unsafe.Sizeof(r))
	}
	return r
}

func nmhdrFromLParam(p uintptr) nmhdr {
	var h nmhdr
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&h)), p, unsafe.Sizeof(h))
	}
	return h
}

func drawItemFromLParam(p uintptr) drawItemStruct {
	var d drawItemStruct
	if p != 0 {
		rtlMoveMemory.Call(uintptr(unsafe.Pointer(&d)), p, unsafe.Sizeof(d))
	}
	return d
}
