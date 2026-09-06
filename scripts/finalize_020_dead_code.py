#!/usr/bin/env python3
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel: str, text: str) -> None:
    (ROOT / rel).write_text(text, encoding="utf-8", newline="\n")


def replace_once(rel: str, old: str, new: str) -> None:
    text = read(rel)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected one match in {rel}, found {count}: {old[:100]!r}")
    write(rel, text.replace(old, new, 1))


# --- Windows: physically remove the presentation-only reference shell. ---
windows = read("internal/desktop/windows.go")
old_fields = '''\tsiteManagerBtn uintptr

\t// Reference-shell controls are presentation-only aliases around the existing
\t// command and action-state layer. They never bypass Engine validation.
\tshellSidebar, shellToolbar, shellLogCard, shellQuickCard, shellLocalCard, shellRemoteCard, shellQueueCard              uintptr
\tsidebarServersLabel, sidebarPrivacyTitle, sidebarPrivacyBody, logTitle, quickTitle, localDeviceLabel, remoteStateLabel uintptr
\tremoteSearch                                                                                                           uintptr
\ttoolbarConnect, toolbarDisconnect, toolbarUpload, toolbarDownload, toolbarRefresh                                      uintptr
\ttoolbarNewFolder, toolbarRename, toolbarDelete, toolbarSites, toolbarSettings, toolbarDiagnostics                      uintptr
'''
if windows.count(old_fields) != 1:
    raise SystemExit("unexpected legacy shell field block")
windows = windows.replace(old_fields, "\tsiteManagerBtn uintptr\n", 1)
windows = windows.replace("\tremoteAllItems       []model.Item\n", "", 1)
windows = windows.replace("\tstatusLog            []string\n", "", 1)
windows = windows.replace("\t\t\tinfo.MinTrackSize.X = int32(a.scale(1080))\n\t\t\tinfo.MinTrackSize.Y = int32(a.scale(700))\n", "\t\t\tinfo.MinTrackSize.X = int32(a.scale(premiumMinWidth))\n\t\t\tinfo.MinTrackSize.Y = int32(a.scale(premiumMinHeight))\n", 1)
windows = windows.replace("\t\ta.ensureSiteManagerButton()\n", "", 1)
windows = windows.replace('''\t\tif id == idRemoteSearch && notify == enChange {
\t\t\ta.applyRemoteSearch()
\t\t\treturn 0
\t\t}
''', "", 1)
windows = windows.replace(" || lParam == a.sidebarPrivacyBody || lParam == a.localDeviceLabel || lParam == a.remoteStateLabel", "", 1)
write("internal/desktop/windows.go", windows)

# Site Manager becomes a normal canonical control, not a lazily-created alias.
ui = read("internal/desktop/ui_windows.go")
ui = ui.replace(
    '''\ta.settingsBtn = mkButton(a.tr("common.settings"), iconSettings, buttonSubtle, idSettings)
\ta.aboutBtn = mkButton(a.tr("common.about"), iconInfo, buttonSubtle, idAbout)
''',
    '''\ta.siteManagerBtn = mkButton(nativeMenuWords(a.languageCode())[5], iconOpenLocal, buttonDefault, idSiteManager)
\ta.settingsBtn = mkButton(a.tr("common.settings"), iconSettings, buttonSubtle, idSettings)
\ta.aboutBtn = mkButton(a.tr("common.about"), iconInfo, buttonSubtle, idAbout)
''',
    1,
)
ui = ui.replace("\tif width < 940 {\n\t\twidth = 940\n\t}\n\tif height < 680 {\n\t\theight = 680\n\t}\n", "\tif width < premiumMinWidth {\n\t\twidth = premiumMinWidth\n\t}\n\tif height < premiumMinHeight {\n\t\theight = premiumMinHeight\n\t}\n", 1)
ui = ui.replace("\t\ta.profilesCombo, a.languageCombo, a.saveProfile, a.removeProfile, a.settingsBtn, a.aboutBtn,\n", "\t\ta.profilesCombo, a.languageCombo, a.saveProfile, a.removeProfile, a.siteManagerBtn, a.settingsBtn, a.aboutBtn,\n", 1)
ui = ui.replace("\tif width < 900 {\n\t\twidth = 900\n\t}\n\tif height < 620 {\n\t\theight = 620\n\t}\n", "\tif width < premiumMinWidth {\n\t\twidth = premiumMinWidth\n\t}\n\tif height < premiumMinHeight {\n\t\theight = premiumMinHeight\n\t}\n", 1)
old_toolbar = '''\ttoolbarY := 51
\tavailableToolbar := width - 2*margin
\tprofileW := clampInt(availableToolbar-126-126-110-112-4*gap, 220, 360)
\tx := margin
\ta.move(a.profilesCombo, x, toolbarY, profileW, rowH)
\tx += profileW + gap
\tbuttonWidths := []int{126, 126, 110, 112}
\tif compact {
\t\tbuttonWidths = []int{118, 118, 104, 104}
\t}
\tfor index, h := range []uintptr{a.saveProfile, a.removeProfile, a.settingsBtn, a.aboutBtn} {
\t\ta.move(h, x, toolbarY, buttonWidths[index], rowH)
\t\tx += buttonWidths[index] + gap
\t}
'''
new_toolbar = '''\ttoolbarY := 51
\tavailableToolbar := width - 2*margin
\tbuttonWidths := []int{120, 120, 132, 108, 108}
\tif compact {
\t\tbuttonWidths = []int{112, 112, 120, 100, 100}
\t}
\tfixedButtons := 0
\tfor _, buttonW := range buttonWidths {
\t\tfixedButtons += buttonW
\t}
\tprofileW := clampInt(availableToolbar-fixedButtons-len(buttonWidths)*gap, 200, 340)
\tx := margin
\ta.move(a.profilesCombo, x, toolbarY, profileW, rowH)
\tx += profileW + gap
\tfor index, h := range []uintptr{a.saveProfile, a.removeProfile, a.siteManagerBtn, a.settingsBtn, a.aboutBtn} {
\t\ta.move(h, x, toolbarY, buttonWidths[index], rowH)
\t\tx += buttonWidths[index] + gap
\t}
'''
if ui.count(old_toolbar) != 1:
    raise SystemExit("canonical toolbar layout target not found")
