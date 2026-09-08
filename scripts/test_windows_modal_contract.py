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
        ):
            source = read(relative)
            self.assertNotIn("PostQuitMessage", source, relative)
            self.assertNotIn("promptPostQuitMessage", source, relative)
            self.assertIn("closed", source, relative)

    def test_custom_dialogs_use_bounded_modal_loops(self) -> None:
        prompt = read("internal/platform/prompt_windows.go")
        option = read("internal/platform/language_windows.go")
        info = read("internal/platform/info_card_windows.go")
        shell = read("internal/platform/dialog_premium_windows.go")

        self.assertIn("for !state.closed", prompt)
        self.assertIn("for !state.closed", option)
        self.assertIn("for !state.closed", info)
        self.assertIn("premiumModalOwner(owner)", prompt)
        self.assertIn("premiumModalOwner(owner)", option)
        self.assertIn("premiumModalOwner(owner)", info)
        self.assertIn("premiumDialogOuterSize", shell)
        self.assertIn("AdjustWindowRectExForDpi", shell)
        self.assertIn("GetDpiForWindow", shell)

    def test_diagnostics_uses_application_owned_theme_shell(self) -> None:
        diagnostics = read("internal/desktop/diagnostics_windows.go")
        self.assertIn("platform.CompactInfoDialog", diagnostics)
        self.assertNotIn("platform.InfoDialog(", diagnostics)

    def test_light_dialog_surface_is_softened(self) -> None:
        shell = read("internal/platform/dialog_premium_windows.go")
        self.assertIn("premiumColor(246, 248, 251)", shell)
        self.assertNotIn("return premiumColor(255, 255, 255)", shell)

    def test_compatibility_prompt_uses_configured_action_labels(self) -> None:
        prompt = read("internal/platform/prompt_windows.go")
        shell = read("internal/platform/dialog_premium_windows.go")
        self.assertIn("dialogActionLabels()", prompt)
        self.assertIn("SetDialogActionLabels", shell)


if __name__ == "__main__":
    unittest.main()
