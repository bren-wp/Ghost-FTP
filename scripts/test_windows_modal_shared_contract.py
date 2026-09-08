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
            "internal/platform/settings_dialog_windows.go",
            "internal/platform/info_card_windows.go",
        ):
            source = read(relative)
            self.assertIn("premiumRunDialogLoop(hwnd", source, relative)
            self.assertNotIn('NewProc("IsDialogMessageW")', source, relative)

    def test_info_cards_use_standard_escape_close_command(self) -> None:
        source = read("internal/platform/info_card_windows.go")
        self.assertIn("infoCardIDClose = 2 // IDCANCEL", source)
        self.assertIn("wsTabStop|bsDefPushButton", source)
        self.assertNotIn("PostQuitMessage", source)


if __name__ == "__main__":
    unittest.main()
