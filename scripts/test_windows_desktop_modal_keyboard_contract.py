#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsDesktopModalKeyboardContractTests(unittest.TestCase):
    def test_getmessage_is_modal_aware(self) -> None:
        defs = read("internal/desktop/win32_defs_windows.go")
        helper = read("internal/desktop/modal_message_windows.go")

        self.assertIn("getMessageW             = newModalAwareGetMessageProc(user32)", defs)
        self.assertIn('user32.NewProc("GetMessageW")', helper)
        self.assertIn('user32.NewProc("IsDialogMessageW")', helper)
        self.assertIn('user32.NewProc("GetAncestor")', helper)
        self.assertIn("desktopGARoot     = 2", helper)

    def test_only_desktop_owned_modal_windows_are_intercepted(self) -> None:
        helper = read("internal/desktop/modal_message_windows.go")

        self.assertIn("siteManagerStates.Load(root)", helper)
        self.assertIn("bookmarkManagerStates.Load(root)", helper)
        self.assertNotIn("apps.Load(root)", helper)
        self.assertIn("if root == 0", helper)

    def test_native_tab_traversal_is_consumed_once(self) -> None:
        helper = read("internal/desktop/modal_message_windows.go")
        site = read("internal/desktop/site_manager_windows.go")
        bookmarks = read("internal/desktop/bookmark_manager_windows.go")

        self.assertIn("isDialogMessage.Call(root, messagePtr)", helper)
        self.assertIn("*message = msg{}", helper)
        self.assertIn("wsTabStop", site)
        self.assertIn("wsTabStop", bookmarks)
        self.assertIn("getMessageW.Call", site)
        self.assertIn("getMessageW.Call", bookmarks)

    def test_enter_and_escape_have_bounded_modal_semantics(self) -> None:
        helper = read("internal/desktop/modal_message_windows.go")

        self.assertIn("desktopVKEscape   = 0x1B", helper)
        self.assertIn("desktopVKReturn   = 0x0D", helper)
        self.assertIn("desktopBMClick    = 0x00F5", helper)
        self.assertIn("p.sendMessage.Call(root, wmClose, 0, 0)", helper)
        self.assertIn("p.sendMessage.Call(button, desktopBMClick, 0, 0)", helper)
        self.assertIn("return state.connect", helper)
        self.assertIn("return state.open", helper)
        self.assertIn("p.windowEnabled(button)", helper)

    def test_consumed_message_returns_control_to_nested_loop(self) -> None:
        helper = read("internal/desktop/modal_message_windows.go")

        self.assertIn("if p.handleModalShortcut(root, message)", helper)
        self.assertGreaterEqual(helper.count("*message = msg{}"), 2)
        self.assertIn("int32(result) <= 0", helper)
        self.assertIn("WM_QUIT/GetMessage errors", helper)


if __name__ == "__main__":
    unittest.main()
