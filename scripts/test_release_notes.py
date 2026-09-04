#!/usr/bin/env python3
from __future__ import annotations

import unittest

from release_notes import build_notes, extract_section


class ReleaseNotesTests(unittest.TestCase):
    def test_extracts_exact_version_section(self) -> None:
        changelog = """# Changelog

## 1.4.0 — Current

- current change

## 1.3.0 — Previous

- previous change
"""
        section = extract_section(changelog, "1.4.0")
        self.assertIn("current change", section)
        self.assertNotIn("previous change", section)

    def test_public_notes_are_english_first_and_describe_release_artifacts(self) -> None:
        notes = build_notes("1.4.0", "- Native iOS release packaging.")
        for marker in (
            "Ghost FTP 1.4.0",
            "Privacy-focused FTP, FTPS and SFTP client",
            "Windows, Linux, macOS, Android, iOS and the web",
            "Highlights",
            "ghostftp-v1.4.0",
            "Public platform packages",
            "Ghost-FTP-1.4.0-Setup-x64.exe",
            "Ghost-FTP-1.4.0-Linux-multiarch.zip",
            "Ghost-FTP-1.4.0-macOS-Universal.pkg",
            "Ghost-FTP-1.4.0-Android.apk",
            "Android debug signing",
            "Ghost-FTP-1.4.0-iOS-arm64-unsigned.ipa",
            "Ghost-FTP-1.4.0-Web.zip",
            "SHA256.txt",
            "BUILD-METADATA.txt",
            "Signing and trust",
            "never fabricates publisher identities",
        ):
            self.assertIn(marker, notes)
        for retired in (
            "ByFTP 1.4.0",
            "Official desktop packages",
            "Android release-unsigned APK",
            "iOS arm64 unsigned app ZIP",
            "Najvažnije promjene",
            "Službeni paketi",
            "Provjera izdanja",
        ):
            self.assertNotIn(retired, notes)


if __name__ == "__main__":
    unittest.main()
