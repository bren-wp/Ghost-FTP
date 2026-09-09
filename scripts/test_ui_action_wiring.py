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

        controls = (
            "connect",
            "disconnect",
            "settings",
            "profile",
            "saveProfile",
            "removeProfile",
            "localUp",
            "localRefresh",
            "remoteUp",
            "remoteRefresh",
            "localNew",
            "localRename",
            "localDelete",
            "remoteNew",
            "remoteRename",
            "remoteDelete",
            "remoteChmod",
            "upload",
            "download",
            "pause",
            "resume",
            "cancelJob",
            "retryJob",
            "clearQueue",
        )
        for control in controls:
            self.assertRegex(
                ui,
                rf"layout\.{re.escape(control)}\s*=\s*linuxRectWH\(",
                f"Linux control {control} is not assigned a clickable rectangle",
            )
            self.assertIn(
                f"case l.{control}.contains(x, y):",
                ui,
                f"Linux control {control} is rendered but has no click handler",
            )

        self.assertIn("case u.remoteEditButtonRect().contains(x, y):", ui)
        self.assertIn("u.openSelectedRemoteEditor()", ui)

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


if __name__ == "__main__":
    unittest.main()