ui = ui.replace(old_toolbar, new_toolbar, 1)
ui = ui.replace(
    '{"profiles", a.profilesCombo}, {"save profile", a.saveProfile}, {"delete profile", a.removeProfile}, {"settings", a.settingsBtn}, {"about", a.aboutBtn},',
    '{"profiles", a.profilesCombo}, {"save profile", a.saveProfile}, {"delete profile", a.removeProfile}, {"site manager", a.siteManagerBtn}, {"settings", a.settingsBtn}, {"about", a.aboutBtn},',
    1,
)
write("internal/desktop/ui_windows.go", ui)

# Live language changes update the canonical Sites button as well.
replace_once(
    "internal/desktop/localization_windows.go",
    '\ta.setButtonLabel(a.removeProfile, a.tr("profile.delete"))\n\ta.setButtonLabel(a.settingsBtn, a.tr("common.settings"))\n',
    '\ta.setButtonLabel(a.removeProfile, a.tr("profile.delete"))\n\ta.setButtonLabel(a.siteManagerBtn, nativeMenuWords(a.languageCode())[5])\n\ta.setButtonLabel(a.settingsBtn, a.tr("common.settings"))\n',
)

# Remove search-only shadow list now that the hidden remote-search alias is gone.
replace_once(
    "internal/desktop/files_actions_windows.go",
    "\t\ta.remoteAllItems = append(a.remoteAllItems[:0], items...)\n\t\ta.remoteItems = append(a.remoteItems[:0], items...)\n",
    "\t\ta.remoteItems = append(a.remoteItems[:0], items...)\n",
)

