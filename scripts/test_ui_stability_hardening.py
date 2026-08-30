#!/usr/bin/env python3
from __future__ import annotations

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

    def test_windows_layout_and_actions_follow_current_context(self) -> None:
        ui = self.read("internal/desktop/ui_windows.go")
        windows = self.read("internal/desktop/windows.go")
        win32 = self.read("internal/desktop/win32_defs_windows.go")
        actions = self.read("internal/desktop/action_state_windows.go")
        for marker in ("preferredWindowBounds", "compact := width < 1180", "resizeListColumns", "layoutPanelWidth"):
            self.assertIn(marker, ui)
        for marker in ("wmGetMinMaxInfo", "lvnItemChanged", "updateActionControls()", "minMaxInfoFromLParam", "minMaxInfoToLParam"):
            self.assertIn(marker, windows)
        self.assertNotIn("(*minMaxInfo)(unsafe.Pointer(lParam))", windows)
        for marker in ("func minMaxInfoFromLParam", "func minMaxInfoToLParam", "rtlMoveMemory.Call"):
            self.assertIn(marker, win32)
        for marker in ("localSelected == 1", "remoteSelected == 1", "deriveTransferActionState"):
            self.assertIn(marker, actions)

    def test_settings_do_not_replace_helpful_host_hint(self) -> None:
        settings = self.read("internal/desktop/settings_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        catalogs = self.read("internal/i18n/catalogs.go")
        # Settings must not replace the connection form's localized cue text.
        self.assertNotIn("cue(a.host", settings)
        self.assertIn('cue(a.host, a.tr("cue.host"))', ui)
        # The canonical English hint remains useful for shared-hosting users,
        # while other locales can provide their own equivalent cue text.
        self.assertIn('"cue.host":           "FTP/SFTP server, e.g. ftp.example.com"', catalogs)
        self.assertIn('"cue.user": "Username, may be user@example.com"', catalogs)


if __name__ == "__main__":
    unittest.main()
