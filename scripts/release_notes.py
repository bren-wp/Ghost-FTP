#!/usr/bin/env python3
"""Generira hrvatske ByFTP bilješke iz točno odgovarajućeg CHANGELOG odjeljka."""

from __future__ import annotations

import argparse
from pathlib import Path
import re
import sys


def extract_section(changelog: str, version: str) -> str:
    header = re.compile(rf"^##\s+{re.escape(version)}(?:\s|$).*$", re.MULTILINE)
    match = header.search(changelog)
    if not match:
        raise ValueError(f"CHANGELOG odjeljak za verziju {version} nije pronađen")
    start = changelog.find("\n", match.end())
    if start < 0:
        return ""
    start += 1
    next_header = re.search(r"^##\s+", changelog[start:], re.MULTILINE)
    end = start + next_header.start() if next_header else len(changelog)
    return changelog[start:end].strip()


def build_notes(version: str, section: str) -> str:
    return f"""ByFTP {version}

Nova stabilna ByFTP 1.x linija tvrtke Brendigo: moderni Windows file-transfer suite i zasebni FTP/FTPS, SFTP, SSH i S3 klijenti.

Najvažnije promjene
-------------------
{section}

Službeni GitHub Release
-----------------------
ByFTP All-in-One:
- Setup x64 EXE
- Setup x86 EXE
- Portable x64 EXE
- Portable x86 EXE

Zasebni klijenti:
- FTP Client Portable x64 / x86
- SFTP Client Portable x64 / x86
- SSH Client Portable x64 / x86
- S3 Client Portable x64 / x86

GitHub uz tag automatski prikazuje standardne Source code (zip) i Source code (tar.gz) poveznice. ByFTP ne dodaje vlastiti Source.zip, verification.txt, standalone Uninstall EXE, Windows ZIP bundle ili release metadata datoteke kao javne assete.

GitHub Packages
---------------
Ista verzija objavljuje se u paketima:
- ByFTP.Suite
- ByFTP.FTP.Client
- ByFTP.SFTP.Client
- ByFTP.SSH.Client
- ByFTP.S3.Client

Sigurnost i privatnost
----------------------
- nema telemetrije aplikacije;
- produkcijski build zahtijeva stvarno isključenu Go build telemetriju;
- SFTP i SSH koriste sistemski OpenSSH uz fail-closed sigurnosne granice;
- S3 koristi vlastitu AWS Signature Version 4 implementaciju bez AWS SDK ovisnosti;
- vjerodajnice se ne stavljaju u URL ili argumente procesa gdje ih ByFTP može izbjeći.

Potpisivanje
------------
Windows binariji nemaju status Verified Publisher dok nije dostupan stvarni Brendigo Authenticode certifikat. Workflow ne fabricira publisher identitet.
"""


def main() -> int:
    parser = argparse.ArgumentParser(description="Generiranje hrvatskih ByFTP bilješki iz CHANGELOG-a")
    parser.add_argument("--version", required=True)
    parser.add_argument("--changelog", default="CHANGELOG.md")
    parser.add_argument("--output", required=True)
    args = parser.parse_args()

    version = args.version.strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        print(f"neispravna verzija izdanja: {version}", file=sys.stderr)
        return 2
    try:
        changelog = Path(args.changelog).read_text(encoding="utf-8")
        section = extract_section(changelog, version)
        if not section:
            raise ValueError(f"CHANGELOG odjeljak za verziju {version} je prazan")
        Path(args.output).write_text(build_notes(version, section), encoding="utf-8", newline="\n")
    except (OSError, ValueError) as exc:
        print(str(exc), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
