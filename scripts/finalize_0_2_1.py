#!/usr/bin/env python3
from __future__ import annotations

import os
import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def write(path: str, text: str) -> None:
    target = ROOT / path
    target.parent.mkdir(parents=True, exist_ok=True)
    target.write_text(text, encoding="utf-8")


def replace_once(path: str, old: str, new: str) -> None:
    text = read(path)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"PATCH_GUARD_FAILED {path}: expected 1 occurrence, found {count}: {old[:100]!r}")
    write(path, text.replace(old, new, 1))


def replace_all_checked(path: str, replacements: list[tuple[str, str]]) -> None:
    text = read(path)
    for old, new in replacements:
        if old not in text:
            raise SystemExit(f"PATCH_GUARD_FAILED {path}: missing marker {old[:100]!r}")
        text = text.replace(old, new)
    write(path, text)


def run(*args: str, env: dict[str, str] | None = None) -> None:
    merged = os.environ.copy()
    if env:
        merged.update(env)
    print("+", " ".join(args), flush=True)
    subprocess.run(args, cwd=ROOT, env=merged, check=True)


# --- Windows brand/header and native dark chrome ---
replace_once(
    "internal/desktop/chrome_windows.go",
    '''func (a *app) refineBrandHeader() {
\tlogo := a.ensureBrandLogo()
\tif logo == 0 {
\t\treturn
\t}
\t// The existing subtitle begins at x=166, so this keeps the original header
\t// rhythm while adding a real 32 px product mark instead of a text-only logo.
\ta.move(logo, 14, 11, 32, 32)
\ta.move(a.brandTitle, 54, 10, 106, 35)
}
''',
    '''func (a *app) refineBrandHeader() {
\tlogo := a.ensureBrandLogo()
\tif logo == 0 {
\t\treturn
\t}
\t// Geometry for the title/subtitle remains owned by app.layout. This helper
\t// only anchors the canonical PE icon so the full “Ghost FTP” wordmark never
\t// gets clipped by a second competing layout rule.
\ta.move(logo, 14, 11, 32, 32)
}
''',
)

replace_once(
    "internal/desktop/ui_windows.go",
    '''func applyDarkTitleBar(hwnd uintptr) {
\tvalue := int32(1)
\t_, _, _ = dwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
}

func applyDarkControl(hwnd uintptr, class string) {
\tswitch class {
\tcase "SysListView32", "COMBOBOX", "BUTTON":
\t\tsetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
\tcase "EDIT":
\t\tsetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_CFD"))), 0)
\t}
}
''',
    '''func applyDarkTitleBar(hwnd uintptr) {
\tenableImmersiveDarkMode(hwnd)
\tvalue := int32(1)
\t_, _, _ = dwmSetWindowAttribute.Call(hwnd, 20, uintptr(unsafe.Pointer(&value)), unsafe.Sizeof(value))
}

func applyDarkControl(hwnd uintptr, class string) {
\tswitch class {
\tcase "SysListView32", "BUTTON":
\t\tsetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
\tcase "COMBOBOX", "EDIT":
\t\tsetWindowTheme.Call(hwnd, uintptr(unsafe.Pointer(wstr("DarkMode_CFD"))), 0)
\t}
}
''',
)

replace_once(
    "internal/desktop/ui_windows.go",
    '''\ta.move(a.brandTitle, margin, headerY, 145, 35)
\tsubtitleX := margin + 152
''',
    '''\t// Reserve a stable icon gutter and enough width for the complete product
\t// name. The previous 106 px post-layout override visibly clipped “FTP”.
\ta.move(a.brandTitle, 54, headerY, 126, 35)
\tsubtitleX := 188
''',
)
replace_once(
    "internal/desktop/ui_windows.go",
    "\tinvalidateRect.Call(a.hwnd, 0, 1)\n}\n\nfunc (a *app) defaultFontControls()",
    "\tinvalidateRect.Call(a.hwnd, 0, 0)\n}\n\nfunc (a *app) defaultFontControls()",
)

write(
    "internal/desktop/dark_mode_windows.go",
    r'''//go:build windows

package desktop

import (
    "syscall"
    "unsafe"
)

var (
    loadLibraryW   = kernel32.NewProc("LoadLibraryW")
    freeLibrary    = kernel32.NewProc("FreeLibrary")
    getProcAddress = kernel32.NewProc("GetProcAddress")
)

// enableImmersiveDarkMode enables the native dark-menu/control path on modern
// Windows without adding a runtime dependency. The uxtheme ordinals are used
// only on Windows 10 1809+ where Microsoft ships the corresponding functions;
// older systems simply retain their normal system menu rendering.
func enableImmersiveDarkMode(hwnd uintptr) {
    build := windowsBuildNumber()
    if hwnd == 0 || build < 17763 {
        return
    }
    module, _, _ := loadLibraryW.Call(uintptr(unsafe.Pointer(wstr("uxtheme.dll"))))
    if module == 0 {
        return
    }
    defer freeLibrary.Call(module)

    preferred, _, _ := getProcAddress.Call(module, 135)
    if preferred != 0 {
        // 1809 exposes AllowDarkModeForApp(BOOL) at this ordinal; 1903+
        // exposes SetPreferredAppMode. Both safely accept these small values.
        mode := uintptr(1)
        if build >= 18362 {
            mode = 2 // AllowDark
        }
        syscall.SyscallN(preferred, mode)
    }
    allowWindow, _, _ := getProcAddress.Call(module, 133)
    if allowWindow != 0 {
        syscall.SyscallN(allowWindow, hwnd, 1)
    }
    flushMenus, _, _ := getProcAddress.Call(module, 136)
    if flushMenus != 0 {
        syscall.SyscallN(flushMenus)
    }
}
''',
)

