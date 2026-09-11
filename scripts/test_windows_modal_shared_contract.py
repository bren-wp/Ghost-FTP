#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsModalSharedContractTests(unittest.TestCase):
    def test_desktop_and_platform_read_one_palette_source(self) -> None:
        desktop = read("internal/desktop/theme.go")
        platform = read("internal/platform/dialog_premium_windows.go")
        palette = read("internal/uipalette/palette.go")

        self.assertIn('"github.com/bren-wp/Ghost-FTP/internal/uipalette"', desktop)
        self.assertIn('"github.com/bren-wp/Ghost-FTP/internal/uipalette"', platform)
        self.assertIn("var darkTheme = uipalette.Dark", desktop)
        self.assertIn("var lightTheme = uipalette.Light", desktop)
        self.assertIn("return uipalette.Dark", platform)
        self.assertIn("return uipalette.Light", platform)
        self.assertIn("var Dark = Theme{", palette)
        self.assertIn("var Light = Theme{", palette)

    def test_platform_modals_do_not_keep_the_old_dark_palette_copy(self) -> None:
        source = read("internal/platform/dialog_premium_windows.go")
        self.assertNotIn("premiumColor(15, 19, 28)", source)
        self.assertNotIn("premiumColor(244, 247, 255)", source)
        self.assertIn("uipalette.Dark.Panel", source)
        self.assertIn("uipalette.Light.Panel", source)

    def test_is_dialog_message_is_owned_by_one_shared_loop(self) -> None:
        loop = read("internal/platform/dialog_loop_windows.go")
        self.assertIn('premiumIsDialogMessageW = user32.NewProc("IsDialogMessageW")', loop)
        self.assertIn("premiumIsDialogMessageW.Call(hwnd", loop)
        self.assertIn("promptTranslateMessage.Call", loop)
        self.assertIn("promptDispatchMessageW.Call", loop)

        for relative in (
            "internal/platform/prompt_windows.go",
            "internal/platform/language_windows.go",
            "internal/platform/settings_dialog_windows.go",
            "internal/platform/info_card_windows.go",
            "internal/platform/decision_card_windows.go",
        ):
            source = read(relative)
            self.assertIn("premiumRunDialogLoop(hwnd", source, relative)
            self.assertNotIn('NewProc("IsDialogMessageW")', source, relative)

    def test_modal_owner_restores_only_unexpected_iconic_state(self) -> None:
        source = read("internal/platform/dialog_premium_windows.go")
        for marker in (
            'premiumIsIconic = user32.NewProc("IsIconic")',
            'premiumShowWindow = user32.NewProc("ShowWindow")',
            "premiumSWRestore         = 9",
            "wasIconic, _, _ := premiumIsIconic.Call(owner)",
            "if wasIconic == 0",
            "isIconic, _, _ := premiumIsIconic.Call(owner)",
            "premiumShowWindow.Call(owner, premiumSWRestore)",
            "premiumSetActiveWindow.Call(owner)",
        ):
            self.assertIn(marker, source)
        self.assertLess(
            source.index("premiumEnableWindow.Call(owner, 1)"),
            source.index("premiumShowWindow.Call(owner, premiumSWRestore)"),
        )

    def test_desktop_top_level_reenable_restores_only_unexpected_iconic_state(self) -> None:
        defs = read("internal/desktop/win32_defs_windows.go")
        helper = read("internal/desktop/modal_enable_windows.go")
        self.assertIn("enableWindow            = newModalAwareEnableWindowProc(user32)", defs)
        for marker in (
            'user32.NewProc("EnableWindow")',
            'user32.NewProc("IsIconic")',
            'user32.NewProc("GetParent")',
            'user32.NewProc("ShowWindow")',
            "const swRestore = 9",
            "trackTopLevel = parent == 0",
            "if _, exists := p.wasIconic[hwnd]; !exists",
            "if tracked && !wasIconic",
            "p.showWindow.Call(hwnd, swRestore)",
            "delete(p.wasIconic, hwnd)",
        ):
            self.assertIn(marker, helper)

        for relative in (
            "internal/desktop/site_manager_windows.go",
            "internal/desktop/bookmark_manager_windows.go",
        ):
            source = read(relative)
            self.assertIn("enableWindow.Call(a.hwnd, 0)", source, relative)
            self.assertIn("enableWindow.Call(a.hwnd, 1)", source, relative)

    def test_option_selector_uses_native_enter_and_escape_commands(self) -> None:
        source = read("internal/platform/language_windows.go")
        self.assertIn("languageIDInstall = 1 // IDOK", source)
        self.assertIn("languageIDCancel  = 2 // IDCANCEL", source)
        self.assertIn("wsTabStop|bsDefPushButton", source)
        self.assertNotIn("PostQuitMessage", source)

    def test_info_cards_use_native_enter_and_escape_commands(self) -> None:
        source = read("internal/platform/info_card_windows.go")
        self.assertIn("infoCardIDClose = 1 // IDOK", source)
        self.assertIn("id == infoCardIDClose || id == promptIDCancel", source)
        self.assertIn("wsTabStop|bsDefPushButton", source)
        self.assertNotIn("PostQuitMessage", source)

    def test_decision_cards_use_native_yes_no_and_ok_commands(self) -> None:
        source = read("internal/platform/decision_card_windows.go")
        self.assertIn("decisionIDYes", source)
        self.assertIn("decisionIDNo", source)
        self.assertIn("promptIDOK", source)
        self.assertIn("promptIDCancel", source)
        self.assertIn("resolvedDialogLabels()", source)
        self.assertNotIn("PostQuitMessage", source)


if __name__ == "__main__":
    unittest.main()
