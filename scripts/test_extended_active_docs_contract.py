#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]

def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")

class ExtendedActiveDocumentationContractTests(unittest.TestCase):
    def test_navigation_docs_cover_active_platforms_and_release_shape(self) -> None:
        text = read("docs/NAVIGATION-BOOKMARKS.md")
        for marker in ("Windows", "Linux", "Android", "13 platform artifacts / 16 public files"):
            self.assertIn(marker, text)
        self.assertNotIn("macOS development behavior", text)

    def test_queue_docs_cover_real_lifecycle_and_active_platforms(self) -> None:
        text = read("docs/QUEUE-PRIORITY.md")
        for marker in ("Windows", "Linux", "Android", "queued", "running", "completed", "failed", "cancelled"):
            self.assertIn(marker, text)
        self.assertIn("13 platform artifacts / 16 public files", text)
        self.assertNotIn("macOS development behavior", text)

    def test_third_party_notices_match_current_dependency_boundary(self) -> None:
        text = read("docs/THIRD-PARTY-NOTICES.md")
        for marker in (
            "Windows, Linux and Android",
            "Browser helpers",
            "Android",
            "OpenSSH",
            "signing",
        ):
            self.assertIn(marker, text)
        self.assertIn("retired", text.lower())
        self.assertNotIn("macOS is an active native development/source frontend", text)

    def test_docs_do_not_reintroduce_retired_application_tree(self) -> None:
        self.assertFalse((ROOT / "macos").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-app.yml").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-production.yml").exists())

if __name__ == "__main__":
    unittest.main()