# With immersive mode active, ItemsView supplies a dark header with light text.
replace_once(
    "internal/desktop/workspace_layout_windows.go",
    '''\t\t// Explorer's dark header theme keeps the column text readable on Windows
\t\t// versions where DarkMode_ItemsView falls back to a dark face with dark
\t\t// text. The screenshot pipeline guards this native-control regression.
\t\tsetWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
''',
    '''\t\t// Immersive dark mode is enabled before controls are created, allowing
\t\t// ItemsView to render a dark column header with readable system text.
\t\tsetWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_ItemsView"))), 0)
''',
)

write(
    "internal/desktop/layout_batch_windows.go",
    r'''//go:build windows

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
''',
)
replace_once(
    "internal/desktop/windows.go",
    '''\tcase wmSize:
\t\tw := int(lParam & 0xffff)
\t\th := int((lParam >> 16) & 0xffff)
\t\ta.layout(w, h)
\t\ta.refineWorkspaceLayout()
\t\treturn 0
''',
    '''\tcase wmSize:
\t\tw := int(lParam & 0xffff)
\t\th := int((lParam >> 16) & 0xffff)
\t\ta.reflowWorkspace(w, h)
\t\treturn 0
''',
)

# --- Windows Site Manager premium parity ---
replace_once(
    "internal/desktop/site_manager_windows.go",
    '''\t\tcase wmCtlColorEdit, wmCtlColorBtn, wmCtlColorStatic, siteWMCtlColorListBox:
''',
    '''\t\tcase wmDrawItem:
\t\t\tif lParam != 0 {
\t\t\t\td := drawItemFromLParam(lParam)
\t\t\t\tif state.parent.drawButton(&d) {
\t\t\t\t\treturn 1
\t\t\t\t}
\t\t\t}
\t\tcase wmCtlColorEdit, wmCtlColorBtn, wmCtlColorStatic, siteWMCtlColorListBox:
''',
)
replace_once(
    "internal/desktop/site_manager_windows.go",
    '''\t\tcase wmDestroy:
\t\t\tstate.closed = true
\t\t\treturn 0
''',
    '''\t\tcase wmDestroy:
\t\t\tfor _, button := range []uintptr{state.save, state.delete, state.connect, state.close} {
\t\t\t\tdelete(state.parent.buttons, button)
\t\t\t}
\t\t\tstate.closed = true
\t\t\treturn 0
''',
)
replace_once(
    "internal/desktop/site_manager_windows.go",
    '''\tstate.save = mk("BUTTON", parent.tr("profile.save"), wsTabStop, 310, 500, 146, 34, siteIDSave)
\tstate.delete = mk("BUTTON", parent.tr("profile.delete"), wsTabStop, 466, 500, 146, 34, siteIDDelete)
\tstate.connect = mk("BUTTON", parent.tr("common.connect"), wsTabStop|siteBSDefPushButton, 622, 500, 120, 34, siteIDConnect)
\tstate.close = mk("BUTTON", parent.tr("common.cancel"), wsTabStop, 752, 500, 128, 34, siteIDClose)
''',
    '''\tstate.save = parent.registerButton(mk("BUTTON", parent.tr("profile.save"), wsTabStop|bsOwnerDraw, 310, 500, 146, 34, siteIDSave), iconSave, parent.tr("profile.save"), buttonDefault)
\tstate.delete = parent.registerButton(mk("BUTTON", parent.tr("profile.delete"), wsTabStop|bsOwnerDraw, 466, 500, 146, 34, siteIDDelete), iconDelete, parent.tr("profile.delete"), buttonDanger)
\tstate.connect = parent.registerButton(mk("BUTTON", parent.tr("common.connect"), wsTabStop|siteBSDefPushButton|bsOwnerDraw, 622, 500, 120, 34, siteIDConnect), iconConnect, parent.tr("common.connect"), buttonAccent)
\tstate.close = parent.registerButton(mk("BUTTON", parent.tr("common.cancel"), wsTabStop|bsOwnerDraw, 752, 500, 128, 34, siteIDClose), iconCancel, parent.tr("common.cancel"), buttonSubtle)
''',
)
replace_once(
    "internal/desktop/site_manager_windows.go",
    '''\tsiteManagerOnce.Do(func() {
\t\tcursor, _, _ := loadCursorW.Call(0, 32512)
\t\tclass := wndClassEx{
\t\t\tCbSize:     uint32(unsafe.Sizeof(wndClassEx{})),
\t\t\tWndProc:    siteManagerProc,
\t\t\tInstance:   hinst,
\t\t\tCursor:     cursor,
\t\t\tBackground: a.brush,
\t\t\tClassName:  wstr(siteManagerClass),
\t\t}
''',
    '''\tsiteManagerOnce.Do(func() {
\t\tcursor, _, _ := loadCursorW.Call(0, 32512)
\t\ticon, _, _ := loadIconW.Call(hinst, 1)
\t\tclass := wndClassEx{
\t\t\tCbSize:     uint32(unsafe.Sizeof(wndClassEx{})),
\t\t\tWndProc:    siteManagerProc,
\t\t\tInstance:   hinst,
\t\t\tIcon:       icon,
\t\t\tCursor:     cursor,
\t\t\tBackground: a.brush,
\t\t\tClassName:  wstr(siteManagerClass),
\t\t\tIconSm:     icon,
\t\t}
''',
)

