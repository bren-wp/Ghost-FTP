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
        self.assertIn("premiumRunDialogLoop(hwnd", source)
        self.assertNotIn("PostQuitMessage", source)
        self.assertNotIn("promptPostQuitMessage", source)

    def test_invalid_numeric_value_stays_inside_settings_dialog(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertIn("value < field.Min || value > field.Max", source)
        self.assertIn("settingsSetText(state.errorLabel, message)", source)
        self.assertIn("promptSetFocus.Call(edit)", source)
        self.assertIn("return 0", source)

    def test_keyboard_navigation_uses_shared_win32_dialog_manager_and_standard_commands(self) -> None:
        settings = read("internal/platform/settings_dialog_windows.go")
        loop = read("internal/platform/dialog_loop_windows.go")
        self.assertIn("settingsIDApply      = 1 // IDOK", settings)
        self.assertIn("settingsIDCancel     = 2 // IDCANCEL", settings)
        self.assertIn("premiumRunDialogLoop(hwnd", settings)
        self.assertNotIn("settingsIsDialogMessageW", settings)
        self.assertIn('premiumIsDialogMessageW = user32.NewProc("IsDialogMessageW")', loop)
        self.assertIn("premiumIsDialogMessageW.Call(hwnd", loop)
        self.assertIn("if handled != 0", loop)
        self.assertIn("continue", loop)

    def test_numeric_labels_reserve_two_lines_for_long_locales(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertIn("numberLabelH    = 42", source)
        self.assertIn("numberRowHeight = 82", source)
        self.assertIn("fieldWidth, numberLabelH", source)
        self.assertIn("row*numberRowHeight", source)
        self.assertNotIn("fieldWidth, 24, 0, captionFont", source)

    def test_lower_settings_regions_expand_with_numeric_rows(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertIn("len(config.Numbers) > 6", source)
        self.assertIn("numberRows := (len(config.Numbers) + 1) / 2", source)
        self.assertIn("separatorY := numberStartY + numberRows*numberRowHeight + 4", source)
        self.assertIn("footerSeparatorY := confirmY + 52", source)
        self.assertIn("clientHeight := footerSeparatorY + 100", source)
        self.assertIn('36, footerSeparatorY, 688, 2', source)
        self.assertIn('36, footerY, 470, 38', source)
        self.assertIn('36, errorY, 470, 24', source)
        self.assertIn('516, buttonY, 98, 38', source)
        self.assertIn('624, buttonY, 100, 38', source)
        self.assertNotIn('36, 500, 688, 2', source)
        self.assertNotIn('36, 514, 470, 38', source)
        self.assertNotIn('516, 530, 98, 38', source)
        self.assertNotIn('36, 552, 470, 24', source)

    def test_platform_settings_dialog_does_not_import_application_model_or_config(self) -> None:
        source = read("internal/platform/settings_dialog_windows.go")
        self.assertNotIn("internal/model", source)
        self.assertNotIn("internal/config", source)
        self.assertNotIn("internal/i18n", source)


if __name__ == "__main__":
    unittest.main()
