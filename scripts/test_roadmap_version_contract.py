#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

class RoadmapVersionContractTests(unittest.TestCase):
    def test_roadmap_tracks_active_platforms_and_release_discipline(self) -> None:
        version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
        roadmap = (ROOT / "docs" / "ROADMAP.md").read_text(encoding="utf-8")

        self.assertEqual(version, "0.0.8")
        for marker in (
            "Windows, Linux and Android",
            "### Windows",
            "### Linux",
            "### Android",
            "macOS is removed from the active roadmap",
            "Published releases remain immutable.",
            "fresh exact-source artifacts/checksums",
        ):
            self.assertIn(marker, roadmap)

        self.assertNotIn("macOS remains a separately validated", roadmap)
        self.assertNotIn("Current source/release candidate", roadmap)

if __name__ == "__main__":
    unittest.main()
