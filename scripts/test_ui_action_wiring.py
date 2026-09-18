#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class UIActionWiringTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_every_main_windows_button_has_a_command_handler(self) -> None:
        ui = self.read("internal/desktop/ui_windows.go")
        commands = self.read("internal/desktop/commands_windows.go")

        # createControls is the authoritative main-window surface. Modal-local
        # controls have their own window procedures and are deliberately outside
        # this check.
        create_controls = ui.split("func (a *app) createControls", 1)[1].split(
            "func windowDPI", 1
        )[0]
        button_ids = set(
            re.findall(r"mkButton\([^\n]*?,\s*(id[A-Za-z0-9_]+)\)", create_controls)
        )
        self.assertGreaterEqual(len(button_ids), 20, "unexpectedly small Windows button surface")

        missing = sorted(
            button_id
            for button_id in button_ids
            if f"case {button_id}:" not in commands
        )
        self.assertEqual(
            missing,
            [],
            "main-window buttons without command handlers: " + ", ".join(missing),
        )

    def test_linux_main_controls_are_both_rendered_and_click_wired(self) -> None:
        ui = self.read("internal/desktop/gui_linux.go")
        master = self.read("internal/desktop/linux_master_rail.go")
        more = self.read("internal/desktop/linux_info_overlay.go")

        for rect in ("files", "connections", "transfers", "settings"):
            self.assertIn(f"layout.{rect} = linuxRectWH", master)
            self.assertIn(f"case rail.{rect}.contains(x, y):", master)

        for rect in (
            "back",
            "forward",
            "refresh",
            "newFolder",
            "upload",
            "download",
            "bookmarks",
            "more",
        ):
            self.assertIn(f"&layout.{rect}", master)
            self.assertIn(f"case layout.{rect}.contains(x, y):", master)

        for marker in (
            "u.navigateLinuxWorkspaceHistory(true)",
            "u.navigateLinuxWorkspaceHistory(false)",
            "u.refreshRemote(u.remoteCurrent)",
            "u.refreshLocal(u.localCurrent)",
            "linuxPromptRemoteMkdir",
            "linuxPromptLocalMkdir",
            'u.queueTransfer("upload")',
            'u.queueTransfer("download")',
            'u.openLinuxBookmarks("")',
            "u.openLinuxInfoOverlay(linuxInfoOverlayMore)",
        ):
            self.assertIn(marker, master)

        for marker in (
            "u.openFileFilterPrompt(false)",
            "u.openFileFilterPrompt(true)",
            "u.openRecursiveSearchPrompt(false)",
            "u.openRecursiveSearchPrompt(true)",
            "u.openSelectedLocalRename()",
            "u.openSelectedRemoteRename()",
            "u.deleteSelectedLocal()",
            "u.deleteSelectedRemote()",
            "u.openSelectedRemoteChmod()",
            "u.openSelectedRemoteEditor()",
            "u.startDirectoryComparisonLinux()",
        ):
            self.assertIn(marker, more)

        self.assertIn("u.handleLinuxMasterRailMouse(x, y)", ui)
        self.assertIn("u.handleLinuxMasterToolbarMouse(x, y)", ui)
        self.assertIn("u.handleLinuxFileSortHeaderMouse(x, y)", ui)
        self.assertIn("case l.clearQueue.contains(x, y):", ui)


    def test_linux_overlay_buttons_are_wired(self) -> None:
        ui = self.read("internal/desktop/gui_linux.go")
        prompts = self.read("internal/desktop/gui_linux_actions.go")

        for control in ("trust", "cancelTrust"):
            self.assertRegex(ui, rf"u\.layout\.{control}\s*=\s*linuxRectWH\(")
            self.assertIn(f"if u.layout.{control}.contains(x, y)", ui)

        for control in ("promptOK", "promptCancel"):
            self.assertRegex(prompts, rf"u\.layout\.{control}\s*=\s*linuxRectWH\(")
            self.assertIn(f"if u.layout.{control}.contains(x, y)", prompts)

    def test_queue_action_errors_are_not_silently_discarded_on_windows(self) -> None:
        transfers = self.read("internal/desktop/transfers_windows.go")
        self.assertNotIn("_ = a.engine.CancelTransfer", transfers)
        self.assertNotIn("_ = a.engine.RetryTransfer", transfers)

    def test_linux_queue_actions_share_policy_and_surface_engine_errors(self) -> None:
        ui = self.read("internal/desktop/gui_linux.go")
        actions = self.read("internal/desktop/queue_actions_linux.go")
        more = self.read("internal/desktop/linux_info_overlay.go")

        for marker in (
            "deriveTransferActionState",
            "usererror.MessageFor",
            "u.engine.PauseTransfers()",
            "u.engine.ResumeTransfers()",
            "u.engine.CancelTransfer(id)",
            "u.engine.RetryTransfer(id)",
            "u.refreshLinuxTransfersPreservingSelection(id)",
            "u.engine.ClearFinishedTransfers()",
        ):
            self.assertIn(marker, actions)

        self.assertNotIn("_ = u.engine.CancelTransfer", ui)
        self.assertNotIn("_ = u.engine.RetryTransfer", ui)

        # Clear Completed remains the one queue-level action visible in the
        # supplied master. The rest stay user-reachable through More.
        self.assertIn("actions := u.linuxTransferActionState()", ui)
        self.assertIn(
            'u.drawButton(u.layout.clearQueue, u.tr("transfer.clear"), actions.Clear && !u.busy, false)',
            ui,
        )
        self.assertIn("case l.clearQueue.contains(x, y):", ui)
        self.assertIn("u.clearFinishedTransfersLinux()", ui)

        self.assertIn("linuxInfoOverlayTransferActions", more)
        self.assertIn('"Transfer actions…"', more)
        self.assertIn("actions := u.linuxTransferActionState()", more)
        self.assertIn("priority := u.selectedQueuePriorityState()", more)
        for marker in (
            "u.pauseTransfersLinux()",
            "u.resumeTransfersLinux()",
            "u.cancelSelectedTransferLinux()",
            "u.retrySelectedTransferLinux()",
            "u.clearFinishedTransfersLinux()",
            "u.moveSelectedQueueTransfer(queuePriorityTop)",
            "u.moveSelectedQueueTransfer(queuePriorityUp)",
            "u.moveSelectedQueueTransfer(queuePriorityDown)",
            "u.moveSelectedQueueTransfer(queuePriorityBottom)",
        ):
            self.assertIn(marker, more)


    def test_linux_disabled_queue_controls_are_functionally_inert(self) -> None:
        actions = self.read("internal/desktop/queue_actions_linux.go")
        tests = self.read("internal/desktop/queue_actions_linux_test.go")
        for state in ("Pause", "Resume", "Cancel", "Retry", "Clear"):
            self.assertIn(f"!state.{state}", actions)
        self.assertGreaterEqual(
            actions.count("u.busy || !state."),
            5,
            "every Linux queue mutation helper must reject busy or disabled state",
        )
        self.assertIn(
            "TestLinuxBusyQueueMutationsAreInert",
            tests,
            "behavioral busy-state regression coverage must remain present",
        )


if __name__ == "__main__":
    unittest.main()
