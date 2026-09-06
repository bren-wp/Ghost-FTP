#!/usr/bin/env python3
"""Generate Ghost FTP release notes from the matching CHANGELOG section."""

from __future__ import annotations

import argparse
from pathlib import Path
import re
import sys


def extract_section(changelog: str, version: str) -> str:
    header = re.compile(rf"^##\s+{re.escape(version)}(?:\s|$).*$", re.MULTILINE)
    match = header.search(changelog)
    if not match:
        raise ValueError(f"CHANGELOG section for version {version} was not found")
    start = changelog.find("\n", match.end())
    if start < 0:
        return ""
    start += 1
    next_header = re.search(r"^##\s+", changelog[start:], re.MULTILINE)
    end = start + next_header.start() if next_header else len(changelog)
    return changelog[start:end].strip()


def build_notes(version: str, section: str) -> str:
    major = int(version.split(".", 1)[0])
    channel = "Beta prerelease" if major == 0 else "Stable"
    package_section = "" if major == 0 else f"""
GitHub Packages
---------------
- Package: ghcr.io/bren-wp/ghost-ftp:{version}
- Type: verified OCI distribution bundle, not a runtime container.
- Contents: the same verified release directory under /ghostftp-release/.
- Stable aliases: {major}, {'.'.join(version.split('.')[:2])}, latest.
- The workflow verifies registry readback before completing publication.
"""

    return f"""Ghost FTP {version}

Privacy-first FTP, FTPS and SFTP desktop client for Windows and Linux.
Release channel: {channel}.

Highlights
----------
{section}

Release tag
-----------
ghostftp-v{version}

Public platform packages
------------------------
Windows:
- Ghost-FTP-{version}-Setup-x64.exe — 64-bit Windows installer.
- Ghost-FTP-{version}-Setup-x86.exe — 32-bit x86 Windows installer.
- Ghost-FTP-{version}-Setup-x32.exe — byte-identical compatibility alias of the x86 installer.
- Ghost-FTP-{version}-Portable-x64.exe — portable 64-bit Windows executable.
- Ghost-FTP-{version}-Portable-x86.exe — portable 32-bit x86 Windows executable.

Linux:
- Ghost-FTP-{version}-Linux-amd64.deb — Debian package for amd64.
- Ghost-FTP-{version}-Linux-arm64.deb — Debian package for arm64.
- Ghost-FTP-{version}-Linux-i386.deb — Debian package for i386.
- Ghost-FTP-{version}-Linux-multiarch.zip — bundle containing the three verified Debian packages.
{package_section}
Verification files
------------------
- SHA256.txt — SHA-256 checksums for every public release file except SHA256.txt itself.
- RELEASE-NOTES.txt — these notes generated from CHANGELOG.md.
- BUILD-METADATA.txt — version, release tag, source commit, signing state and distribution metadata.

Release contract
----------------
- 9 platform artifacts.
- 12 public release files total, including the three verification/metadata files.
- Active application platforms: Windows and Linux.
- Local language catalog: 24 selectable languages with English default/fallback.
- Application telemetry: disabled.

Signing and trust
-----------------
The workflow never fabricates publisher identities. Stable Windows publication requires the configured trusted Authenticode identity. Release signing secrets are supplied only through protected GitHub Actions secrets and are removed from the runner after use. Always verify SHA256.txt before installation or deployment.

Privacy
-------
Release bundles contain only the explicit verified artifact allow-list. They do not contain saved profiles, FTP/SFTP passwords, private-key passphrases, signing private keys, local application data or user files.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate Ghost FTP release notes from CHANGELOG.md")
    parser.add_argument("--version", required=True)
    parser.add_argument("--changelog", default="CHANGELOG.md")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    version = args.version.strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        print(f"invalid release version: {version}", file=sys.stderr)
        return 2

    try:
        changelog = Path(args.changelog).read_text(encoding="utf-8")
        section = extract_section(changelog, version)
        if not section:
            raise ValueError(f"CHANGELOG section for version {version} is empty")
        Path(args.output).write_text(build_notes(version, section), encoding="utf-8", newline="\n")
    except (OSError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
