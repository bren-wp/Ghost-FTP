#!/usr/bin/env python3
"""Provjerava da ByFTP koristi VERSION kao jedini produkcijski izvor verzije."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")


def fail(message: str) -> None:
    raise SystemExit("VERSION_AUDIT_NIJE_PROSAO: " + message)


def read(path: str) -> str:
    p = ROOT / path
    if not p.is_file():
        fail(f"nedostaje {path}")
    return p.read_text(encoding="utf-8")


def main() -> int:
    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION nije semantička verzija: {version!r}")

    # Izravno pokretanje `go run`/`go build` bez produkcijskih ldflags mora
    # jasno pokazati razvojni build, a ne zastarjelu produkcijsku verziju.
    for rel in ("cmd/byftp/main.go", "cmd/installer/main.go"):
        text = read(rel)
        if 'var version = "dev"' not in text:
            fail(f"{rel} nema sigurni razvojni fallback verzije")
        stale = re.search(r'var\s+version\s*=\s*"\d+\.\d+\.\d+"', text)
        if stale:
            fail(f"{rel} sadrži hardkodiranu produkcijsku verziju: {stale.group(0)}")

    readme = read("README.md")
    if f"Trenutačno izdanje: {version}" not in readme:
        fail("README ne prikazuje VERSION kao trenutačno izdanje")

    changelog = read("CHANGELOG.md")
    if f"## {version}" not in changelog:
        fail("CHANGELOG nema odjeljak za VERSION")

    windows_build = read("BUILD-WINDOWS.ps1")
    if "Get-Content -LiteralPath $versionFile" not in windows_build or "-X main.version=$version" not in windows_build:
        fail("Windows build ne povezuje VERSION u runtime verziju")

    local_build = read("scripts/BUILD-LOCAL.sh")
    if "VERSION=\"$(tr -d '\\r\\n' < VERSION)\"" not in local_build or "-X main.version=$VERSION" not in local_build:
        fail("lokalni build ne koristi VERSION kao runtime verziju")

    # Operativne površine ne smiju ponovno uvoditi ručno sinkronizirane brojeve
    # verzije. Workflow može primiti ručni broj, ali zadani slučaj mora čitati VERSION.
    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow ima hardkodiranu zadanu produkcijsku verziju")
    if re.search(r"(?i)primjer\s+\d+\.\d+\.\d+", release_workflow):
        fail("release workflow ima hardkodirani verzijski primjer")
    for marker in ("$manualVersion", "Get-Content -LiteralPath 'VERSION' -Raw"):
        if marker not in release_workflow:
            fail(f"release workflow nema VERSION fallback marker: {marker}")

    bug_template = read(".github/ISSUE_TEMPLATE/bug_report.yml")
    if re.search(r"(?m)^\s*placeholder:\s*['\"]\d+\.\d+\.\d+['\"]", bug_template):
        fail("bug predložak hardkodira trenutačnu verziju")

    croatian_audit = read("scripts/audit_croatian.py")
    if re.search(r"Trenutačno izdanje:\s*\d+\.\d+\.\d+", croatian_audit):
        fail("hrvatski audit ponovno hardkodira trenutačnu verziju")
    if 'version = (ROOT / "VERSION")' not in croatian_audit:
        fail("hrvatski audit ne čita VERSION dinamički")

    print(f"VERSION_AUDIT=PROSAO ({version})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
