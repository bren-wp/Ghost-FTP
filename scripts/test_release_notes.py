#!/usr/bin/env python3
from __future__ import annotations

import unittest

from release_notes import build_notes, extract_section


class ReleaseNotesTests(unittest.TestCase):
    def test_extracts_exact_version_section(self) -> None:
        changelog = """# Changelog

## 0.0.6 — Current

- current change

## 0.0.5 — Previous

- previous change
"""
        section = extract_section(changelog, "0.0.6")
        self.assertIn("current change", section)
        self.assertNotIn("previous change", section)

    def test_notes_match_current_cross_platform_release_contract(self) -> None:
        notes = build_notes("0.0.6", "- Production stability improvement.")
        for marker in (
            "Ghost FTP 0.0.6",
            "public Windows, Linux and Android applications",
            "Release channel: Current",
            "GitHub prerelease flag: false",
            "ghostftp-v0.0.6",
            "Ghost-FTP-0.0.6-Setup.exe",
            "Ghost-FTP-0.0.6-Portable.exe",
            "Ghost-FTP-0.0.6-Linux-Debian-Installer.run",
            "Ghost-FTP-0.0.6-Linux-Debian-Portable.tar.gz",
            "Ghost-FTP-0.0.6-Linux-Ubuntu-Installer.run",
            "Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz",
            "Ghost-FTP-0.0.6-Linux-Fedora-Installer.run",
            "Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz",
            "Ghost-FTP-0.0.6-Android.apk",
            "Ghost-FTP-0.0.6-Chrome-Extension.zip",
            "Ghost-FTP-0.0.6-Edge-Extension.zip",
            "Ghost-FTP-0.0.6-Firefox-Extension.zip",
            "Ghost-FTP-0.0.6-Opera-Extension.zip",
            "ghcr.io/bren-wp/ghost-ftp:0.0.6",
            "verified OCI distribution bundle, not a runtime container",
            "Current aliases: 0, 0.0, latest",
            "13 platform artifacts",
            "16 public release files",
            "SHA256.txt",
            "BUILD-METADATA.txt",
            "Official Windows publication is signed-only",
            "protected production publisher identity",
            "SFTP remains hidden",
            "Application telemetry: disabled",
            "latest-only retention verification",
        ):
            self.assertIn(marker, notes)

        for retired in (
            "Ghost-FTP-0.0.6-Setup-x64.exe",
            "Ghost-FTP-0.0.6-Setup-x32.exe",
            "Ghost-FTP-0.0.6-Portable-x64.exe",
            "Ghost-FTP-0.0.6-Linux-Debian-amd64.deb",
            "Ghost-FTP-0.0.6-Linux-Fedora-x86_64.rpm",
            "Ghost-FTP-0.0.6-Linux-Portable-amd64.tar.gz",
            "18 platform artifacts",
            "21 public release files",
            "12 platform artifacts",
            "15 public release files",
            "Production Authenticode signing is optional",
            "WINDOWS_AUTHENTICODE=unsigned",
            "Release channel: Beta prerelease",
            "Release channel: Stable",
            "Stable aliases",
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
