#!/usr/bin/env python3
from pathlib import Path
import os
import subprocess

ROOT = Path(__file__).resolve().parents[1]

def read(p): return (ROOT/p).read_text(encoding='utf-8')
def write(p,s): (ROOT/p).write_text(s, encoding='utf-8')
def repl(p,old,new):
    s=read(p)
    if s.count(old)!=1:
        raise SystemExit(f'guard failed {p}: {s.count(old)} occurrences')
    write(p,s.replace(old,new,1))
def run(*a,env=None):
    e=os.environ.copy(); e.update(env or {})
    print('+',' '.join(a),flush=True)
    subprocess.run(a,cwd=ROOT,env=e,check=True)

# Full wordmark geometry: keep it robust at the real 976px screenshot width.
repl('internal/desktop/chrome_windows.go', '''func (a *app) refineBrandHeader() {
\tlogo := a.ensureBrandLogo()
\tif logo == 0 {
\t\treturn
\t}
\t// Geometry for the title/subtitle remains owned by app.layout. This helper
\t// only anchors the canonical PE icon so the full “Ghost FTP” wordmark never
\t// gets clipped by a second competing layout rule.
\ta.move(logo, 14, 11, 32, 32)
}
''', '''func (a *app) refineBrandHeader() {
\tlogo := a.ensureBrandLogo()
\tif logo == 0 {
\t\treturn
\t}
\ta.move(logo, 14, 11, 32, 32)

\t// Reserve a fixed wordmark gutter large enough for the real Segoe UI bold
\t// rendering seen on Windows runners. Keep the subtitle responsive inside
\t// the remaining header space rather than allowing it to overlap the title.
\tconst titleX, titleWidth, subtitleX = 54, 168, 230
\ta.move(a.brandTitle, titleX, 10, titleWidth, 35)
\tvar client rect
\tif ok, _, _ := getClientRect.Call(a.hwnd, uintptr(unsafe.Pointer(&client))); ok != 0 {
\t\tlogicalWidth := a.unscale(int(client.Right - client.Left))
\t\tsubtitleWidth := logicalWidth - subtitleX - 320
\t\tif subtitleWidth < 120 {
\t\t\tsubtitleWidth = 120
\t\t}
\t\ta.move(a.brandSubtitle, subtitleX, 17, subtitleWidth, 20)
\t}
}
''')

# Native dark menu: owner draw all real menu items and set the menu background
# explicitly instead of relying on the user's Windows light/dark preference.
repl('internal/desktop/menu_windows.go', '''\tmfString    = 0x0000
\tmfPopup     = 0x0010
\tmfSeparator = 0x0800
''', '''\tmfString    = 0x0000
\tmfPopup     = 0x0010
\tmfOwnerDraw = 0x0100
\tmfSeparator = 0x0800
''')
repl('internal/desktop/menu_windows.go', '''func appendMenuItem(menu uintptr, id int, label string) {
\tappendMenuW.Call(menu, mfString, uintptr(id), uintptr(unsafe.Pointer(wstr(label))))
}

func appendMenuSeparator(menu uintptr) { appendMenuW.Call(menu, mfSeparator, 0, 0) }

func appendPopup(root, popup uintptr, label string) {
\tappendMenuW.Call(root, mfPopup, popup, uintptr(unsafe.Pointer(wstr(label))))
}
''', '''func appendMenuItem(menu uintptr, id int, label string) {
\tkey := uintptr(id)
\tregisterMenuVisual(key, label, false)
\tappendMenuW.Call(menu, mfString|mfOwnerDraw, uintptr(id), key)
}

func appendMenuSeparator(menu uintptr) { appendMenuW.Call(menu, mfSeparator, 0, 0) }

func appendPopup(root, popup uintptr, label string) {
\tregisterMenuVisual(popup, label, true)
\tappendMenuW.Call(root, mfPopup|mfOwnerDraw, popup, popup)
}
''')
repl('internal/desktop/menu_windows.go', '''func (a *app) installMainMenu() {
\tif a == nil || a.hwnd == 0 {
\t\treturn
\t}
''', '''func (a *app) installMainMenu() {
\tif a == nil || a.hwnd == 0 {
\t\treturn
\t}
\tresetMenuVisuals()
''')
repl('internal/desktop/menu_windows.go', '''\tappendPopup(root, toolsMenu, words[7])
\tappendPopup(root, helpMenu, words[4])

\told, _, _ := getMenuW.Call(a.hwnd)
''', '''\tappendPopup(root, toolsMenu, words[7])
\tappendPopup(root, helpMenu, words[4])
\tapplyDarkMenuBackground(root, a.panelBrush)

\told, _, _ := getMenuW.Call(a.hwnd)
''')

