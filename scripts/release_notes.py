#!/usr/bin/env python3
"""Generate English ByFTP release notes from the exact matching CHANGELOG section."""

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
    return f"""ByFTP {version}

Privacy-focused FTP / FTPS / SFTP client for Windows, Linux, macOS and Android.

Highlights
----------
{section}

Official desktop packages
-------------------------
Windows:
- Setup x64 EXE: recommended installer for 64-bit Windows 10/11
- Portable x64 EXE: 64-bit build that runs without installation
- Windows x64 ZIP: Setup + Portable + documentation + bundle checksums
- Setup x86 EXE: installer for supported 32-bit Windows systems
- Portable x86 EXE: 32-bit build that runs without installation
- Windows x86 ZIP: Setup + Portable + documentation + bundle checksums

Linux:
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB

macOS:
- macOS Universal PKG: Intel x86_64 + Apple Silicon arm64

Android
-------
The native Android source is under android/ and is a required release-quality gate starting with 1.1.0. CI verifies Android unit tests, lint and APK compilation. A public production Android APK is not published until a stable private Android signing identity is configured outside the repository; debug-signed CI artifacts are not production packages.

Release verification
--------------------
- SHA256.txt: SHA-256 checksums for every public desktop package and shared release metadata file
- RELEASE-NOTES.txt: these release notes generated from the matching CHANGELOG section
- BUILD-METADATA.txt: version, release commit and production workflow provenance

Before installing
-----------------
1. Download the package that matches your operating system and architecture.
2. Compare the downloaded desktop package SHA-256 hash with the official SHA256.txt file.
3. On Windows, use Setup for a normal installation; use Portable when installation is not required.
4. Do not treat Android CI debug APK artifacts as a production-signed release.

Signing status
--------------
Windows binaries do not show Verified Publisher until a real ByFTP Authenticode certificate is available. The macOS PKG is not Developer ID signed/notarized without a real Apple signing identity. Android production distribution likewise requires a stable private signing identity. The workflow never fabricates a publisher identity; SHA-256 verification and release provenance remain mandatory parts of public desktop releases.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate ByFTP release notes from CHANGELOG.md")
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