# --- Connection policy: make the real Settings timeout effective and cancellable ---
write(
    "internal/desktop/connection_policy.go",
    r'''package desktop

import (
    "time"

    "github.com/bren-wp/Ghost-FTP/internal/model"
)

func connectionTimeoutDuration(settings model.Settings) time.Duration {
    seconds := settings.ConnectionTimeoutSeconds
    if seconds < 5 {
        seconds = 15
    }
    if seconds > 60 {
        seconds = 60
    }
    return time.Duration(seconds) * time.Second
}
''',
)
write(
    "internal/desktop/connection_policy_test.go",
    r'''package desktop

import (
    "testing"
    "time"

    "github.com/bren-wp/Ghost-FTP/internal/model"
)

func TestConnectionTimeoutDurationUsesValidatedSetting(t *testing.T) {
    cases := []struct {
        seconds int
        want    time.Duration
    }{
        {0, 15 * time.Second},
        {5, 5 * time.Second},
        {27, 27 * time.Second},
        {60, 60 * time.Second},
        {999, 60 * time.Second},
    }
    for _, tc := range cases {
        got := connectionTimeoutDuration(model.Settings{ConnectionTimeoutSeconds: tc.seconds})
        if got != tc.want {
            t.Fatalf("seconds=%d got=%s want=%s", tc.seconds, got, tc.want)
        }
    }
}
''',
)
replace_once(
    "internal/desktop/connection_profiles_windows.go",
    '''\tfor _, h := range []uintptr{
\t\ta.connect, a.disconnect, a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass,
\t\ta.keyPath, a.chooseKey, a.passphrase, a.saveProfile, a.removeProfile, a.settingsBtn,
\t} {
\t\tsetControlEnabled(h, false)
\t}
\ta.setRemoteControls(false)
''',
    '''\tfor _, h := range []uintptr{
\t\ta.connect, a.profilesCombo, a.protocol, a.host, a.port, a.user, a.pass,
\t\ta.keyPath, a.chooseKey, a.passphrase, a.saveProfile, a.removeProfile, a.settingsBtn,
\t} {
\t\tsetControlEnabled(h, false)
\t}
\t// The visible Disconnect control doubles as a safe cancel action while a
\t// connection attempt is pending, so users are never trapped behind timeout.
\tsetControlEnabled(a.disconnect, true)
\ta.setRemoteControls(false)
''',
)
replace_once(
    "internal/desktop/connection_profiles_windows.go",
    '''\ta.setRemoteControls(connected)
\ta.updateActionControls()
\tinvalidateRect.Call(a.hwnd, 0, 1)
}
''',
    '''\ta.setRemoteControls(connected)
\ta.updateActionControls()
\t// Repaint only the session badge. Erasing the entire parent on every
\t// connection transition caused a visible flash on real Windows systems.
\tinvalidateRect.Call(a.connectionBadge, 0, 0)
}
''',
)
replace_all_checked(
    "internal/desktop/connection_profiles_windows.go",
    [
        ("a.connectionContext(75 * time.Second)", "a.connectionContext(connectionTimeoutDuration(a.settings))"),
    ],
)
replace_once(
    "internal/desktop/connection_profiles_windows.go",
    '''func (a *app) disconnectNow() {
\tif a.connectionBusy || !a.connected {
\t\treturn
\t}
''',
    '''func (a *app) disconnectNow() {
\tif a.connectionBusy && !a.connected {
\t\ta.beginConnectionTransition()
\t\ta.cancelConnectionAttempt()
\t\ta.setConnectionUI(false)
\t\ta.setStatus(a.tr("common.cancel"))
\t\treturn
\t}
\tif !a.connected {
\t\treturn
\t}
''',
)

# --- Windows x86 SFTP/OpenSSH resolution and neutral internal errors ---
replace_all_checked(
    "internal/remote/tools.go",
    [
        ('errors.New("Windows mrežna komponenta za FTP nije dostupna")', 'errors.New("Windows FTP transport component is not available")'),
        ('errors.New("mrežna komponenta za FTP nije pronađena")', 'errors.New("FTP transport component was not found")'),
    ],
)
replace_once(
    "internal/remote/sftp.go",
    '''func findOpenSSH(name string) (string, error) {
\tif systemDir, err := systemDirectory(); err == nil && systemDir != "" {
\t\tp := filepath.Join(systemDir, "OpenSSH", name)
\t\tif st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
\t\t\treturn p, nil
\t\t}
\t}
\tif runtime.GOOS == "windows" {
\t\treturn "", errors.New("SFTP podrška nije dostupna u sustavu Windows")
\t}
\tif strings.HasSuffix(strings.ToLower(name), ".exe") {
\t\tname = strings.TrimSuffix(name, filepath.Ext(name))
\t}
\tif p, err := exec.LookPath(name); err == nil {
\t\treturn p, nil
\t}
\treturn "", errors.New("SFTP komponenta nije pronađena")
}
''',
    '''func windowsOpenSSHCandidates(systemDir, arch, name string) []string {
\tsystemDir = filepath.Clean(strings.TrimSpace(systemDir))
\tif systemDir == "" || systemDir == "." {
\t\treturn nil
\t}
\tcandidates := []string{filepath.Join(systemDir, "OpenSSH", name)}
\tif arch == "386" && strings.EqualFold(filepath.Base(systemDir), "SysWOW64") {
\t\tcandidates = append(candidates, filepath.Join(filepath.Dir(systemDir), "Sysnative", "OpenSSH", name))
\t}
\treturn candidates
}

func findOpenSSH(name string) (string, error) {
\tif runtime.GOOS == "windows" {
\t\tif systemDir, err := systemDirectory(); err == nil && systemDir != "" {
\t\t\tif p := existingRegularFile(windowsOpenSSHCandidates(systemDir, runtime.GOARCH, name)...); p != "" {
\t\t\t\treturn p, nil
\t\t\t}
\t\t}
\t\treturn "", errors.New("SFTP support is not available on this Windows installation")
\t}
\tif systemDir, err := systemDirectory(); err == nil && systemDir != "" {
\t\tp := filepath.Join(systemDir, "OpenSSH", name)
\t\tif st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
\t\t\treturn p, nil
\t\t}
\t}
\tif strings.HasSuffix(strings.ToLower(name), ".exe") {
\t\tname = strings.TrimSuffix(name, filepath.Ext(name))
\t}
\tif p, err := exec.LookPath(name); err == nil {
\t\treturn p, nil
\t}
\treturn "", errors.New("SFTP component was not found")
}
''',
)
replace_once(
    "internal/remote/sftp.go",
    'return "", "", "", errors.New("neispravan SFTP port")',
    'return "", "", "", errors.New("invalid SFTP port")',
)
write(
    "internal/remote/sftp_tools_test.go",
    r'''package remote

import (
    "path/filepath"
    "testing"
)

func TestWindowsOpenSSHCandidatesUseNativeSystemDirectory(t *testing.T) {
    systemDir := filepath.Join("C:", "Windows", "System32")
    got := windowsOpenSSHCandidates(systemDir, "amd64", "sftp.exe")
    if len(got) != 1 || got[0] != filepath.Join(systemDir, "OpenSSH", "sftp.exe") {
        t.Fatalf("unexpected candidates: %#v", got)
    }
}

func TestWindowsOpenSSHCandidatesAddSysnativeForWOW64X86(t *testing.T) {
    windowsDir := filepath.Join("C:", "Windows")
    systemDir := filepath.Join(windowsDir, "SysWOW64")
    got := windowsOpenSSHCandidates(systemDir, "386", "ssh.exe")
    if len(got) != 2 {
        t.Fatalf("candidate count=%d want 2: %#v", len(got), got)
    }
    want := filepath.Join(windowsDir, "Sysnative", "OpenSSH", "ssh.exe")
    if got[1] != want {
        t.Fatalf("Sysnative candidate=%q want %q", got[1], want)
    }
}
''',
)