# One direct action-state graph; no duplicate toolbar authorization mirror.
write(
    "internal/desktop/action_state_windows.go",
    '''//go:build windows

package desktop

func setControlEnabled(hwnd uintptr, enabled bool) {
\tif hwnd == 0 {
\t\treturn
\t}
\tvalue := uintptr(0)
\tif enabled {
\t\tvalue = 1
\t}
\tenableWindow.Call(hwnd, value)
}

func validSelectionCount(list uintptr, itemCount int) int {
\tcount := 0
\tfor _, index := range selectedIndices(list) {
\t\tif index >= 0 && index < itemCount {
\t\t\tcount++
\t\t}
\t}
\treturn count
}

func (a *app) updateActionControls() {
\tif a == nil || a.closing {
\t\treturn
\t}

\tprofileEditable := !a.connected && !a.connectionBusy
\tsetControlEnabled(a.siteManagerBtn, profileEditable)
\tsetControlEnabled(a.saveProfile, profileEditable)
\tsetControlEnabled(a.removeProfile, profileEditable && a.selectedProfileID != "")
\tsetControlEnabled(a.settingsBtn, !a.connectionBusy)

\tlocalSelected := validSelectionCount(a.localList, len(a.localItems))
\tsetControlEnabled(a.localRename, localSelected == 1)
\tsetControlEnabled(a.localDelete, localSelected > 0)
\tsetControlEnabled(a.upload, a.connected && !a.connectionBusy && localSelected > 0)

\tremoteSelected := validSelectionCount(a.remoteList, len(a.remoteItems))
\tremoteReady := a.connected && !a.connectionBusy
\tsetControlEnabled(a.remoteMkdir, remoteReady)
\tsetControlEnabled(a.remoteRename, remoteReady && remoteSelected == 1)
\tsetControlEnabled(a.remoteDelete, remoteReady && remoteSelected > 0)
\tsetControlEnabled(a.download, remoteReady && remoteSelected > 0)

\tchmodSelected := 0
\tif remoteReady {
\t\tfor _, index := range selectedIndices(a.remoteList) {
\t\t\tif index >= 0 && index < len(a.remoteItems) && !a.remoteItems[index].IsSymlink {
\t\t\t\tchmodSelected++
\t\t\t}
\t\t}
\t}
\tsetControlEnabled(a.remoteChmod, remoteReady && chmodSelected > 0)

\ttransferState := deriveTransferActionState(a.transferJobs, selectedIndices(a.transferList), remoteReady, a.queuePaused)
\tif a.connectionBusy {
\t\ttransferState.Pause = false
\t\ttransferState.Resume = false
\t\ttransferState.Cancel = false
\t\ttransferState.Retry = false
\t}
\tsetControlEnabled(a.pauseQueue, transferState.Pause)
\tsetControlEnabled(a.resumeQueue, transferState.Resume)
\tsetControlEnabled(a.cancelJob, transferState.Cancel)
\tsetControlEnabled(a.retryJob, transferState.Retry)
\tsetControlEnabled(a.clearQueue, transferState.Clear && !a.connectionBusy)

\ta.refineWorkspaceLayout()
}
''',
)

# Command routing uses only canonical control/menu IDs.
write(
    "internal/desktop/commands_windows.go",
    '''//go:build windows

package desktop

import "path/filepath"

func (a *app) command(id int) {
\tswitch id {
\tcase idConnect:
\t\ta.connectNow()
\tcase idDisconnect:
\t\ta.disconnectNow()
\tcase idSiteManager:
\t\ta.openSiteManager()
\tcase idExitApp:
\t\tpostMessageW.Call(a.hwnd, wmClose, 0, 0)
\tcase idChooseKey:
\t\ta.choosePrivateKey()
\tcase idSaveProfile:
\t\ta.saveCurrentProfile()
\tcase idRemoveProfile:
\t\ta.removeCurrentProfile()
\tcase idSettings:
\t\ta.openSettings()
\tcase idAbout:
\t\ta.openAbout()
\tcase idDiagnostics:
\t\ta.showDiagnostics()
\tcase idLocalRefresh:
\t\ta.refreshLocal(getText(a.localPath))
\tcase idLocalChoose:
\t\ta.chooseLocalDirectory()
\tcase idRemoteRefresh:
\t\ta.refreshRemote(getText(a.remotePath))
\tcase idLocalUp:
\t\ta.refreshLocal(filepath.Dir(getText(a.localPath)))
\tcase idRemoteUp:
\t\ta.remoteUpOne()
\tcase idLocalMkdir:
\t\ta.localMkdirAction()
\tcase idLocalRename:
\t\ta.localRenameAction()
\tcase idLocalDelete:
\t\ta.localDeleteAction()
\tcase idRemoteMkdir:
\t\ta.remoteMkdirAction()
\tcase idRemoteRename:
\t\ta.remoteRenameAction()
\tcase idRemoteDelete:
\t\ta.remoteDeleteAction()
\tcase idRemoteChmod:
\t\ta.remoteChmodAction()
\tcase idUpload:
\t\ta.uploadSelected()
\tcase idDownload:
\t\ta.downloadSelected()
\tcase idPauseQueue:
\t\ta.pauseTransfers()
\tcase idResumeQueue:
\t\ta.resumeTransfers()
\tcase idCancelJob:
\t\ta.cancelSelectedTransfer()
\tcase idRetryJob:
\t\ta.retrySelectedTransfer()
\tcase idClearQueue:
\t\ta.clearFinishedTransfers()
\tcase idRefreshAll:
\t\ta.refreshLocal(getText(a.localPath))
\t\tif a.connected {
\t\t\ta.refreshRemote(getText(a.remotePath))
\t\t}
\t\ta.setStatus(a.tr("status.refresh_all"))
\t}
}
''',
)

