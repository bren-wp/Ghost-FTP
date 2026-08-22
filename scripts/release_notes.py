#!/usr/bin/env python3
"""Generate ByFTP release notes from the exact matching CHANGELOG section."""

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

Privacy-focused FTP / FTPS / SFTP client for Windows, Linux and macOS.

Key changes
-----------
{section}

Official packages
-----------------
Windows:
- Setup x64 EXE: recommended installation for 64-bit Windows 10/11
- Portable x64 EXE: 64-bit use without installation
- Windows x64 ZIP: Setup + Portable + documentation + bundle checksums
- Setup x86 EXE: installation for supported 32-bit Windows
- Portable x86 EXE: 32-bit use without installation
- Windows x86 ZIP: Setup + Portable + documentation + bundle checksums

Linux:
- Linux amd64 DEB
- Linux arm64 DEB
- Linux i386 DEB

macOS:
- macOS Universal PKG: Intel x86_64 + Apple Silicon arm64

Release verification:
- SHA256.txt: SHA-256 checksums for all public packages and shared release metadata
- RELEASE-NOTES.txt: these release notes
- BUILD-METADATA.txt: version, release commit and production-build provenance

Before installation
-------------------
1. Download the package matching your operating system and architecture.
2. Compare the package SHA-256 with the official SHA256.txt.
3. Windows users should normally choose Setup; Portable is intended for use without installation.
4. Linux and macOS editions use a terminal interface over the same ByFTP engine and security/transfer core.

Signing status
--------------
Windows binaries do not show Verified Publisher until a real Brendigo Authenticode certificate is available. The macOS PKG is not Developer ID signed/notarized without a real Apple certificate. The workflow never fabricates publisher identity; SHA-256 and release provenance remain mandatory release evidence.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generate ByFTP release notes from CHANGELOG")
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
