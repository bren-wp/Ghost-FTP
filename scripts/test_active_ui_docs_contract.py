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

    def test_screenshot_evidence_requires_complete_authentic_capture_provenance(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        workflow = read(".github/workflows/ui-screenshots.yml")

        for marker in (
            "authentic UI workflow run **#201**",
            "a1e9635f5724ea8b53afca9830f28f7fa9159798",
            "29f3a9a069df37107772265987ecfd251b645c3e",
            "659caccf3fab3e9add23424e45709f541c902219ba5212f6287ab5ed25c8cb6e",
            "63e9292530030afab7a95b21788d9ec1da80b6f7bca1ba0ce8a132fd665602a9",
            "b46b8c9c0730e96b1a0ed9ba54e84633eb6f2030271407f0944d433046c0c870",
            "1d1b6487be473e3f59af2620584cf09e2ef3225d30712818a9cb8ca1fa492ca4",
            "Mockups, image-generation output and manually composed approximations are not accepted",
        ):
            self.assertIn(marker, reference)

        for image in (
            "ghost-ftp-main-workspace.png",
            "ghost-ftp-site-manager.png",
            "ghost-ftp-settings.png",
            "ghost-ftp-about.png",
        ):
            self.assertIn(image, reference)
            self.assertIn(image, workflow)

        self.assertIn("Verify screenshot outputs", workflow)
        self.assertIn("AUTHENTIC_UI_SCREENSHOTS=PERSISTED", workflow)


if __name__ == "__main__":
    unittest.main()
