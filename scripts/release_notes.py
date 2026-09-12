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
    parts = version.split(".")
    major = int(parts[0])
    minor_alias = ".".join(parts[:2])

    return f"""Ghost FTP {version}

Privacy-first FTP, FTPS and SFTP desktop client for Windows and Linux.
Release channel: Current.
GitHub prerelease flag: false.

Highlights
----------
{section}

Release tag
-----------
ghostftp-v{version}

Public platform packages
------------------------
Windows:
- Ghost-FTP-{version}-Setup.exe — self-contained universal Windows Setup launcher with verified native x64/x86 application payloads.
- Ghost-FTP-{version}-Portable.exe — self-contained universal Windows Portable launcher with verified native x64/x86 application payloads.

Linux / Debian:
- Ghost-FTP-{version}-Linux-Debian-amd64.deb
- Ghost-FTP-{version}-Linux-Debian-arm64.deb
- Ghost-FTP-{version}-Linux-Debian-i386.deb

Linux / Ubuntu:
- Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb
- Ghost-FTP-{version}-Linux-Ubuntu-arm64.deb
- Ghost-FTP-{version}-Linux-Ubuntu-i386.deb

Linux / Fedora:
- Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm
- Ghost-FTP-{version}-Linux-Fedora-aarch64.rpm
- Ghost-FTP-{version}-Linux-Fedora-i686.rpm

Linux / Portable:
- Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz
- Ghost-FTP-{version}-Linux-Portable-arm64.tar.gz
- Ghost-FTP-{version}-Linux-Portable-i386.tar.gz

GitHub Packages
---------------
- Package: ghcr.io/bren-wp/ghost-ftp:{version}
- Type: verified OCI distribution bundle, not a runtime container.
- Contents: the same verified release directory under /ghostftp-release/.
- Current aliases: {major}, {minor_alias}, latest.
- The workflow verifies the exact-version registry readback before completing publication.

Verification files
------------------
- SHA256.txt — SHA-256 checksums for every public release file except SHA256.txt itself.
- RELEASE-NOTES.txt — these notes generated from CHANGELOG.md.
- BUILD-METADATA.txt — version, release tag, exact source commit, signing state and distribution metadata.

Release contract
----------------
- Current Ghost FTP releases are not inferred to be prereleases from semantic-version major zero.
- 14 platform artifacts.
- 17 public release files total, including BUILD-METADATA.txt, RELEASE-NOTES.txt and SHA256.txt.
- Public release platforms: Windows and Linux.
- The Android development APK is independently exact-head verified but remains outside the public Windows/Linux release allow-list.
- Local language catalog: 24 selectable desktop languages with English default/fallback.
- Application telemetry: disabled.
- Linux Debian/Ubuntu/Fedora/Portable packages reuse one verified production executable per matching architecture and are byte-parity checked before publication.
- Publication is bound to the exact verified main commit and followed by canonical latest-only retention verification.

Signing and trust
-----------------
The workflow never fabricates publisher identities. Production Authenticode signing is optional: when a protected trusted certificate is configured, Windows artifacts are signed and verified; when it is not configured, the release remains explicitly unsigned and BUILD-METADATA.txt records WINDOWS_AUTHENTICODE=unsigned. Never treat a locally generated or self-signed certificate as a trusted public publisher identity. Always verify SHA256.txt and the official GitHub release location before installation or deployment.

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