# Diagnostics stays available from Tools without retaining the hidden shell.
replace_once(
    "internal/desktop/menu_windows.go",
    "\tidSiteManager = 701\n\tidExitApp     = 702\n",
    "\tidSiteManager = 701\n\tidExitApp     = 702\n\tidDiagnostics = 703\n",
)
replace_once("internal/desktop/menu_windows.go", "\tappendMenuItem(toolsMenu, idToolbarDiagnostics, words[8])\n", "\tappendMenuItem(toolsMenu, idDiagnostics, words[8])\n")
replace_once(
    "internal/desktop/menu_windows.go",
    "// Tools, Diagnostics. Keep the stable indices because the reference shell also\n// reuses the localized Servers and Site Manager nouns.\n",
    "// Tools, Diagnostics. Keep the stable indices because Site Manager and the\n// Tools menu use the same localized nouns across runtime language changes.\n",
)

# Compact diagnostics-only localization extracted from the removed shell.
write(
    "internal/desktop/diagnostics_windows.go",
    '''//go:build windows

package desktop

import (
\t"fmt"
\t"strings"

\t"github.com/bren-wp/Ghost-FTP/internal/platform"
)

type diagnosticsWordsSet struct {
\tPrivacyBody   string
\tRemoteOffline string
}

var diagnosticsWords = map[string]diagnosticsWordsSet{
\t"en": {"No telemetry or tracking. Saved profiles stay on this computer.", "Not connected"},
\t"hr": {"Bez telemetrije i praćenja. Spremljeni profili ostaju na ovom računalu.", "Nije povezano"},
\t"de": {"Keine Telemetrie oder Verfolgung. Gespeicherte Profile bleiben auf diesem Computer.", "Nicht verbunden"},
\t"fr": {"Aucune télémétrie ni suivi. Les profils enregistrés restent sur cet ordinateur.", "Non connecté"},
\t"es": {"Sin telemetría ni seguimiento. Los perfiles guardados permanecen en este equipo.", "Sin conexión"},
\t"tr": {"Telemetri veya izleme yok. Kaydedilen profiller bu bilgisayarda kalır.", "Bağlı değil"},
\t"el": {"Χωρίς τηλεμετρία ή παρακολούθηση. Τα αποθηκευμένα προφίλ μένουν σε αυτόν τον υπολογιστή.", "Δεν υπάρχει σύνδεση"},
\t"pt": {"Sem telemetria ou rastreio. Os perfis guardados ficam neste computador.", "Não ligado"},
\t"zh": {"无遥测或跟踪。已保存的配置保留在此计算机上。", "未连接"},
\t"ru": {"Без телеметрии и отслеживания. Сохранённые профили остаются на этом компьютере.", "Не подключено"},
\t"hi": {"कोई टेलीमेट्री या ट्रैकिंग नहीं। सहेजे गए प्रोफ़ाइल इसी कंप्यूटर पर रहते हैं।", "कनेक्ट नहीं"},
\t"ja": {"テレメトリや追跡はありません。保存したプロファイルはこのコンピューター内に残ります。", "未接続"},
\t"it": {"Nessuna telemetria o tracciamento. I profili salvati restano su questo computer.", "Non connesso"},
\t"pl": {"Brak telemetrii i śledzenia. Zapisane profile pozostają na tym komputerze.", "Brak połączenia"},
\t"nl": {"Geen telemetrie of tracking. Opgeslagen profielen blijven op deze computer.", "Niet verbonden"},
\t"cs": {"Bez telemetrie a sledování. Uložené profily zůstávají v tomto počítači.", "Nepřipojeno"},
\t"uk": {"Без телеметрії та відстеження. Збережені профілі залишаються на цьому комп’ютері.", "Не підключено"},
\t"sv": {"Ingen telemetri eller spårning. Sparade profiler stannar på den här datorn.", "Inte ansluten"},
\t"ro": {"Fără telemetrie sau urmărire. Profilurile salvate rămân pe acest computer.", "Neconectat"},
\t"hu": {"Nincs telemetria vagy követés. A mentett profilok ezen a számítógépen maradnak.", "Nincs kapcsolat"},
\t"da": {"Ingen telemetri eller sporing. Gemte profiler forbliver på denne computer.", "Ikke forbundet"},
\t"fi": {"Ei telemetriaa tai seurantaa. Tallennetut profiilit pysyvät tällä tietokoneella.", "Ei yhteyttä"},
\t"no": {"Ingen telemetri eller sporing. Lagrede profiler forblir på denne datamaskinen.", "Ikke tilkoblet"},
\t"ko": {"원격 측정이나 추적이 없습니다. 저장된 프로필은 이 컴퓨터에만 남습니다.", "연결되지 않음"},
}

func (a *app) showDiagnostics() {
\tif a == nil {
\t\treturn
\t}
\twords, ok := diagnosticsWords[a.languageCode()]
\tif !ok {
\t\twords = diagnosticsWords["en"]
\t}
\tstate := words.RemoteOffline
\tif a.connected {
\t\tstate = a.tr("badge.connected")
\t}
\ttitle := nativeMenuWords(a.languageCode())[8]
\tbody := fmt.Sprintf(
\t\t"Ghost FTP %s\\n\\n%s · %s\\n%s\\n\\n%s",
\t\ta.version, strings.ToUpper(a.protocolValue()), state, a.remoteCurrent, words.PrivacyBody,
\t)
\tplatform.InfoDialog("Ghost FTP — "+title, title, body)
}
''',
)

