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
            self.assertIn(version, text, relative)
            self.assertNotIn(f"{version} Beta", text, relative)
            self.assertNotIn(f"{version} Stable", text, relative)

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
        self.assertIn("independent upload and download bandwidth ceilings", reference)
        self.assertIn("`KiB/s`", reference)
        self.assertIn("one application-owned native Settings dialog", settings)
        self.assertIn("Invalid input keeps the dialog open", settings)
        self.assertIn("aggregate ceiling for that direction", settings)

    def test_screenshot_evidence_requires_exact_head_cross_platform_provenance(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        workflow = read(".github/workflows/ui-screenshots.yml")
        assembly = read("scripts/assemble_ui_evidence.py")

        for marker in (
            "Windows — 5 images",
            "Linux — 3 images",
            "Android — 7 images",
            "exactly **15 runtime images**",
            "ghostftp-authentic-ui-verified-bundle",
            "does **not** commit or push screenshots",
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

        for marker in (
            "permissions:\n  contents: read",
            "Checkout exact source",
            "Verify exact-head cross-platform evidence bundle",
            "ghostftp-authentic-ui-windows",
            "ghostftp-authentic-ui-linux",
            "ghostftp-authentic-ui-android",
            "ghostftp-authentic-ui-verified-bundle",
            "scripts/assemble_ui_evidence.py",
            'dist\\internal\\Ghost-FTP-$version-Portable-x64.exe',
        ):
            self.assertIn(marker, workflow)

        for marker in (
            '"windows/Ghost-FTP-main-workspace.png"',
            '"windows/Ghost-FTP-bookmarks.png"',
            '"linux/ghost-ftp-linux-main-workspace.png"',
            '"linux/ghost-ftp-linux-settings.png"',
            '"android/ghost-ftp-android-files.png"',
            '"android/ghost-ftp-android-navigation.png"',
            '"android/ghost-ftp-android-about.png"',
            '"capture_source_sha"',
            '"workflow_run_id"',
            '"sha256"',
            "AUTHENTIC_UI_EVIDENCE=VERIFIED",
        ):
            self.assertIn(marker, assembly)
        self.assertEqual(assembly.count('(\"windows/'), 5)
        self.assertEqual(assembly.count('(\"linux/'), 3)
        self.assertEqual(assembly.count('(\"android/'), 7)

        for forbidden in (
            "contents: write",
            "AUTHENTIC_UI_SCREENSHOTS=PERSISTED",
            "git push",
            "git commit",
            "github-actions[bot]",
        ):
            self.assertNotIn(forbidden, workflow)


if __name__ == "__main__":
    unittest.main()
