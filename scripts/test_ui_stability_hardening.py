#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class UIStabilityHardeningTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_connection_callbacks_are_generation_bound(self) -> None:
        text = self.read("internal/desktop/connection_profiles_windows.go")
        for marker in (
            "connectionGeneration++",
            "beginConnectionTransition()",
            "generation != a.connectionGeneration",
            "cancelHealthCheck()",
            "finishDisconnected(",
        ):
            self.assertIn(marker, text)
        busy = text.split("func (a *app) setConnectionBusy", 1)[1].split("func (a *app) setConnectionUI", 1)[0]
        for marker in ("a.saveProfile", "a.removeProfile", "a.settingsBtn"):
            self.assertIn(marker, busy)

    def test_profile_endpoint_is_validated_before_reading_typed_secrets(self) -> None:
        text = self.read("internal/desktop/connection_profiles_windows.go")
        save = text.split("func (a *app) saveCurrentProfile()", 1)[1]
        self.assertLess(save.index("validateRawConnectionInput("), save.index("password := getText(a.pass)"))
        validator = self.read("internal/desktop/connection_input.go")
        self.assertIn("strconv.Atoi(portText)", validator)
        self.assertIn("security.ValidateConnection(protocol, host, username, port)", validator)

    def test_partial_remote_mutations_refresh_real_state(self) -> None:
        text = self.read("internal/desktop/files_actions_windows.go")
        for marker in (
            "runRemoteBatchMutationWithTimeout",
            "executeBatchMutation(ctx, count, operation)",
            "result.Succeeded > 0",
            "failedSelections",
            "skippedLinks",
        ):
            self.assertIn(marker, text)
        self.assertIn("disconnectGeneration := a.beginConnectionTransition()", text)
        self.assertIn("generation != a.connectionGeneration", text)

    def test_refreshes_preserve_selection_and_reduce_flicker(self) -> None:
        helpers = self.read("internal/desktop/helpers_windows.go")
        files = self.read("internal/desktop/files_actions_windows.go")
        transfers = self.read("internal/desktop/transfers_windows.go")
        for marker in ("selectedItemNames", "restoreItemSelection", "wmSetRedraw", "lvmSetItemState"):
            self.assertIn(marker, helpers)
        self.assertGreaterEqual(files.count("restoreItemSelection("), 2)
        for marker in ("selectedTransferIDSet", "restoreTransferSelection", 'event.Type == "state"', "event.Paused"):
            self.assertIn(marker, transfers)

    def test_windows_layout_and_actions_follow_canonical_workspace(self) -> None:
        ui = self.read("internal/desktop/ui_windows.go")
        layout = self.read("internal/desktop/workspace_layout_windows.go")
        windows = self.read("internal/desktop/windows.go")
        win32 = self.read("internal/desktop/win32_defs_windows.go")
        actions = self.read("internal/desktop/action_state_windows.go")
        commands = self.read("internal/desktop/commands_windows.go")
        sidebar = self.read("internal/desktop/sidebar_windows.go")
        navigation = self.read("internal/desktop/navigation_windows.go")

        for marker in (
            "preferredWindowBounds", "premiumMinWidth", "premiumMinHeight",
            "a.siteManagerBtn", "resizeListColumns", "layoutPanelWidth",
        ):
            self.assertIn(marker, ui)
        for marker in (
            "applyFileColumnOrder", "[4]int32{0, 2, 1, 3}", "[5]int32{0, 2, 1, 3, 4}",
            "showControls(sftp, a.keyPath, a.chooseKey, a.passphrase)",
            "a.applyApplicationSidebar()",
        ):
            self.assertIn(marker, layout)
        for forbidden in ("shellSidebar", "toolbarConnect", "remoteSearch", "ReferenceShell"):
            self.assertNotIn(forbidden, windows + layout + actions + commands)
        self.assertFalse((ROOT / "internal/desktop/reference_shell_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/site_toolbar_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/menu_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/menu_draw_windows.go").exists())
        for marker in ("wmGetMinMaxInfo", "lvnItemChanged", "updateActionControls()", "minMaxInfoFromLParam", "minMaxInfoToLParam"):
            self.assertIn(marker, windows)
        self.assertNotIn("(*minMaxInfo)(unsafe.Pointer(lParam))", windows)
        for marker in ("func minMaxInfoFromLParam", "func minMaxInfoToLParam", "rtlMoveMemory.Call"):
            self.assertIn(marker, win32)
        for marker in ("localSelected == 1", "remoteSelected == 1", "deriveTransferActionState"):
            self.assertIn(marker, actions)
        for forbidden in ("idToolbarConnect", "idToolbarUpload", "idToolbarDelete", "idRemoteSearch"):
            self.assertNotIn(forbidden, commands + sidebar + navigation)
        for command_id, value in (
            ("idFilesNav", 700),
            ("idSiteManager", 701),
            ("idTransferQueueNav", 702),
            ("idDiagnostics", 703),
        ):
            self.assertRegex(navigation, rf"\b{command_id}\s*=\s*{value}\b")
            self.assertIn(command_id, commands + sidebar)
        self.assertIn("applyApplicationSidebar", sidebar)
        self.assertIn("buttonNavActive", sidebar)
        self.assertIn("updateSidebarTransferBadge", sidebar)

    def test_remote_permissions_column_is_backed_by_real_metadata(self) -> None:
        model = self.read("internal/model/types.go")
        permissions = self.read("internal/remote/permissions.go")
        util = self.read("internal/remote/util.go")
        ftp = self.read("internal/remote/curl_ftp.go")
        ui = self.read("internal/desktop/ui_windows.go")
        localization = self.read("internal/desktop/localization_windows.go")
        layout = self.read("internal/desktop/workspace_layout_windows.go")

        self.assertIn('Permissions string    `json:"permissions,omitempty"`', model)
        for marker in ("normalizePermissionDisplay", 'strings.ContainsRune("-bcdlps"', 'strings.ContainsRune("rwxstST-"'):
            self.assertIn(marker, permissions)
        self.assertIn("Permissions: normalizePermissionDisplay(f[0])", util)
        self.assertIn('item.Permissions = normalizePermissionDisplay(facts["unix.mode"])', ftp)
        self.assertNotIn('normalizePermissionDisplay(facts["perm"])', ftp)
        self.assertIn("a.setupFileColumns(a.localList, false)", ui)
        self.assertIn("a.setupFileColumns(a.remoteList, true)", ui)
        self.assertIn('a.insertColumn(list, 4, a.tr("common.permissions"), 112)', ui)
        self.assertIn('a.setColumnTitle(a.remoteList, 4, a.tr("common.permissions"))', localization)
        self.assertIn("columns = append(columns, item.Permissions)", localization)
        self.assertIn("[5]int32{0, 2, 1, 3, 4}", layout)

    def test_settings_do_not_replace_helpful_host_hint(self) -> None:
        settings = self.read("internal/desktop/settings_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        catalogs = self.read("internal/i18n/catalogs.go")
        self.assertNotIn("cue(a.host", settings)
        self.assertIn('cue(a.host, a.tr("cue.host"))', ui)
        self.assertIn('"cue.host":           "FTP/SFTP server, e.g. ftp.example.com"', catalogs)
        self.assertIn('"cue.user": "Username, may be user@example.com"', catalogs)

    def test_windows_connection_surface_never_overwrites_locale_with_croatian_literals(self) -> None:
        profiles = self.read("internal/desktop/connection_profiles_windows.go")
        transfers = self.read("internal/desktop/transfers_windows.go")
        actions = self.read("internal/desktop/files_actions_windows.go")

        for forbidden in (
            '"Brzi spoj (bez profila)"',
            '"FTP / SFTP lozinka"',
            '"Zaporka privatnog ključa"',
            '"● POVEZANO"',
            '"● NIJE POVEZANO"',
            '"Povezivanje s "',
            '"Provjera SFTP ključa i povezivanje…"',
        ):
            self.assertNotIn(forbidden, profiles)

        for marker in (
            'a.tr("profile.quick")',
            'a.tr("cue.password")',
            'a.tr("cue.passphrase")',
            'a.tr("badge.connected")',
        ):
            self.assertIn(marker, profiles)

        for marker in ('a.tr("status.queued")', 'a.tr("transfer.pause")', 'a.tr("transfer.resume")'):
            self.assertIn(marker, transfers)
        self.assertIn('a.tr("common.cancel")', actions)


if __name__ == "__main__":
    unittest.main()
