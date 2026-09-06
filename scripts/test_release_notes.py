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

    def test_stable_notes_match_windows_linux_release_contract(self) -> None:
        notes = build_notes("1.4.0", "- Production stability improvement.")
        for marker in (
            "Ghost FTP 1.4.0",
            "Privacy-first FTP, FTPS and SFTP desktop client for Windows and Linux",
            "Release channel: Stable",
            "GitHub prerelease flag: false",
            "ghostftp-v1.4.0",
            "Ghost-FTP-1.4.0-Setup-x64.exe",
            "Ghost-FTP-1.4.0-Setup-x32.exe",
            "Ghost-FTP-1.4.0-Linux-amd64.deb",
            "Ghost-FTP-1.4.0-Linux-arm64.deb",
            "Ghost-FTP-1.4.0-Linux-i386.deb",
            "Ghost-FTP-1.4.0-Linux-multiarch.zip",
            "ghcr.io/bren-wp/ghost-ftp:1.4.0",
            "distribution bundle, not a runtime container",
            "9 platform artifacts",
            "12 public release files",
            "SHA256.txt",
            "BUILD-METADATA.txt",
            "WINDOWS_AUTHENTICODE=unsigned",
            "WINDOWS_TRUST_MODE=sha256+github-release-provenance",
            "self-signed or invented publisher identity is never substituted",
            "Application telemetry: disabled",
        ):
            self.assertIn(marker, notes)
        for retired in (
            "macOS",
            "Android",
            "iOS",
            "Web.zip",
            "NuGet",
            "nuget.pkg.github.com",
        ):
            self.assertNotIn(retired, notes)

    def test_pre_1_0_notes_are_rejected(self) -> None:
        with self.assertRaisesRegex(ValueError, "prerelease publication is disabled"):
            build_notes("0.9.9", "- Historical Beta verification.")


if __name__ == "__main__":
    unittest.main()