# --- Linux: localized visible workspace and no idle full-window repaint loop ---
replace_once(
    "internal/desktop/gui_linux.go",
    '"path/filepath"\n\t"strconv"',
    '"path/filepath"\n\t"reflect"\n\t"strconv"',
)
replace_once(
    "internal/desktop/gui_linux.go",
    '''\t\tremoteCurrent:    "/",
\t\tstatus:           "Ready. Enter server details or select a saved profile.",
\t\tresultCh:         make(chan linuxUIResult, 8),
\t}
\tif settings, err := engine.Settings(); err == nil {
\t\tu.language = i18n.Normalize(settings.Language)
\t}
''',
    '''\t\tremoteCurrent:    "/",
\t\tresultCh:         make(chan linuxUIResult, 8),
\t}
\tif settings, err := engine.Settings(); err == nil {
\t\tu.language = i18n.Normalize(settings.Language)
\t}
\tu.status = i18n.T(u.language, "status.ready")
''',
)
replace_all_checked(
    "internal/desktop/gui_linux.go",
    [
        ('u.x.text(premiumOuterGap, 30, "GHOST FTP", premiumTheme.Text, premiumTheme.Panel)', 'u.x.text(premiumOuterGap, 30, strings.ToUpper(brand.ProductName), premiumTheme.Text, premiumTheme.Panel)'),
        ('u.x.text(premiumOuterGap, 51, "FTP / FTPS / SFTP  |  private by design", premiumTheme.Muted, premiumTheme.Panel)', 'u.x.text(premiumOuterGap, 51, "FTP / FTPS / SFTP  |  "+u.tr("app.subtitle"), premiumTheme.Muted, premiumTheme.Panel)'),
        ('badge := "OFFLINE"', 'badge := strings.ToUpper(u.tr("badge.disconnected"))'),
        ('badge = "CONNECTED"', 'badge = strings.ToUpper(u.tr("badge.connected"))'),
        ('badge = "WORKING"', 'badge = strings.ToUpper(linuxTrimForUI(u.tr("connection.connecting", u.host), 24))'),
        ('u.drawButton(u.layout.settings, "Settings", !u.busy, false)', 'u.drawButton(u.layout.settings, u.tr("common.settings"), !u.busy, false)'),
        ('u.x.text(premiumOuterGap, 84, "QUICK CONNECT", premiumTheme.Muted, premiumTheme.Window)', 'u.x.text(premiumOuterGap, 84, strings.ToUpper(linuxTrimForUI(u.tr("profile.quick"), 28)), premiumTheme.Muted, premiumTheme.Window)'),
        ('u.drawField(linuxFieldProtocol, "Protocol")', 'u.drawField(linuxFieldProtocol, u.tr("terminal.protocol"))'),
        ('u.drawField(linuxFieldHost, "Server")', 'u.drawField(linuxFieldHost, u.tr("terminal.server"))'),
        ('u.drawField(linuxFieldPort, "Port")', 'u.drawField(linuxFieldPort, u.tr("terminal.port"))'),
        ('u.drawField(linuxFieldUser, "Username")', 'u.drawField(linuxFieldUser, u.tr("terminal.username"))'),
        ('u.drawField(linuxFieldPassword, "Password")', 'u.drawField(linuxFieldPassword, u.tr("terminal.password"))'),
        ('u.drawButton(u.layout.connect, "Connect", !u.connected && !u.busy, true)', 'u.drawButton(u.layout.connect, u.tr("common.connect"), !u.connected && !u.busy, true)'),
        ('profileLabel := "Profiles"', 'profileLabel := u.tr("profile.quick")'),
        ('u.drawButton(u.layout.saveProfile, "Save profile", u.host != "" && !u.busy, false)', 'u.drawButton(u.layout.saveProfile, u.tr("profile.save"), u.host != "" && !u.busy, false)'),
        ('u.drawButton(u.layout.removeProfile, "Remove", u.selectedProfileID != "" && !u.busy, false)', 'u.drawButton(u.layout.removeProfile, u.tr("profile.delete"), u.selectedProfileID != "" && !u.busy, false)'),
        ('u.drawField(linuxFieldKey, "SFTP private key path")', 'u.drawField(linuxFieldKey, u.tr("cue.private_key"))'),
        ('u.drawField(linuxFieldPassphrase, "Key passphrase")', 'u.drawField(linuxFieldPassphrase, u.tr("cue.passphrase"))'),
        ('u.drawButton(u.layout.disconnect, "Disconnect", u.connected && !u.busy, false)', 'u.drawButton(u.layout.disconnect, u.tr("common.disconnect"), u.connected && !u.busy, false)'),
        ('u.x.text(r.left+8, r.top+17, "NAME", premiumTheme.Muted, premiumTheme.List)', 'u.x.text(r.left+8, r.top+17, strings.ToUpper(u.tr("column.name")), premiumTheme.Muted, premiumTheme.List)'),
        ('u.x.text(r.right-112, r.top+17, "SIZE", premiumTheme.Muted, premiumTheme.List)', 'u.x.text(r.right-112, r.top+17, strings.ToUpper(u.tr("column.size")), premiumTheme.Muted, premiumTheme.List)'),
        ('u.x.text(premiumOuterGap, leftPanel.top+20, "LOCAL COMPUTER", premiumTheme.Muted, premiumTheme.Panel)', 'u.x.text(premiumOuterGap, leftPanel.top+20, strings.ToUpper(u.tr("section.local")), premiumTheme.Muted, premiumTheme.Panel)'),
        ('u.x.text(u.layout.remotePath.left, leftPanel.top+20, "SERVER", premiumTheme.Muted, premiumTheme.Panel)', 'u.x.text(u.layout.remotePath.left, leftPanel.top+20, strings.ToUpper(u.tr("section.remote")), premiumTheme.Muted, premiumTheme.Panel)'),
        ('u.drawField(linuxFieldLocalPath, "Local path")', 'u.drawField(linuxFieldLocalPath, u.tr("column.local"))'),
        ('u.drawButton(u.layout.localUp, "Up", !u.busy, false)', 'u.drawButton(u.layout.localUp, u.tr("common.up"), !u.busy, false)'),
        ('u.drawButton(u.layout.localRefresh, "Refresh", !u.busy, false)', 'u.drawButton(u.layout.localRefresh, u.tr("common.refresh"), !u.busy, false)'),
        ('u.drawField(linuxFieldRemotePath, "Remote path")', 'u.drawField(linuxFieldRemotePath, u.tr("column.remote"))'),
        ('u.drawButton(u.layout.remoteUp, "Up", u.connected && !u.busy, false)', 'u.drawButton(u.layout.remoteUp, u.tr("common.up"), u.connected && !u.busy, false)'),
        ('u.drawButton(u.layout.remoteRefresh, "Refresh", u.connected && !u.busy, false)', 'u.drawButton(u.layout.remoteRefresh, u.tr("common.refresh"), u.connected && !u.busy, false)'),
        ('u.drawButton(u.layout.localNew, "New folder", !u.busy, false)', 'u.drawButton(u.layout.localNew, u.tr("common.new_folder"), !u.busy, false)'),
        ('u.drawButton(u.layout.localRename, "Rename", u.selectedLocal >= 0 && !u.busy, false)', 'u.drawButton(u.layout.localRename, u.tr("common.rename"), u.selectedLocal >= 0 && !u.busy, false)'),
        ('u.drawButton(u.layout.localDelete, "Delete", u.selectedLocal >= 0 && !u.busy, false)', 'u.drawButton(u.layout.localDelete, u.tr("common.delete"), u.selectedLocal >= 0 && !u.busy, false)'),
        ('u.drawButton(u.layout.remoteNew, "New folder", u.connected && !u.busy, false)', 'u.drawButton(u.layout.remoteNew, u.tr("common.new_folder"), u.connected && !u.busy, false)'),
        ('u.drawButton(u.layout.remoteRename, "Rename", u.connected && u.selectedRemote >= 0 && !u.busy, false)', 'u.drawButton(u.layout.remoteRename, u.tr("common.rename"), u.connected && u.selectedRemote >= 0 && !u.busy, false)'),
        ('u.drawButton(u.layout.remoteDelete, "Delete", u.connected && u.selectedRemote >= 0 && !u.busy, false)', 'u.drawButton(u.layout.remoteDelete, u.tr("common.delete"), u.connected && u.selectedRemote >= 0 && !u.busy, false)'),
        ('u.drawButton(u.layout.remoteChmod, "Permissions", u.connected && u.selectedRemote >= 0 && !u.busy, false)', 'u.drawButton(u.layout.remoteChmod, u.tr("common.permissions"), u.connected && u.selectedRemote >= 0 && !u.busy, false)'),
        ('u.drawButton(u.layout.upload, "Upload ->", u.connected && u.selectedLocal >= 0 && !u.busy, true)', 'u.drawButton(u.layout.upload, u.tr("transfer.upload")+" →", u.connected && u.selectedLocal >= 0 && !u.busy, true)'),
        ('u.drawButton(u.layout.download, "<- Download", u.connected && u.selectedRemote >= 0 && !u.busy, true)', 'u.drawButton(u.layout.download, "← "+u.tr("transfer.download"), u.connected && u.selectedRemote >= 0 && !u.busy, true)'),
        ('u.x.text(premiumOuterGap, u.layout.pause.top+19, "TRANSFERS", premiumTheme.Muted, premiumTheme.Window)', 'u.x.text(premiumOuterGap, u.layout.pause.top+19, strings.ToUpper(u.tr("section.transfers")), premiumTheme.Muted, premiumTheme.Window)'),
        ('u.drawButton(u.layout.pause, "Pause", !u.queuePaused, false)', 'u.drawButton(u.layout.pause, u.tr("transfer.pause"), !u.queuePaused, false)'),
        ('u.drawButton(u.layout.resume, "Resume", u.queuePaused, false)', 'u.drawButton(u.layout.resume, u.tr("transfer.resume"), u.queuePaused, false)'),
        ('u.drawButton(u.layout.cancelJob, "Cancel", u.selectedTransfer >= 0, false)', 'u.drawButton(u.layout.cancelJob, u.tr("common.cancel"), u.selectedTransfer >= 0, false)'),
        ('u.drawButton(u.layout.retryJob, "Retry", u.selectedTransfer >= 0, false)', 'u.drawButton(u.layout.retryJob, u.tr("transfer.retry"), u.selectedTransfer >= 0, false)'),
        ('u.drawButton(u.layout.clearQueue, "Clear done", len(u.transferJobs) > 0, false)', 'u.drawButton(u.layout.clearQueue, u.tr("transfer.clear"), len(u.transferJobs) > 0, false)'),
        ('u.x.text(u.layout.queue.left+8, u.layout.queue.top+17, "DIRECTION   LOCAL / SERVER", premiumTheme.Muted, premiumTheme.List)', 'u.x.text(u.layout.queue.left+8, u.layout.queue.top+17, strings.ToUpper(u.tr("column.direction")+"   "+u.tr("column.local")+" / "+u.tr("column.remote")), premiumTheme.Muted, premiumTheme.List)'),
        ('u.setStatus("Refreshing local files...")', 'u.setStatus(u.tr("common.refresh")+" · "+u.tr("section.local"))'),
        ('u.setStatus("Refreshing server files...")', 'u.setStatus(u.tr("common.refresh")+" · "+u.tr("section.remote"))'),
        ('u.setStatus("Connecting securely...")', 'u.setStatus(u.tr("connection.connecting", u.host))'),
        ('context.WithTimeout(context.Background(), 75*time.Second)', 'context.WithTimeout(context.Background(), connectionTimeoutDuration(func() model.Settings { settings, _ := u.engine.Settings(); return settings }()))'),
        ('u.setStatus("Disconnecting and cancelling active transfers...")', 'u.setStatus(u.tr("disconnect.progress"))'),
        ('u.setStatus("Profile removed.")', 'u.setStatus(u.tr("profile.delete"))'),
        ('u.setStatus("Connected securely to " + u.host + ".")', 'u.setStatus(u.tr("connection.connected", u.host))'),
        ('u.setStatus("Disconnected.")', 'u.setStatus(u.tr("disconnect.done"))'),
        ('u.setStatus("SFTP host-key trust was cancelled.")', 'u.setStatus(u.tr("sftp.cancelled"))'),
        ('u.drawButton(u.layout.trust, "Trust", !u.busy, true)', 'u.drawButton(u.layout.trust, u.tr("sftp.trust"), !u.busy, true)'),
        ('u.drawButton(u.layout.cancelTrust, "Cancel", !u.busy, false)', 'u.drawButton(u.layout.cancelTrust, u.tr("common.cancel"), !u.busy, false)'),
    ],
)
# There are two host-key cancellation sites; the checked replacement above may
# have replaced both because replace_all_checked intentionally replaces all.
replace_once(
    "internal/desktop/gui_linux.go",
    '''\t\tcase <-ticker.C:
\t\t\tu.transferJobs = u.engine.Transfers()
\t\t\tif u.selectedTransfer >= len(u.transferJobs) {
\t\t\t\tu.selectedTransfer = -1
\t\t\t}
\t\t\tif err := u.renderAll(); err != nil {
\t\t\t\treturn err
\t\t\t}
''',
    '''\t\tcase <-ticker.C:
\t\t\tjobs := u.engine.Transfers()
\t\t\tif reflect.DeepEqual(jobs, u.transferJobs) {
\t\t\t\tcontinue
\t\t\t}
\t\t\tu.transferJobs = jobs
\t\t\tif u.selectedTransfer >= len(u.transferJobs) {
\t\t\t\tu.selectedTransfer = -1
\t\t\t}
\t\t\tif err := u.renderAll(); err != nil {
\t\t\t\treturn err
\t\t\t}
''',
)

