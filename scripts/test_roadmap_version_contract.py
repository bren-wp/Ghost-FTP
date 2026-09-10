#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class RoadmapVersionContractTests(unittest.TestCase):
    def test_roadmap_tracks_current_version_and_shipped_navigation_status(self) -> None:
        version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
        roadmap = (ROOT / "docs" / "ROADMAP.md").read_text(encoding="utf-8")

        self.assertRegex(version, r"^0\.0\.[1-9]\d*$")
        self.assertIn(f"Ghost FTP **{version}** is the current source/release candidate.", roadmap)
        self.assertIn(f"## Current {version} foundation", roadmap)
        self.assertIn(f"## {version} navigation work", roadmap)
        self.assertIn(f"**Status: implemented in Ghost FTP {version}.**", roadmap)
        self.assertNotIn("implemented in the maintained Unreleased source line", roadmap)
        self.assertNotIn("These capabilities remain part of the Unreleased source line", roadmap)

        current_markers = re.findall(r"Ghost FTP \*\*(\d+\.\d+\.\d+)\*\* is the current source/release candidate", roadmap)
        self.assertEqual(current_markers, [version])


if __name__ == "__main__":
    unittest.main()