# One canonical workspace helper; no hidden HWNDs or reference compatibility aliases.
write(
    "internal/desktop/workspace_layout_windows.go",
    '''//go:build windows

package desktop

import "unsafe"

const (
\tworkspaceLVMGetHeader           = 0x101F
\tworkspaceLVMSetColumnOrderArray = lvmFirst + 58
\tworkspaceSWHide                 = 0
)

func styleWorkspaceList(list uintptr) {
\tif list == 0 {
\t\treturn
\t}
\tsetWindowTheme.Call(list, uintptr(unsafe.Pointer(wstr("DarkMode_Explorer"))), 0)
\tsendMessageW.Call(list, lvmSetBkColor, 0, listColor())
\tsendMessageW.Call(list, lvmSetTextBkColor, 0, listColor())
\tsendMessageW.Call(list, lvmSetTextColor, 0, textColor())
\theader, _, _ := sendMessageW.Call(list, workspaceLVMGetHeader, 0, 0)
\tif header != 0 {
\t\tsetWindowTheme.Call(header, uintptr(unsafe.Pointer(wstr("DarkMode_ItemsView"))), 0)
\t}
}

func applyFileColumnOrder(list uintptr, remote bool) {
\tif list == 0 {
\t\treturn
\t}
\tif remote {
\t\torder := [5]int32{0, 2, 1, 3, 4}
\t\tsendMessageW.Call(list, workspaceLVMSetColumnOrderArray, uintptr(len(order)), uintptr(unsafe.Pointer(&order[0])))
\t\treturn
\t}
\torder := [4]int32{0, 2, 1, 3}
\tsendMessageW.Call(list, workspaceLVMSetColumnOrderArray, uintptr(len(order)), uintptr(unsafe.Pointer(&order[0])))
}

func showControls(show bool, controls ...uintptr) {
\tstate := workspaceSWHide
\tif show {
\t\tstate = swShow
\t}
\tfor _, control := range controls {
\t\tif control != 0 {
\t\t\tshowWindow.Call(control, uintptr(state))
\t\t}
\t}
}

// refineWorkspaceLayout applies visibility/style rules to the single canonical
// native workspace. Geometry itself is owned by app.layout so resize/DPI state
// has exactly one source of truth.
func (a *app) refineWorkspaceLayout() {
\tif a == nil || a.hwnd == 0 {
\t\treturn
\t}

\tsftp := a.protocolValue() == "sftp" && !a.connected && !a.connectionBusy
\tshowControls(sftp, a.keyPath, a.chooseKey, a.passphrase)
\tif a.connected || a.connectionBusy {
\t\tshowControls(false, a.connect)
\t\tshowControls(true, a.disconnect)
\t} else {
\t\tshowControls(true, a.connect)
\t\tshowControls(false, a.disconnect)
\t}

\tfor _, list := range []uintptr{a.localList, a.remoteList, a.transferList} {
\t\tstyleWorkspaceList(list)
\t}
\tapplyFileColumnOrder(a.localList, false)
\tapplyFileColumnOrder(a.remoteList, true)
\ta.resizeListColumns()
\tinvalidateRect.Call(a.hwnd, 0, 1)
}
''',
)

