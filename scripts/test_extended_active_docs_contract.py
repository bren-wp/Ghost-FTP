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
            self.assertNotIn("12 platform artifacts / 15 public files", text, relative)
            self.assertNotIn("9 platform artifacts / 12 public files", text, relative)
            self.assertNotIn("6 platform artifacts / 9 public files", text, relative)
            self.assertNotIn("production Authenticode is optional", text, relative)
            self.assertNotIn("GhostFTP WEB/", text, relative)

    def test_navigation_docs_preserve_public_and_macos_boundaries(self) -> None:
        text = read("docs/NAVIGATION-BOOKMARKS.md")
        for marker in (
            "## macOS development behavior",
            "same shared bookmark Engine APIs",
            "source/development parity",
            "Windows/Linux remain the public release surfaces",
            "macOS remains a separately validated native development/source frontend",
        ):
            self.assertIn(marker, text)

    def test_queue_docs_preserve_shared_engine_and_release_boundaries(self) -> None:
        text = read("docs/QUEUE-PRIORITY.md")
        for marker in (
            "## macOS development behavior",
            "same typed `internal/api.Engine` operations",
            "does not create a Mac-only scheduler or protocol stack",
            "14 platform artifacts / 17 public files",
            "macOS remains a separately validated native development/source frontend",
        ):
            self.assertIn(marker, text)

    def test_third_party_notices_use_current_dependency_and_signing_truth(self) -> None:
        text = read("docs/THIRD-PARTY-NOTICES.md")
        for marker in (
            "current public release platforms are Windows and Linux",
            "macOS is an active native development/source frontend",
            "## Android development dependency boundary",
            "## Browser connection helper",
            "does not provide a supported browser-to-desktop URI/native-messaging handoff today",
            "Official public Windows publication requires the protected trusted production Authenticode identity",
            "WINDOWS_AUTHENTICODE=signed",
            "There is no supported unsigned continuation",
            "macOS production distribution has a separate fail-closed Developer ID + notarization boundary",
        ):
            self.assertIn(marker, text)


if __name__ == "__main__":
    unittest.main()
