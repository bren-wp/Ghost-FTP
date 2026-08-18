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

    # Detaljna dokumentacija treba biti dugovječna i ne smije skrivati stari
    # "trenutačni" broj verzije u proznom tekstu. Ako neki dokument ipak ima
    # eksplicitnu oznaku trenutačnog izdanja, ona mora odgovarati VERSION-u.
    for path in sorted((ROOT / "docs").rglob("*.md")):
        text = path.read_text(encoding="utf-8")
        rel = path.relative_to(ROOT)
        for match in re.finditer(r"Trenutačno izdanje:\s*(\d+\.\d+\.\d+)", text):
            if match.group(1) != version:
                fail(f"{rel} prikazuje zastarjelo trenutačno izdanje {match.group(1)}")

    security_doc = read("docs/SIGURNOST.md")
    if "Aktualna produkcijska bazna linija dodatno učvršćuje release granicu:" not in security_doc:
        fail("SIGURNOST.md mora opisivati release sigurnost bez hardkodiranog starog broja verzije")
    if re.search(r"(?m)^\d+\.\d+\.\d+\s+dodatno učvršćuje release granicu:", security_doc):
        fail("SIGURNOST.md hardkodira broj verzije u aktualni release-sigurnosni opis")

    plan = read("docs/PLAN-RAZVOJA.md")
    if "## Završene produkcijske cjeline" not in plan:
        fail("PLAN-RAZVOJA.md mora koristiti verzijski neutralan naslov završenih cjelina")
    if re.search(r"(?m)^##\s+Završeno u\s+\d", plan):
        fail("PLAN-RAZVOJA.md sadrži zastarjeli verzionirani naslov završenih cjelina")

    windows_build = read("BUILD-WINDOWS.ps1")
    if "Get-Content -LiteralPath $versionFile" not in windows_build or "-X main.version=$version" not in windows_build:
        fail("Windows build ne povezuje VERSION u runtime verziju")

    for rel in ("scripts/BUILD-LOCAL.sh", "scripts/BUILD-LINUX.sh", "scripts/BUILD-MACOS.sh"):
        text = read(rel)
        if "< VERSION" not in text:
            fail(f"{rel} ne čita kanonski VERSION")
    if "-X main.version=$VERSION" not in read("scripts/BUILD-LOCAL.sh"):
        fail("lokalni build ne povezuje VERSION u runtime verziju")
    if "-X main.version=${VERSION}" not in read("scripts/BUILD-LINUX.sh"):
        fail("Linux build ne povezuje VERSION u runtime verziju")
    if "-X main.version=${VERSION}" not in read("scripts/BUILD-MACOS.sh"):
        fail("macOS build ne povezuje VERSION u runtime verziju")

    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow ima hardkodiranu zadanu produkcijsku verziju")
    if re.search(r"(?i)primjer\s+\d+\.\d+\.\d+", release_workflow):
        fail("release workflow ima hardkodirani verzijski primjer")
    for marker in ("$manual", "Get-Content -LiteralPath 'VERSION' -Raw"):
        if marker not in release_workflow:
            fail(f"release workflow nema VERSION fallback marker: {marker}")

    # GitHub Package mora biti izveden iz istog VERSION izvora kao runtime i
    # GitHub Release. Time budući refaktor ne može tiho ostaviti stari paket.
    package_markers = (
        "<PackageId>ByFTP.Windows</PackageId>",
        "<Version>$env:VERSION</Version>",
        "dotnet nuget push",
        "nuget.pkg.github.com/bren-wp/index.json",
        "--skip-duplicate",
    )
    for marker in package_markers:
        if marker not in release_workflow:
            fail(f"GitHub Package nije vezan uz kanonski VERSION: nedostaje {marker}")

    bug_template = read(".github/ISSUE_TEMPLATE/bug_report.yml")
    if re.search(r"(?m)^\s*placeholder:\s*['\"]\d+\.\d+\.\d+['\"]", bug_template):
        fail("bug predložak hardkodira trenutačnu verziju")

    croatian_audit = read("scripts/audit_croatian.py")
    if re.search(r"Trenutačno izdanje:\s*\d+\.\d+\.\d+", croatian_audit):
        fail("hrvatski audit ponovno hardkodira trenutačnu verziju")
    if 'version = (ROOT / "VERSION")' not in croatian_audit:
        fail("hrvatski audit ne čita VERSION dinamički")

    print(f"VERSION_AUDIT=PROSAO ({version})")
    print("PLATFORM_VERSION_SOURCES=WINDOWS,LINUX,MACOS")
    print("GITHUB_PACKAGE_VERSION_SOURCE=VERSION")
    print("PRODUCTION_DOC_VERSION_DRIFT=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
