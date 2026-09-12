#!/usr/bin/env python3
"""Protect the current public release-notes contract from stale artifact names/counts."""

from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MODULE_PATH = ROOT / "scripts" / "release_notes.py"

spec = importlib.util.spec_from_file_location("ghostftp_release_notes", MODULE_PATH)
assert spec and spec.loader
release_notes = importlib.util.module_from_spec(spec)
spec.loader.exec_module(release_notes)


class ReleaseNotesCurrentContractTests(unittest.TestCase):
    def test_release_notes_match_current_public_artifact_contract(self) -> None:
        version = "9.8.7"
        notes = release_notes.build_notes(version, "- Example release highlight.")

        required = (
            f"Ghost FTP {version}",
            f"ghostftp-v{version}",
            f"Ghost-FTP-{version}-Setup.exe",
            f"Ghost-FTP-{version}-Portable.exe",
            f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
            f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
            "14 platform artifacts",
            "17 public release files",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "prerelease flag: false",
        )
        for marker in required:
            self.assertIn(marker, notes)

        stale = (
            f"Ghost-FTP-{version}-Setup-x64.exe",
            f"Ghost-FTP-{version}-Setup-x86.exe",
            f"Ghost-FTP-{version}-Setup-x32.exe",
            f"Ghost-FTP-{version}-Portable-x64.exe",
            f"Ghost-FTP-{version}-Portable-x86.exe",
            f"Ghost-FTP-{version}-Linux-amd64.deb",
            f"Ghost-FTP-{version}-Linux-multiarch.zip",
            "12 platform artifacts",
            "15 public release files",
        )
        for marker in stale:
            self.assertNotIn(marker, notes)


if __name__ == "__main__":
    unittest.main()
