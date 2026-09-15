#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class ExtendedActiveDocumentationContractTests(unittest.TestCase):
    def test_feature_and_notice_docs_follow_current_version(self) -> None:
        version = read("VERSION").strip()
        for relative in (
            "docs/NAVIGATION-BOOKMARKS.md",
            "docs/QUEUE-PRIORITY.md",
            "docs/THIRD-PARTY-NOTICES.md",
        ):
            text = read(relative)
            self.assertIn(f"Ghost FTP **{version}**", text, relative)
            self.assertIn("16", text, relative)
            for stale in (
                "18 platform artifacts / 21 public files",
                "21-file release",
                "12 platform artifacts / 15 public files",
                "9 platform artifacts / 12 public files",
                "6 platform artifacts / 9 public files",
            ):
                self.assertNotIn(stale, text, relative)

    def test_navigation_docs_preserve_security_and_macos_boundaries(self) -> None:
        text = read("docs/NAVIGATION-BOOKMARKS.md")
        for marker in (
            "## macOS development behavior",
            "source/development parity",
            "macOS remains a separately validated native development/source frontend",
            "13 platform artifacts / 16 public files",
        ):
            self.assertIn(marker, text)
        self.assertIn("Android", text)

    def test_queue_docs_preserve_shared_engine_and_release_boundaries(self) -> None:
        text = read("docs/QUEUE-PRIORITY.md")
        for marker in (
            "## macOS development behavior",
            "same typed `internal/api.Engine` operations",
            "does not create a Mac-only scheduler or protocol stack",
            "13 platform artifacts / 16 public files",
            "macOS remains a separately validated native development/source frontend",
        ):
            self.assertIn(marker, text)

    def test_third_party_notices_use_current_dependency_and_signing_truth(self) -> None:
        text = read("docs/THIRD-PARTY-NOTICES.md")
        for marker in (
            "Windows, Linux and Android",
            "Chrome, Edge, Firefox and Opera",
            "## Android public dependency boundary",
            "## Browser extensions",
            "protected trusted production Authenticode identity",
            "WINDOWS_AUTHENTICODE=signed",
            "Developer ID Application",
            "Apple notarization",
        ):
            self.assertIn(marker, text)
        self.assertNotIn("21-file", text)


if __name__ == "__main__":
    unittest.main()