# Delete the actual dead shell and its dynamic Site Manager helper.
for rel in (
    "internal/desktop/reference_shell_windows.go",
    "internal/desktop/site_toolbar_windows.go",
):
    path = ROOT / rel
    if path.exists():
        path.unlink()

# --- Tests: keep desktop/security coverage; delete tests for intentionally removed Web source. ---
write(
    "scripts/test_runtime_hardening.py",
    '''#!/usr/bin/env python3
"""Ghost FTP remote/transfer and release-runtime hardening regressions."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class RuntimeHardeningTests(unittest.TestCase):
    def test_remote_session_cannot_escape_lifecycle_guard(self) -> None:
        manager = (ROOT / "internal/remote/manager.go").read_text(encoding="utf-8")
        engine = (ROOT / "internal/api/engine.go").read_text(encoding="utf-8")
        self.assertNotIn("func (m *Manager) Session()", manager)
        self.assertNotIn("remote.Session()", engine)
        self.assertIn("func (m *Manager) Operation(ctx context.Context)", manager)
        self.assertIn("ctx = nonNilContext(ctx)", manager)

    def test_transfer_reads_use_shared_lock(self) -> None:
        source = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        self.assertIn("mu             sync.RWMutex", source)
        for signature in (
            "func (m *Manager) List() []model.TransferJob",
            "func (m *Manager) Events(since int64) ([]Event, int64)",
            "func (m *Manager) ActiveCount() int",
            "func (m *Manager) jobSnapshot(id string) (model.TransferJob, bool)",
        ):
            start = source.index(signature)
            body = source[start : start + 260]
            self.assertIn("m.mu.RLock()", body, signature)
            self.assertIn("m.mu.RUnlock()", body, signature)

    def test_transfer_selection_validation_is_centralized(self) -> None:
        source = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        self.assertEqual(source.count("func selectedIDs("), 1)
        self.assertGreaterEqual(source.count("selectedIDs(ids)"), 2)
        self.assertIn("if ctx == nil", source[source.index("func (m *Manager) waitWorkers") :])

    def test_release_workflow_requires_delayed_remote_readback(self) -> None:
        source = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
        immediate = "assert_release_asset_set immediate"
        delay = "sleep 5"
        main_guard = 'test "$main_sha" = "$GITHUB_SHA" || { echo "main moved before delayed release verification"'
        delayed = "assert_release_asset_set delayed"
        for marker in (
            "assert_release_asset_set()",
            "RELEASE_ASSET_READBACK=PASS ($phase; channel=$RELEASE_CHANNEL; windows_signing=$WINDOWS_SIGNING_STATE)",
            "gh release view \"$RELEASE_TAG\" --repo \"$repo\" --json assets",
            "gh release download \"$RELEASE_TAG\" --repo \"$repo\" --pattern 'SHA256.txt'",
            "cmp release/SHA256.txt \"$readback_dir/SHA256.txt\"",
            "remote_prerelease=\"$(gh api \"repos/$repo/releases/tags/$RELEASE_TAG\" --jq .prerelease)\"",
            immediate, delay, main_guard, delayed,
        ):
            self.assertIn(marker, source)
        self.assertLess(source.index(immediate), source.index(delay, source.index(immediate)))
        self.assertLess(source.index(delay, source.index(immediate)), source.index(main_guard))
        self.assertLess(source.index(main_guard), source.index(delayed, source.index(main_guard)))


if __name__ == "__main__":
    unittest.main()
''',
)

shared = read("scripts/test_shared_hosting_ftp.py")
start = shared.find("    def test_product_docs_link_to_detailed_shared_hosting_guide(self) -> None:\n")
if start < 0:
    raise SystemExit("stale shared-hosting doc test not found")
end = shared.find("\n\n\nif __name__ == \"__main__\":", start)
if end < 0:
    raise SystemExit("shared-hosting doc test end not found")
shared = shared[:start] + shared[end:]
write("scripts/test_shared_hosting_ftp.py", shared)

# The old stability file only contained two Web/PHP tests plus one duplicate locale check.
path = ROOT / "scripts/test_stability_hardening.py"
if path.exists():
    path.unlink()

