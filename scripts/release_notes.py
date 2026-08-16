#!/usr/bin/env python3
"""Generate ByFTP release notes from the matching CHANGELOG section."""

from __future__ import annotations

import argparse
from pathlib import Path
import re
import sys


def extract_section(changelog: str, version: str) -> str:
    header = re.compile(rf"^##\s+{re.escape(version)}(?:\s|$).*$", re.MULTILINE)
    match = header.search(changelog)
    if not match:
        raise ValueError(f"CHANGELOG section for {version} was not found")
    start = changelog.find("\n", match.end())
    if start < 0:
        return ""
    start += 1
    next_header = re.search(r"^##\s+", changelog[start:], re.MULTILINE)
    end = start + next_header.start() if next_header else len(changelog)
    return changelog[start:end].strip()


def build_notes(version: str, section: str) -> str:
    return f"""ByFTP {version}

Native Windows FTP / FTPS / SFTP client by Brendigo.

Release highlights
------------------
{section}

Downloads
---------
- Setup x64 EXE: recommended installation
- Portable x64 EXE: run without installation
- Uninstaller x64 EXE: standalone removal binary
- Windows x64 ZIP: complete ready-to-use Windows bundle
- Source ZIP: exact tracked source snapshot for this release
- SHA256.txt: checksums for public release artifacts
- verification.txt: PE/security verification report
- BUILD-METADATA.txt: source commit and build-toolchain provenance

Verify SHA-256 values before distribution. Public production binaries should be Authenticode-signed with the real Brendigo signing identity.
"""


def main() -> int:
    parser = argparse.ArgumentParser()
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
            raise ValueError(f"CHANGELOG section for {version} is empty")
        notes = build_notes(version, section)
        Path(args.output).write_text(notes, encoding="utf-8", newline="\n")
    except (OSError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
