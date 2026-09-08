#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsPickerLocalizationContractTests(unittest.TestCase):
    def test_native_pickers_use_runtime_label_provider(self) -> None:
        platform = read("internal/platform/windows.go")
        provider = read("internal/platform/picker_label_provider_windows.go")
        desktop = read("internal/desktop/dialog_localization_windows.go")

        self.assertIn("resolvedPickerLabels()", platform)
        self.assertIn("SetPickerLabelProvider", provider)
        self.assertIn("resolvedPickerLabels", provider)
        self.assertIn("platform.SetPickerLabelProvider", desktop)
        self.assertIn("activeDialogLocale()", desktop)
        self.assertIn("privateKeyDialogTitle(language)", desktop)
        self.assertIn("privateKeyFilterLabel(language)", desktop)
        self.assertIn("allFilesFilterLabel(language)", desktop)
        self.assertIn("directoryDialogTitle(language)", desktop)

    def test_picker_platform_has_english_fallback_not_hardcoded_croatian_ui(self) -> None:
        platform = read("internal/platform/windows.go")
        provider = read("internal/platform/picker_label_provider_windows.go")

        for stale in (
            "Odaberi SSH privatni ključ",
            "SSH privatni ključevi|id_*;*.pem;*.key|Sve datoteke|*.*",
            "Odaberi lokalnu mapu za GhostFTP",
        ):
            self.assertNotIn(stale, platform)

        for fallback in (
            "Select SSH private key",
            "SSH private keys",
            "All files",
            "Select local folder for Ghost FTP",
        ):
            self.assertIn(fallback, provider)

    def test_decision_card_layout_is_content_adaptive(self) -> None:
        card = read("internal/platform/decision_card_windows.go")
        self.assertIn("decisionCardLayoutForText", card)
        self.assertIn("decisionCardEstimatedLines", card)
        self.assertIn("decisionSSEditControl", card)
        self.assertIn("layout.clientHeight", card)
        self.assertIn("layout.headingHeight", card)
        self.assertIn("layout.bodyHeight", card)
        self.assertNotIn("clientHeight = 320", card)
        self.assertNotIn("608, 126, 0, bodyFont", card)


if __name__ == "__main__":
    unittest.main()
