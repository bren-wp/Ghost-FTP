#!/usr/bin/env python3
from __future__ import annotations

import unittest

from release_notes import build_notes, extract_section


class ReleaseNotesTests(unittest.TestCase):
    def test_extracts_exact_version_section(self) -> None:
        changelog = """# Changelog

## 0.0.5 — Current

- current change

## 0.0.4 — Previous

- previous change
"""
        section = extract_section(changelog, "0.0.5")
        self.assertIn("current change", section)
        self.assertNotIn("previous change", section)

    def test_current_notes_match_windows_linux_release_contract(self) -> None:
        notes = build_notes("0.0.5", "- Production stability improvement.")
        for marker in (
            "Ghost FTP 0.0.5",
            "Privacy-first FTP, FTPS and SFTP desktop client for Windows and Linux",
            "Release channel: Current",
            "GitHub prerelease flag: false",
            "ghostftp-v0.0.5",
            "Ghost-FTP-0.0.5-Setup.exe",
            "Ghost-FTP-0.0.5-Portable.exe",
            "Ghost-FTP-0.0.5-Linux-Debian-amd64.deb",
            "Ghost-FTP-0.0.5-Linux-Debian-arm64.deb",
            "Ghost-FTP-0.0.5-Linux-Debian-i386.deb",
            "Ghost-FTP-0.0.5-Linux-Ubuntu-amd64.deb",
            "Ghost-FTP-0.0.5-Linux-Fedora-x86_64.rpm",
            "Ghost-FTP-0.0.5-Linux-Portable-amd64.tar.gz",
            "ghcr.io/bren-wp/ghost-ftp:0.0.5",
            "verified OCI distribution bundle, not a runtime container",
            "Current aliases: 0, 0.0, latest",
            "14 platform artifacts",
            "17 public release files",
            "SHA256.txt",
            "BUILD-METADATA.txt",
            "Production Authenticode signing is optional",
            "WINDOWS_AUTHENTICODE=unsigned",
            "Never treat a locally generated or self-signed certificate as a trusted public publisher identity",
            "Application telemetry: disabled",
            "latest-only retention verification",
        ):
            self.assertIn(marker, notes)

        for retired in (
            "Ghost-FTP-0.0.5-Setup-x64.exe",
            "Ghost-FTP-0.0.5-Setup-x32.exe",
            "Ghost-FTP-0.0.5-Portable-x64.exe",
            "Ghost-FTP-0.0.5-Linux-amd64.deb",
            "Ghost-FTP-0.0.5-Linux-multiarch.zip",
            "12 platform artifacts",
            "15 public release files",
            "Release channel: Beta prerelease",
            "Release channel: Stable",
            "Stable aliases",
            "macOS",
            "iOS",
            "Web.zip",
            "NuGet",
            "nuget.pkg.github.com",
        ):
            self.assertNotIn(retired, notes)

    def test_zero_major_does_not_imply_prerelease_or_hide_package(self) -> None:
        notes = build_notes("0.9.9", "- Current-line verification.")
        self.assertIn("Release channel: Current", notes)
        self.assertIn("GitHub prerelease flag: false", notes)
        self.assertIn("ghcr.io/bren-wp/ghost-ftp:0.9.9", notes)
        self.assertIn("Current aliases: 0, 0.9, latest", notes)
        self.assertNotIn("Beta prerelease", notes)


if __name__ == "__main__":
    unittest.main()
