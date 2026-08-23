#!/usr/bin/env python3
from __future__ import annotations

import unittest

from release_notes import build_notes, extract_section


class ReleaseNotesTests(unittest.TestCase):
    def test_extracts_exact_version_section(self) -> None:
        changelog = """# Changelog

## 1.2.0 — Current

- current change

## 1.1.1 — Previous

- previous change
"""
        section = extract_section(changelog, "1.2.0")
        self.assertIn("current change", section)
        self.assertNotIn("previous change", section)

    def test_public_notes_are_english_first_and_describe_mobile_artifacts(self) -> None:
        notes = build_notes("1.2.0", "- Native iOS release packaging.")
        for marker in (
            "ByFTP 1.2.0",
            "Privacy-focused file-transfer client",
            "Windows, Linux, macOS, Android and iOS",
            "Highlights",
            "Official packages",
            "Android debug APK",
            "Android release-unsigned APK",
            "iOS arm64 unsigned IPA",
            "iOS arm64 unsigned app ZIP",
            "valid Apple signing identity",
            "Release verification",
            "Before installing",
            "Signing status",
        ):
            self.assertIn(marker, notes)
        for retired in (
            "Official desktop packages",
            "source is under android/ and is a required release-quality gate",
            "Najvažnije promjene",
            "Službeni paketi",
            "Provjera izdanja",
            "Preporuka prije instalacije",
        ):
            self.assertNotIn(retired, notes)


if __name__ == "__main__":
    unittest.main()