write(
    "scripts/test_maintenance.py",
    '''#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def between(text: str, start: str, end: str) -> str:
    a = text.index(start)
    b = text.index(end, a)
    return text[a:b]


class MaintenanceRegressionTests(unittest.TestCase):
    def test_windows_remote_names_are_validated_before_use(self) -> None:
        src = read("internal/desktop/files_actions_windows.go")
        mkdir = between(src, "func (a *app) remoteMkdirAction()", "func (a *app) remoteRenameAction()")
        rename = between(src, "func (a *app) remoteRenameAction()", "func (a *app) remoteDeleteAction()")
        self.assertIn("security.ValidateRemoteName(name)", mkdir)
        self.assertIn("security.ValidateRemoteName(name)", rename)
        self.assertNotIn("strings.TrimSpace(name)", mkdir)
        self.assertNotIn("strings.TrimSpace(name)", rename)
        self.assertIn("RemoteMkdir(ctx, base, name)", mkdir)
        self.assertIn("RemoteRename(ctx, base, item.Name, name)", rename)

    def test_linux_terminal_preserves_identity_and_normalizes_optional_key_path(self) -> None:
        src = read("internal/desktop/other.go")
        prompt = between(src, "func prompt(", "func stty(")
        self.assertIn('strings.TrimRight(line, "\\r\\n")', prompt)
        self.assertNotIn("strings.TrimSpace(line)", prompt)
        self.assertIn("cfg.PrivateKeyPath = strings.TrimSpace(keyPath)", src)
        self.assertIn('if cfg.PrivateKeyPath == "" {', src)
        self.assertIn("cfg.Password = password", src)
        self.assertIn("cfg.Passphrase = passphrase", src)

    def test_retired_application_targets_and_release_surfaces_remain_absent(self) -> None:
        for rel in (
            "android", "ios", "macos", "GhostFTP WEB",
            "scripts/package_web.py", "scripts/test_package_web.py",
            "scripts/package_nuget.py", "scripts/audit_web.py",
        ):
            self.assertFalse((ROOT / rel).exists(), f"retired application/release surface exists: {rel}")

    def test_platform_contract_rejects_retired_target_reintroduction(self) -> None:
        audit = read("scripts/audit_platform_contract.py")
        self.assertIn('RETIRED_ROOTS = ("android/", "ios/", "macos/", "GhostFTP WEB/")', audit)
        self.assertIn("retired application platform/surface is tracked", audit)
        self.assertIn("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX", audit)
        self.assertIn("RETIRED_APPLICATION_PLATFORMS=ANDROID,IOS,MACOS", audit)

    def test_release_workflow_refuses_stale_main_or_tag_rewrite(self) -> None:
        workflow = read(".github/workflows/release.yml")
        self.assertIn("RELEASE_TAG=ghostftp-v$version", workflow)
        self.assertIn("main moved from release commit", workflow)
        self.assertIn("refusing to rewrite it", workflow)
        self.assertIn("gh release create", workflow)
        self.assertLess(workflow.index("main moved from release commit"), workflow.index("gh release create"))

    def test_version_history_and_current_desktop_contract(self) -> None:
        version = read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        self.assertEqual(version, "0.2.0")
        readme = read("README.md")
        changelog = read("CHANGELOG.md")
        history = read("docs/RELEASE-HISTORY.md")
        self.assertIn(f"Current Ghost FTP version: **{version}**", readme)
        self.assertIn("Development status: **Beta**", readme)
        self.assertIn("0.1.0 Beta → 0.1.1 Beta → 0.2.0 Beta", readme)
        self.assertIn("first stable", readme.lower())
        self.assertIn("**1.0.0**", readme)
        self.assertIn(f"## {version}", changelog)
        self.assertIn("## 2.0.0", changelog)
        self.assertIn("## 1.0.0", changelog)
        self.assertIn("## 2.0.0", history)
        self.assertIn("## 1.0.0", history)
        self.assertIn("ghostftp-vX.Y.Z", readme)
        sections = [m.group(1) for m in re.finditer(r"^##\\s+(\\d+\\.\\d+\\.\\d+)(?:\\s|$)", changelog, re.MULTILINE)]
        for expected in (version, "2.0.0", "1.0.0"):
            self.assertIn(expected, sections)


if __name__ == "__main__":
    unittest.main()
''',
)