replace_all_checked(
    "internal/desktop/gui_linux_actions.go",
    [
        ('u.setStatus("Operation cancelled.")', 'u.setStatus(u.tr("common.cancel"))'),
        ('u.drawButton(u.layout.promptCancel, "Cancel", !u.busy, false)', 'u.drawButton(u.layout.promptCancel, u.tr("common.cancel"), !u.busy, false)'),
        ('u.openPrompt(linuxPromptLocalMkdir, "New local folder", "New folder")', 'u.openPrompt(linuxPromptLocalMkdir, u.tr("common.new_folder")+" · "+u.tr("section.local"), u.tr("common.new_folder"))'),
        ('u.openPrompt(linuxPromptRemoteMkdir, "New server folder", "New folder")', 'u.openPrompt(linuxPromptRemoteMkdir, u.tr("common.new_folder")+" · "+u.tr("section.remote"), u.tr("common.new_folder"))'),
    ],
)

# --- Regression guards ---
write(
    "scripts/test_windows_visual_regressions.py",
    r'''import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class WindowsVisualRegressionTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_workspace_refinement_does_not_erase_entire_parent(self):
        source = self.read("internal/desktop/workspace_layout_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", source)
        self.assertIn("a.stabilizeWorkspaceChrome()", source)
        connection = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", connection)

    def test_workspace_uses_real_pe_brand_icon_and_full_title_gutter(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        self.assertIn('loadImageW.Call(hinst, 1, imageIcon', chrome)
        self.assertIn('stmSetImage', chrome)
        self.assertNotIn('a.move(a.brandTitle, 54, 10, 106, 35)', chrome)
        self.assertIn('a.move(a.brandTitle, 54, headerY, 126, 35)', ui)
        self.assertIn('subtitleX := 188', ui)

    def test_native_dark_chrome_and_site_manager_use_premium_controls(self):
        dark = self.read("internal/desktop/dark_mode_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        site = self.read("internal/desktop/site_manager_windows.go")
        self.assertIn("enableImmersiveDarkMode(hwnd)", ui)
        self.assertIn("SetPreferredAppMode", dark)
        self.assertIn('case "COMBOBOX", "EDIT":', ui)
        self.assertIn("case wmDrawItem:", site)
        self.assertIn("bsOwnerDraw", site)
        self.assertIn("IconSm:     icon", site)

    def test_resize_is_batched_without_background_erase(self):
        source = self.read("internal/desktop/layout_batch_windows.go")
        windows = self.read("internal/desktop/windows.go")
        self.assertIn("wmSetRedraw", source)
        self.assertIn("rdwAllChildren", source)
        self.assertNotIn("rdwErase", source)
        self.assertIn("a.reflowWorkspace(w, h)", windows)

    def test_disconnected_remote_list_keeps_dark_enabled_surface(self):
        source = self.read("internal/desktop/chrome_windows.go")
        self.assertIn("setControlEnabled(a.remoteList, true)", source)
        self.assertIn("styleWorkspaceList(list)", source)

    def test_startup_settings_skip_redundant_language_rebuild(self):
        source = self.read("internal/desktop/settings_windows.go")
        self.assertIn("previousLanguage := a.languageCode()", source)
        self.assertIn("if a.languageCode() != previousLanguage", source)

    def test_connect_timeout_and_cancel_are_real_ui_behaviors(self):
        source = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertIn("connectionTimeoutDuration(a.settings)", source)
        self.assertIn("a.connectionBusy && !a.connected", source)
        self.assertIn("a.cancelConnectionAttempt()", source)

    def test_x86_ftp_and_sftp_have_secure_sysnative_fallbacks(self):
        ftp = self.read("internal/remote/tools.go")
        sftp = self.read("internal/remote/sftp.go")
        self.assertIn('arch == "386"', ftp)
        self.assertIn('"Sysnative", "curl.exe"', ftp)
        self.assertNotIn('exec.LookPath("curl.exe")', ftp)
        self.assertIn("windowsOpenSSHCandidates", sftp)
        self.assertIn('"Sysnative", "OpenSSH", name', sftp)


if __name__ == "__main__":
    unittest.main()
''',
)

