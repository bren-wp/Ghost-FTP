#!/usr/bin/env python3
from __future__ import annotations

import unittest

from release_notes import build_notes, extract_section


class ReleaseNotesTests(unittest.TestCase):
    def test_extracts_exact_version_section(self) -> None:
        changelog = """# Changelog

## 1.1.1 — Current

- current change

## 1.1.0 — Previous

- previous change
"""
        section = extract_section(changelog, "1.1.1")
        self.assertIn("current change", section)
        self.assertNotIn("previous change", section)

    def test_public_notes_are_english_first_and_describe_android_artifacts(self) -> None:
        notes = build_notes("1.1.1", "- Android APK release packaging.")
        for marker in (
            "ByFTP 1.1.1",
            "Privacy-focused FTP / FTPS / SFTP client",
            "Highlights",
            "Official packages",
            "Android debug APK",
            "Android release-unsigned APK",
            "production store-signed build",
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
