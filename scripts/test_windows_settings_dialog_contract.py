#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsSettingsDialogContractTests(unittest.TestCase):
    def test_desktop_settings_are_presented_in_one_application_owned_dialog(self) -> None:
        source = read("internal/desktop/settings_windows.go")
        self.assertIn("platform.SettingsDialog(platform.SettingsDialogConfig{", source)
        self.assertNotIn("func (a *app) promptNumber", source)
        self.assertNotIn("func (a *app) promptAppearance", source)
        self.assertNotIn("func (a *app) promptConflictPolicy", source)
        self.assertNotIn("platform.PromptDialogWithLabels(", source)
        self.assertNotIn("platform.SelectOptionDialog(", source)

    def test_unified_dialog_preserves_all_existing_settings_fields(self) -> None:
        source = read("internal/desktop/settings_windows.go")
        for marker in (
            "settings.Parallelism = result.Numbers[0]",
            "settings.ConnectionTimeoutSeconds = result.Numbers[1]",
            "settings.AutoRetryCount = result.Numbers[2]",
            "settings.RetryDelaySeconds = result.Numbers[3]",
            "applyConflictPolicySelection(&settings, result.ConflictIndex)",
            "settings.ConfirmDelete = result.ConfirmDelete",
            "applyAppearanceSelection(&settings, result.AppearanceIndex)",
        ):
            self.assertIn(marker, source)

    def test_validation_uses_canonical_config_bounds(self) -> None:
        source = read("internal/desktop/settings_windows.go")
        for marker in (
            "config.MinParallelism, config.MaxParallelism",
            "config.MinConnectionTimeoutSeconds, config.MaxConnectionTimeoutSeconds",
            "config.MinAutoRetryCount, config.MaxAutoRetryCount",
            "config.MinRetryDelaySeconds, config.MaxRetryDelaySeconds",
        ):
            self.assertIn(marker, source)

    def test_settings_modal_is_dpi_aware_owner_modal_and_never_posts_quit(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertIn("premiumDialogOuterSize", source)
        self.assertIn("premiumDialogDPI(owner)", source)
        self.assertIn("premiumModalOwner(owner)", source)
        self.assertIn("applyPremiumDialogWindow(hwnd)", source)
        self.assertIn("for !state.closed", source)
        self.assertNotIn("PostQuitMessage", source)
        self.assertNotIn("promptPostQuitMessage", source)

    def test_invalid_numeric_value_stays_inside_settings_dialog(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertIn("value < field.Min || value > field.Max", source)
        self.assertIn("settingsSetText(state.errorLabel, message)", source)
        self.assertIn("promptSetFocus.Call(edit)", source)
        self.assertIn("return 0", source)

    def test_platform_settings_dialog_does_not_import_application_model_or_config(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertNotIn("internal/model", source)
        self.assertNotIn("internal/config", source)
        self.assertNotIn("internal/i18n", source)


if __name__ == "__main__":
    unittest.main()