ui_tests = read("scripts/test_ui_stability_hardening.py")
start = ui_tests.index("    def test_windows_layout_and_actions_follow_reference_shell_context(self) -> None:\n")
end = ui_tests.index("\n    def test_remote_permissions_column_is_backed_by_real_metadata", start)
replacement = '''    def test_windows_layout_and_actions_follow_canonical_workspace(self) -> None:
        ui = self.read("internal/desktop/ui_windows.go")
        layout = self.read("internal/desktop/workspace_layout_windows.go")
        windows = self.read("internal/desktop/windows.go")
        win32 = self.read("internal/desktop/win32_defs_windows.go")
        actions = self.read("internal/desktop/action_state_windows.go")
        commands = self.read("internal/desktop/commands_windows.go")
        menu = self.read("internal/desktop/menu_windows.go")

        for marker in (
            "preferredWindowBounds", "premiumMinWidth", "premiumMinHeight",
            "a.siteManagerBtn", "resizeListColumns", "layoutPanelWidth",
        ):
            self.assertIn(marker, ui)
        for marker in (
            "applyFileColumnOrder", "[4]int32{0, 2, 1, 3}", "[5]int32{0, 2, 1, 3, 4}",
            "showControls(sftp, a.keyPath, a.chooseKey, a.passphrase)",
        ):
            self.assertIn(marker, layout)
        for forbidden in ("shellSidebar", "toolbarConnect", "remoteSearch", "ReferenceShell"):
            self.assertNotIn(forbidden, windows + layout + actions + commands)
        self.assertFalse((ROOT / "internal/desktop/reference_shell_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/site_toolbar_windows.go").exists())
        for marker in ("wmGetMinMaxInfo", "lvnItemChanged", "updateActionControls()", "minMaxInfoFromLParam", "minMaxInfoToLParam"):
            self.assertIn(marker, windows)
        self.assertNotIn("(*minMaxInfo)(unsafe.Pointer(lParam))", windows)
        for marker in ("func minMaxInfoFromLParam", "func minMaxInfoToLParam", "rtlMoveMemory.Call"):
            self.assertIn(marker, win32)
        for marker in ("localSelected == 1", "remoteSelected == 1", "deriveTransferActionState"):
            self.assertIn(marker, actions)
        for forbidden in ("idToolbarConnect", "idToolbarUpload", "idToolbarDelete", "idRemoteSearch"):
            self.assertNotIn(forbidden, commands + menu)
        self.assertIn("idDiagnostics", commands)
        self.assertIn("idDiagnostics = 703", menu)

'''
ui_tests = ui_tests[:start] + replacement + ui_tests[end:]
start = ui_tests.index("    def test_reference_shell_remains_usable_at_authentic_capture_width(self) -> None:\n")
end = ui_tests.index("\n\n\nif __name__ == \"__main__\":", start)
replacement = '''    def test_canonical_workspace_remains_resizable_without_duplicate_shell(self) -> None:
        ui = self.read("internal/desktop/ui_windows.go")
        theme = self.read("internal/desktop/theme.go")
        windows = self.read("internal/desktop/windows.go")
        layout = self.read("internal/desktop/workspace_layout_windows.go")
        for marker in (
            "premiumStartWidth", "premiumStartHeight", "premiumMinWidth", "premiumMinHeight",
        ):
            self.assertIn(marker, theme)
        for marker in (
            "info.MinTrackSize.X = int32(a.scale(premiumMinWidth))",
            "info.MinTrackSize.Y = int32(a.scale(premiumMinHeight))",
            "case wmSize:", "case wmDpiChanged:",
        ):
            self.assertIn(marker, windows)
        for marker in (
            "compact := width < 1180", "profileW := clampInt", "localPathW", "remotePathW",
            "queueH := clampInt", "a.move(a.transferList",
        ):
            self.assertIn(marker, ui)
        self.assertNotIn("sidebarW", layout)
        self.assertNotIn("remoteSearch", layout)
'''
ui_tests = ui_tests[:start] + replacement + ui_tests[end:]
write("scripts/test_ui_stability_hardening.py", ui_tests)

# Release migration no longer owns maintenance-test conversion; this script does.
release_migration = read("scripts/finalize_020_release_surface.py")
block_start = release_migration.index("# Convert maintenance tests from the retired Web contract to the desktop-only contract.\n")
block_end = release_migration.index("# Ensure no active tooling still names the retired package scripts.\n", block_start)
release_migration = release_migration[:block_start] + release_migration[block_end:]
write("scripts/finalize_020_release_surface.py", release_migration)

# Guard against accidentally leaving presentation aliases in tracked Windows source.
for path in (ROOT / "internal/desktop").glob("*_windows.go"):
    text = path.read_text(encoding="utf-8")
    for forbidden in ("idToolbarConnect", "idToolbarUpload", "idToolbarDelete", "idRemoteSearch", "ensureReferenceShellControls"):
        if forbidden in text:
            raise SystemExit(f"legacy shell marker remains in {path.relative_to(ROOT)}: {forbidden}")

# Self-remove; the enclosing finalizer performs full audits/builds before commit.
Path(__file__).unlink()
