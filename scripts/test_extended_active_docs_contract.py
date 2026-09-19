#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]

def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")

class ExtendedActiveDocumentationContractTests(unittest.TestCase):
    def test_navigation_docs_cover_active_platforms(self):
        text = read("docs/NAVIGATION-BOOKMARKS.md")
        for marker in ("Windows", "Linux", "Android"):
            self.assertIn(marker, text)

    def test_queue_docs_cover_active_platforms(self):
        text = read("docs/QUEUE-PRIORITY.md")
        for marker in ("Windows", "Linux", "Android"):
            self.assertIn(marker, text)

    def test_third_party_notices_use_current_platform_boundary(self):
        text = read("docs/THIRD-PARTY-NOTICES.md")
        self.assertIn("Windows, Linux and Android", text)
        self.assertNotIn("macOS production distribution", text)

if __name__ == "__main__":
    unittest.main()
