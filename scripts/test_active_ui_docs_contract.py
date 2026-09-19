#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]

def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")

class ActiveUIDocumentationContractTests(unittest.TestCase):
    def test_reference_ui_describes_active_master_surfaces(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        for marker in (
            "Windows, Linux and Android",
            "Files",
            "Connections / Sites",
            "Transfer Queue / Transfers",
            "Settings",
            "Back",
            "Forward",
            "Refresh",
            "New Folder",
            "Upload",
            "Download",
            "Bookmarks",
            "More",
            "Local Files",
            "Remote Files",
            "File",
            "Direction",
            "Progress",
            "Status",
            "Speed",
            "ETA",
            "Clear Completed",
        ):
            self.assertIn(marker, reference)

    def test_reference_ui_requires_real_runtime_evidence(self) -> None:
        reference = read("docs/REFERENCE-UI.md")
        self.assertIn("Only authentic runtime screenshots are product evidence.", reference)
        self.assertIn("Generated images and visual references are design targets, not proof of execution.", reference)

        workflow = read(".github/workflows/ui-screenshots.yml")
        for marker in (
            "permissions:\n  contents: read",
            "ghostftp-authentic-ui-windows",
            "ghostftp-authentic-ui-linux",
            "ghostftp-authentic-ui-android",
            "ghostftp-authentic-ui-verified-bundle",
            "scripts/assemble_ui_evidence.py",
        ):
            self.assertIn(marker, workflow)
        for forbidden in ("contents: write", "git push", "git commit"):
            self.assertNotIn(forbidden, workflow)

    def test_settings_documentation_is_platform_scoped(self) -> None:
        settings = read("docs/SETTINGS.md")
        for marker in ("Windows", "Linux", "Android"):
            self.assertIn(marker, settings)
        self.assertNotIn("macOS uses native AppKit", settings)

    def test_readme_runtime_evidence_is_local_and_maintained(self) -> None:
        readme = read("README.md")
        for path in (
            "docs/images/ghost-ftp-main-workspace.png",
            "docs/images/ghost-ftp-linux-main-workspace.png",
            "docs/images/ghost-ftp-android-files.png",
        ):
            self.assertIn(path, readme)
            self.assertTrue((ROOT / path).is_file(), path)

if __name__ == "__main__":
    unittest.main()