write('internal/desktop/menu_draw_windows.go', r'''//go:build windows

package desktop

import (
    "sync"
    "unicode/utf8"
    "unsafe"
)

const (
    odtMenu      = 1
    odsGrayed    = 0x0002
    wmMeasureItem = 0x002C
    mimBackground = 0x00000002
    mimApplyToSubmenus = 0x80000000
)

type menuVisual struct {
    label string
    root  bool
}

var menuVisualState = struct {
    sync.RWMutex
    items map[uintptr]menuVisual
}{items: make(map[uintptr]menuVisual)}

var (
    fillRectW    = user32.NewProc("FillRect")
    setMenuInfoW = user32.NewProc("SetMenuInfo")
)

type measureItemStruct struct {
    CtlType    uint32
    CtlID      uint32
    ItemID     uint32
    ItemWidth  uint32
    ItemHeight uint32
    ItemData   uintptr
}

type menuInfo struct {
    CbSize          uint32
    FMask           uint32
    DwStyle         uint32
    CyMax           uint32
    HbrBack         uintptr
    DwContextHelpID uint32
    DwMenuData      uintptr
}

func resetMenuVisuals() {
    menuVisualState.Lock()
    menuVisualState.items = make(map[uintptr]menuVisual)
    menuVisualState.Unlock()
}

func registerMenuVisual(key uintptr, label string, root bool) {
    menuVisualState.Lock()
    menuVisualState.items[key] = menuVisual{label: label, root: root}
    menuVisualState.Unlock()
}

func menuVisualFor(key uintptr) (menuVisual, bool) {
    menuVisualState.RLock()
    v, ok := menuVisualState.items[key]
    menuVisualState.RUnlock()
    return v, ok
}

func measureItemFromLParam(p uintptr) measureItemStruct {
    var item measureItemStruct
    if p != 0 {
        rtlMoveMemory.Call(uintptr(unsafe.Pointer(&item)), p, unsafe.Sizeof(item))
    }
    return item
}

func measureItemToLParam(p uintptr, item measureItemStruct) {
    if p != 0 {
        rtlMoveMemory.Call(p, uintptr(unsafe.Pointer(&item)), unsafe.Sizeof(item))
    }
}

func (a *app) measureMenuItem(lParam uintptr) bool {
    if lParam == 0 {
        return false
    }
    item := measureItemFromLParam(lParam)
    if item.CtlType != odtMenu {
        return false
    }
    visual, ok := menuVisualFor(item.ItemData)
    if !ok {
        return false
    }
    glyphs := utf8.RuneCountInString(visual.label)
    width := 30 + glyphs*8
    height := 28
    if visual.root {
        width = 20 + glyphs*8
        height = 24
    }
    if width < 48 {
        width = 48
    }
    item.ItemWidth = uint32(width)
    item.ItemHeight = uint32(height)
    measureItemToLParam(lParam, item)
    return true
}

func (a *app) drawMenuItem(d *drawItemStruct) bool {
    if a == nil || d == nil || d.CtlType != odtMenu {
        return false
    }
    visual, ok := menuVisualFor(d.ItemData)
    if !ok {
        return false
    }
    brush := a.panelBrush
    temporary := uintptr(0)
    if d.ItemState&odsSelected != 0 {
        temporary, _, _ = createSolidBrush.Call(selectionColor())
        if temporary != 0 {
            brush = temporary
        }
    }
    if brush != 0 {
        fillRectW.Call(d.HDC, uintptr(unsafe.Pointer(&d.RcItem)), brush)
    }
    if temporary != 0 {
        deleteObject.Call(temporary)
    }
    color := textColor()
    if d.ItemState&(odsDisabled|odsGrayed) != 0 {
        color = mutedColor()
    }
    setTextColor.Call(d.HDC, color)
    setBkMode.Call(d.HDC, transparentBkMode)
    rc := d.RcItem
    if visual.root {
        rc.Left += 8
        rc.Right -= 8
    } else {
        rc.Left += 12
        rc.Right -= 12
    }
    label := wstr(visual.label)
    drawTextW.Call(d.HDC, uintptr(unsafe.Pointer(label)), ^uintptr(0), uintptr(unsafe.Pointer(&rc)), dtLeft|dtVCenter|dtSingleLine|dtNoPrefix)
    return true
}

func applyDarkMenuBackground(menu, brush uintptr) {
    if menu == 0 || brush == 0 {
        return
    }
    info := menuInfo{
        CbSize:  uint32(unsafe.Sizeof(menuInfo{})),
        FMask:   mimBackground | mimApplyToSubmenus,
        HbrBack: brush,
    }
    setMenuInfoW.Call(menu, uintptr(unsafe.Pointer(&info)))
}
''')