# --- Version 0.2.1 and detailed release documentation ---
write("VERSION", "0.2.1\n")

readme = read("README.md")
readme = readme.replace("Current Ghost FTP version: **0.2.0**", "Current Ghost FTP version: **0.2.1**", 1)
readme = re.sub(
    r"## What 0\.2\.0 changes\n.*?See \[CHANGELOG\.md\]\(CHANGELOG\.md\) for the complete version-by-version record\.",
    '''## What 0.2.1 changes

Ghost FTP 0.2.1 is a Windows/Linux polish and reliability release driven by real runtime screenshots and connection-path regression testing.

Key changes include:

- complete **Ghost FTP** branding in the Windows workspace using the canonical packaged PE icon;
- immersive dark-mode integration for native Windows menus, combo boxes, file headers and the Site Manager;
- owner-drawn Site Manager actions matching the main premium toolbar instead of mixed bright native buttons;
- batched, non-erasing Windows resize redraws plus narrower repaint regions during connection-state changes to reduce visible flicker;
- the persisted **Connection timeout** setting now controls real Windows and Linux connection attempts instead of being bypassed by a hard-coded timeout;
- a pending Windows connection can be cancelled immediately through the visible Disconnect control rather than forcing the user to wait for timeout;
- secure Windows x86/WOW64 `Sysnative` resolution for both the OS `curl.exe` FTP/FTPS transport and Windows OpenSSH/SFTP tools, without trusting user `PATH`;
- additional Linux GUI localization through the shared 24-language registry and removal of the idle 750 ms full-window repaint when transfer state has not changed;
- continued zero-telemetry, zero-external-Go-module, explicit SFTP host-key trust and fail-closed path/credential protections;
- refreshed automated visual, transport, timeout, localization and platform-parity regression coverage.

See [CHANGELOG.md](CHANGELOG.md) for the complete version-by-version record.''',
    readme,
    count=1,
    flags=re.S,
)
if "## What 0.2.1 changes" not in readme:
    raise SystemExit("README version section rewrite failed")
