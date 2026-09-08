#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsModalContractTests(unittest.TestCase):
    def test_only_main_window_posts_quit_message(self) -> None:
        main_window = read("internal/desktop/windows.go")
        self.assertIn("case wmDestroy:", main_window)
        self.assertIn("postQuitMessage.Call(0)", main_window)

        for relative in (
            "internal/platform/prompt_windows.go",
            "internal/platform/language_windows.go",
            "internal/platform/info_card_windows.go",
            "internal/platform/settings_dialog_windows.go",
            "internal/platform/decision_card_windows.go",
        ):
            source = read(relative)
            self.assertNotIn("PostQuitMessage", source, relative)
            self.assertNotIn("promptPostQuitMessage", source, relative)
            self.assertIn("closed", source, relative)

    def test_custom_dialogs_use_one_bounded_modal_loop(self) -> None:
        loop = read("internal/platform/dialog_loop_windows.go")
        shell = read("internal/platform/dialog_premium_windows.go")

        self.assertIn("for !closed()", loop)
        self.assertIn("premiumIsDialogMessageW.Call(hwnd", loop)
        self.assertIn("promptTranslateMessage.Call", loop)
        self.assertIn("promptDispatchMessageW.Call", loop)

        for relative in (
            "internal/platform/prompt_windows.go",
            "internal/platform/language_windows.go",
            "internal/platform/info_card_windows.go",
            "internal/platform/settings_dialog_windows.go",
            "internal/platform/decision_card_windows.go",
        ):
            source = read(relative)
            self.assertIn("premiumRunDialogLoop(hwnd", source, relative)
            self.assertIn("premiumModalOwner(owner)", source, relative)

        self.assertIn("premiumDialogOuterSize", shell)
        self.assertIn("AdjustWindowRectExForDpi", shell)
        self.assertIn("GetDpiForWindow", shell)

    def test_option_dialog_uses_standard_enter_and_escape_commands(self) -> None:
        option = read("internal/platform/language_windows.go")
        self.assertIn("languageIDInstall = 1 // IDOK", option)
        self.assertIn("languageIDCancel  = 2 // IDCANCEL", option)
        self.assertIn("wsTabStop|bsDefPushButton", option)

    def test_diagnostics_uses_application_owned_theme_shell(self) -> None:
        diagnostics = read("internal/desktop/diagnostics_windows.go")
        self.assertIn("platform.CompactInfoDialog", diagnostics)
        self.assertNotIn("platform.InfoDialog(", diagnostics)

    def test_light_dialog_surface_is_softened(self) -> None:
        shell = read("internal/platform/dialog_premium_windows.go")
        palette = read("internal/uipalette/palette.go")
        self.assertIn("premiumPaletteColor(uipalette.Light.Panel)", shell)
        self.assertIn("Panel:        RGB{0xF6, 0xF8, 0xFB}", palette)
        self.assertNotIn("RGB{0xFF, 0xFF, 0xFF}", palette)

    def test_compatibility_prompt_uses_runtime_localized_action_labels(self) -> None:
        prompt = read("internal/platform/prompt_windows.go")
        shell = read("internal/platform/dialog_premium_windows.go")
        commands = read("internal/desktop/commands_windows.go")
        catalogs = read("internal/i18n/catalogs.go")

        self.assertIn("dialogActionLabels()", prompt)
        self.assertIn("SetDialogActionLabels", shell)
        self.assertIn(
            'platform.SetDialogActionLabels(okLabel(a.languageCode()), a.tr("common.cancel"))',
            commands,
        )
        self.assertIn('"common.cancel": "Otkaži"', catalogs)


if __name__ == "__main__":
    unittest.main()