# Deterministic dark header text through NM_CUSTOMDRAW. DarkMode_ItemsView gives
# the face; custom draw supplies the readable text color even in light OS mode.
write('internal/desktop/header_draw_windows.go', r'''//go:build windows

package desktop

import "unsafe"

const (
    nmCustomDraw       = 0xFFFFFFF4
    cddsPrepaint       = 0x00000001
    cddsItemPrepaint   = 0x00010001
    cdrfNewFont        = 0x00000002
    cdrfNotifyItemDraw = 0x00000020
)

type nmCustomDrawStruct struct {
    Hdr        nmhdr
    DrawStage  uint32
    HDC        uintptr
    Rc         rect
    ItemSpec   uintptr
    ItemState  uint32
    ItemLParam uintptr
}

func customDrawFromLParam(p uintptr) nmCustomDrawStruct {
    var d nmCustomDrawStruct
    if p != 0 {
        rtlMoveMemory.Call(uintptr(unsafe.Pointer(&d)), p, unsafe.Sizeof(d))
    }
    return d
}

func headerForList(list uintptr) uintptr {
    if list == 0 {
        return 0
    }
    h, _, _ := sendMessageW.Call(list, lvmGetHeader, 0, 0)
    return h
}

func (a *app) isWorkspaceHeader(hwnd uintptr) bool {
    if a == nil || hwnd == 0 {
        return false
    }
    return hwnd == headerForList(a.localList) || hwnd == headerForList(a.remoteList) || hwnd == headerForList(a.transferList)
}

func (a *app) drawWorkspaceHeader(lParam uintptr) uintptr {
    d := customDrawFromLParam(lParam)
    switch d.DrawStage {
    case cddsPrepaint:
        return cdrfNotifyItemDraw
    case cddsItemPrepaint:
        setTextColor.Call(d.HDC, textColor())
        setBkColor.Call(d.HDC, panelColor())
        setBkMode.Call(d.HDC, transparentBkMode)
        return cdrfNewFont
    default:
        return 0
    }
}
''')

# Add LVM_GETHEADER constant.
repl('internal/desktop/win32_defs_windows.go', '''\tlvmGetNextItem              = lvmFirst + 12
\tlvmSetColumnWidth           = lvmFirst + 30
''', '''\tlvmGetNextItem              = lvmFirst + 12
\tlvmSetColumnWidth           = lvmFirst + 30
\tlvmGetHeader                = lvmFirst + 31
''')