write("README.md", readme)

replace_once("docs/README.md", "**Current Ghost FTP release: 0.2.0**", "**Current Ghost FTP release: 0.2.1**")
replace_once(
    "docs/README.md",
    "0.1.0 Beta → 0.1.1 Beta → 0.2.0 Beta → 0.x.y Beta → 1.0.0 stable",
    "0.1.0 Beta → 0.1.1 Beta → 0.2.0 Beta → 0.2.1 Beta → 0.x.y Beta → 1.0.0 stable",
)

changelog = read("CHANGELOG.md")
entry = '''## 0.2.1 - 2026-09-06 Beta

### Windows visual quality and flicker

- Added the canonical packaged Ghost FTP icon directly to the Windows workspace and corrected header geometry so the full **Ghost FTP** name remains visible beside it.
- Enabled best-effort immersive dark rendering for modern Windows native menus/controls and aligned combo boxes and file headers with the application palette.
- Upgraded Site Manager action buttons to the same owner-drawn premium control system used by the main workspace and gave the Site Manager the packaged application icon.
- Batched resize relayout into one non-erasing redraw and removed full-parent erase operations from connection-state/DPI refresh paths that could cause visible flashing.
- Kept the disconnected SERVER pane dark and enabled as a surface while individual remote actions remain correctly disabled.

### Connection reliability and settings correctness

- Connected the persisted **Connection timeout** option to actual Windows and Linux connection attempts, with a validated 5–60 second range and a safe 15-second fallback.
- Allowed an in-progress Windows connection attempt to be cancelled immediately from the visible Disconnect control; stale async callbacks are rejected through the existing generation guard.
- Added secure WOW64/Sysnative discovery for Windows x86 OpenSSH tools so SFTP does not fail simply because a 32-bit process sees redirected System32 paths.
- Retained the secure Windows curl lookup that never trusts the user PATH, and extended the same architecture-aware principle to SFTP executables.
- Normalized low-level transport fallback errors to English, while user-facing desktop error mapping remains localized through the shared catalog.

### Linux parity, localization and efficiency

- Replaced major hard-coded Linux workspace labels with the shared 24-language catalog: connection fields, profile actions, file panes, transfer controls, queue headings, session badges and trust/cancel actions now follow the selected language.
- Made Linux use the same validated connection-timeout setting as Windows.
- Stopped redrawing the complete Linux X11 workspace every 750 ms when transfer state has not changed, reducing idle CPU/GPU/X11 work and eliminating unnecessary visual churn.

### Validation and regression coverage

- Added shared connection-timeout policy tests, Windows x86 OpenSSH/Sysnative tests and expanded Windows visual regression guards for dark chrome, full branding, resize batching and cancellable connects.
- Re-ran Go tests/race/vet, repository/platform/security/privacy/localization/documentation audits and Windows cross-build gates before release.

'''
if changelog.startswith("# Changelog\n\n"):
    changelog = "# Changelog\n\n" + entry + changelog[len("# Changelog\n\n"):]
