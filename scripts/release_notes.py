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
    tag = f"ghostftp-v{version}"
    return f"""Ghost FTP {version}

Privacy-focused FTP, FTPS and SFTP client for Windows, Linux, macOS, Android, iOS and the web.

Highlights
----------
{section}

Release tag
-----------
{tag}

Public platform packages
------------------------
Windows:
- Ghost-FTP-{version}-Setup-x64.exe — 64-bit Windows installer.
- Ghost-FTP-{version}-Setup-x86.exe — 32-bit x86 Windows installer.
- Ghost-FTP-{version}-Setup-x32.exe — compatibility alias of the x86 installer; x32 and x86 refer to the same 32-bit architecture here.

Linux:
- Ghost-FTP-{version}-Linux-multiarch.zip — contains verified amd64, arm64 and i386 Debian packages.

macOS:
- Ghost-FTP-{version}-macOS-Universal.pkg — universal Intel x86_64 + Apple Silicon arm64 package.

Android:
- Ghost-FTP-{version}-Android.apk — installable CI test build using Android debug signing. It is not represented as a Play Store production-signed package.

iOS:
- Ghost-FTP-{version}-iOS-arm64-unsigned.ipa — unsigned device archive for external Apple signing/provisioning.

Web:
- Ghost-FTP-{version}-Web.zip — shared-hosting web package.

Verification files
------------------
- SHA256.txt — SHA-256 checksums for every public release file except SHA256.txt itself.
- RELEASE-NOTES.txt — these notes generated from CHANGELOG.md.
- BUILD-METADATA.txt — version, release tag, commit and signing/provenance status.

Signing and trust
-----------------
The workflow never fabricates publisher identities. Windows Authenticode, Apple Developer ID/notarization, Android production signing and iOS distribution signing require real external credentials. Where such credentials are not configured, the release notes and file names state the limitation explicitly. Always verify SHA256.txt before installation.
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