# Wire menu measure/draw and header custom draw into the canonical wndproc.
repl('internal/desktop/windows.go', '''\tcase wmDrawItem:
\t\tif lParam != 0 {
\t\t\td := drawItemFromLParam(lParam)
\t\t\tif a.drawButton(&d) {
\t\t\t\treturn 1
\t\t\t}
\t\t}
''', '''\tcase wmMeasureItem:
\t\tif a.measureMenuItem(lParam) {
\t\t\treturn 1
\t\t}
\tcase wmDrawItem:
\t\tif lParam != 0 {
\t\t\td := drawItemFromLParam(lParam)
\t\t\tif a.drawMenuItem(&d) {
\t\t\t\treturn 1
\t\t\t}
\t\t\tif a.drawButton(&d) {
\t\t\t\treturn 1
\t\t\t}
\t\t}
''')
repl('internal/desktop/windows.go', '''\tcase wmNotify:
\t\tif lParam != 0 {
\t\t\th := nmhdrFromLParam(lParam)
\t\t\tif h.Code == lvnItemChanged''', '''\tcase wmNotify:
\t\tif lParam != 0 {
\t\t\th := nmhdrFromLParam(lParam)
\t\t\tif h.Code == nmCustomDraw && a.isWorkspaceHeader(h.HwndFrom) {
\t\t\t\treturn a.drawWorkspaceHeader(lParam)
\t\t\t}
\t\t\tif h.Code == lvnItemChanged''')

# Add explicit visual regression guards for the real screenshot defects.
test = read('scripts/test_windows_visual_regressions.py')
needle='''    def test_disconnected_remote_list_keeps_dark_enabled_surface(self):\n'''
block='''    def test_full_wordmark_dark_headers_and_ownerdrawn_menu_are_guarded(self):\n        chrome = self.read("internal/desktop/chrome_windows.go")\n        header = self.read("internal/desktop/header_draw_windows.go")\n        menu = self.read("internal/desktop/menu_draw_windows.go")\n        wnd = self.read("internal/desktop/windows.go")\n        self.assertIn("titleWidth, subtitleX = 54, 168, 230", chrome)\n        self.assertIn("nmCustomDraw", header)\n        self.assertIn("setTextColor.Call(d.HDC, textColor())", header)\n        self.assertIn("mfOwnerDraw", self.read("internal/desktop/menu_windows.go"))\n        self.assertIn("applyDarkMenuBackground", menu)\n        self.assertIn("a.measureMenuItem(lParam)", wnd)\n        self.assertIn("a.drawMenuItem(&d)", wnd)\n\n'''
if test.count(needle)!=1: raise SystemExit('test insertion guard failed')
write('scripts/test_windows_visual_regressions.py',test.replace(needle,block+needle,1))

files=['internal/desktop/chrome_windows.go','internal/desktop/menu_windows.go','internal/desktop/menu_draw_windows.go','internal/desktop/header_draw_windows.go','internal/desktop/win32_defs_windows.go','internal/desktop/windows.go']
run('gofmt','-w',*files)
run('go','telemetry','off')
run('go','test','./...')
run('go','vet','./...')
run('python','scripts/audit_security.py')
run('python','scripts/audit_privacy.py')
run('python','scripts/audit_repository.py')
run('python','scripts/audit_platform_contract.py')
run('python','-m','unittest','scripts.test_windows_visual_regressions')
run('go','build','-o',str(ROOT/'.tmp-polish-amd64.exe'),'./cmd/ghostftp',env={'GOOS':'windows','GOARCH':'amd64','CGO_ENABLED':'0'})
run('go','build','-o',str(ROOT/'.tmp-polish-386.exe'),'./cmd/ghostftp',env={'GOOS':'windows','GOARCH':'386','CGO_ENABLED':'0'})
(ROOT/'.tmp-polish-amd64.exe').unlink(missing_ok=True); (ROOT/'.tmp-polish-386.exe').unlink(missing_ok=True)
run('git','config','user.name','ghostftp-quality-bot'); run('git','config','user.email','actions@users.noreply.github.com')
run('git','add','internal/desktop','scripts/test_windows_visual_regressions.py')
run('git','commit','-m','Polish Ghost FTP 0.2.1 native Windows chrome')
branch=os.environ.get('GITHUB_REF_NAME','work/windows-ui-stability-logo-20260906')
run('git','push','origin',f'HEAD:{branch}')
print('GHOSTFTP_WINDOWS_VISUAL_POLISH=PASS')