else:
    raise SystemExit("unexpected CHANGELOG header")
write("CHANGELOG.md", changelog)

history = read("docs/RELEASE-HISTORY.md")
history_entry = '''## 0.2.1 — 2026-09-06 Beta

Theme: **runtime-verified Windows polish, effective connection settings, stronger x86 transport discovery and deeper Linux localization**.

Ghost FTP 0.2.1 follows the real Windows screenshot review performed after 0.2.0. The release fixes the clipped product title, mixed bright/dark native controls and remaining repaint paths that could produce visible flashing. Site Manager now uses the same branded icon and owner-drawn action language as the main workspace, while modern Windows receives a best-effort immersive dark-menu path without adding external UI dependencies.

Connection behavior is also tightened. The user-configured connection-timeout value now governs actual Windows and Linux connection attempts, Windows users can cancel a pending connection without waiting for timeout, and the Windows x86 build resolves OpenSSH through the secure Sysnative path when WOW64 redirects System32. FTP/FTPS and SFTP continue to reject untrusted PATH-based Windows tool discovery.

Linux receives broader 24-language coverage across the visible workspace and no longer performs a full X11 repaint every 750 ms while transfer state is unchanged. These changes preserve the same typed Engine, profile/settings model, SFTP host-key trust, transfer validation and privacy boundaries shared with Windows.

Validation for this release includes Go race tests and vet, repository/platform/security/privacy/localization/documentation audits, Windows amd64/386 cross-builds, production CI packaging and authentic Windows screenshot capture before merge.

'''
if history.startswith("# Ghost FTP release history\n\n"):
    history = "# Ghost FTP release history\n\n" + history_entry + history[len("# Ghost FTP release history\n\n"):]
else:
    raise SystemExit("unexpected release history header")
write("docs/RELEASE-HISTORY.md", history)

# Format and prove the migrated tree before any commit.
go_files = [
    "internal/desktop/chrome_windows.go",
    "internal/desktop/dark_mode_windows.go",
    "internal/desktop/layout_batch_windows.go",
    "internal/desktop/ui_windows.go",
    "internal/desktop/windows.go",
    "internal/desktop/site_manager_windows.go",
    "internal/desktop/connection_policy.go",
    "internal/desktop/connection_policy_test.go",
    "internal/desktop/connection_profiles_windows.go",
    "internal/desktop/gui_linux.go",
    "internal/desktop/gui_linux_actions.go",
    "internal/remote/tools.go",
    "internal/remote/sftp.go",
    "internal/remote/sftp_tools_test.go",
]
run("gofmt", "-w", *go_files)
run("go", "telemetry", "off")
run("go", "test", "./...")
run("go", "test", "-race", "./...")
run("go", "vet", "./...")
for audit in (
    "audit_brand_hardcut.py",
    "audit_repository.py",
    "audit_platform_contract.py",
    "audit_desktop_surface.py",
    "audit_dependencies.py",
    "audit_version.py",
    "audit_localization.py",
    "audit_security.py",
    "audit_privacy.py",
    "audit_docs.py",
    "audit_release.py",
):
    run("python", f"scripts/{audit}")
run("python", "-m", "unittest", "discover", "-s", "scripts", "-p", "test_*.py")

# Cross-build the real Windows application for both supported Windows arches.
run("go", "build", "-o", str(ROOT / ".tmp-ghostftp-amd64.exe"), "./cmd/ghostftp", env={"GOOS": "windows", "GOARCH": "amd64", "CGO_ENABLED": "0"})
run("go", "build", "-o", str(ROOT / ".tmp-ghostftp-386.exe"), "./cmd/ghostftp", env={"GOOS": "windows", "GOARCH": "386", "CGO_ENABLED": "0"})
for temp in (ROOT / ".tmp-ghostftp-amd64.exe", ROOT / ".tmp-ghostftp-386.exe"):
    temp.unlink(missing_ok=True)

run("git", "status", "--short")
run("git", "config", "user.name", "ghostftp-quality-bot")
run("git", "config", "user.email", "actions@users.noreply.github.com")
run("git", "add", "VERSION", "README.md", "CHANGELOG.md", "docs/README.md", "docs/RELEASE-HISTORY.md", "internal", "scripts/test_windows_visual_regressions.py")
# Do not commit this one-shot migration script or its workflow from inside the
# workflow; they are removed by the authorized connector after the gated commit.
run("git", "commit", "-m", "Finalize Ghost FTP 0.2.1 desktop quality")
branch = os.environ.get("GITHUB_REF_NAME", "work/windows-ui-stability-logo-20260906")
run("git", "push", "origin", f"HEAD:{branch}")
print("GHOSTFTP_0_2_1_FINALIZER=PASS")
