#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class ActiveUIDocumentationContractTests(unittest.TestCase):
    def test_settings_and_reference_ui_follow_current_version(self) -> None:
        version = read("VERSION").strip()
        for relative in ("docs/SETTINGS.md", "docs/REFERENCE-UI.md"):
            text = read(relative)
            self.assertIn("Ghost FTP", text, relative)
            self.assertIn(f"{version} Stable", text, relative)
            self.assertNotIn("1.1.1 Stable", text, relative)

    def test_reference_ui_uses_current_soft_light_palette(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        theme = read("internal/uipalette/palette.go")

        for marker in (
            "`#EEF1F5`",
            "`#F6F8FB`",
            "`#FAFBFD`",
            "`#0B0F17`",
            "`#121824`",
            "`#161D2A`",
        ):
            self.assertIn(marker, reference)

        self.assertIn("deliberately avoids pure white as the dominant application surface", reference)
        self.assertIn("Light deliberately avoids pure white as the dominant application surface", theme)
        self.assertNotIn("Panel | `255, 255, 255` (`#FFFFFF`)", reference)
        self.assertNotIn("List | `255, 255, 255` (`#FFFFFF`)", reference)

    def test_reference_ui_documents_modal_lifecycle_and_unified_settings(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        settings = read("docs/SETTINGS.md")

        self.assertIn("only the main desktop window owns process-level `WM_QUIT`/`PostQuitMessage` lifecycle", reference)
        self.assertIn("Closing **Nova mapa**, **Preimenuj**, **Postavke**, **Dijagnostika** or **O programu**", reference)
        self.assertIn("Windows Settings is one application-owned modal surface", reference)
        self.assertIn("one application-owned native Settings dialog", settings)
        self.assertIn("Invalid input keeps the dialog open", settings)

    def test_screenshot_evidence_is_not_rewritten_without_authentic_capture(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        self.assertIn("authentic UI workflow run **#122**", reference)
        self.assertIn("3b1e82695bcd3cbd87c15ed62ef20fe7c5870d4120d22dba6b44fe69141cce90", reference)
        self.assertIn("5817a0fc4012c4a4f1042c8d0cea809e6907e53680659e2d55390dd61ff88eea", reference)
        self.assertIn("historical evidence only", reference)


if __name__ == "__main__":
    unittest.main()
