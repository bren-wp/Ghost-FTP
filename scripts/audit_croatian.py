#!/usr/bin/env python3
"""Provjerava da su korisničke, GitHub i release površine ByFTP-a na hrvatskom."""

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]

REQUIRED = [
    ROOT / "docs" / "README.md",
    ROOT / "docs" / "ARHITEKTURA.md",
    ROOT / "docs" / "DOPRINOS.md",
    ROOT / "docs" / "PRIVATNOST.md",
    ROOT / "docs" / "PROVJERA-IZDANJA.md",
    ROOT / "docs" / "PLAN-RAZVOJA.md",
    ROOT / "docs" / "SIGURNOST.md",
    ROOT / "docs" / "POTPISIVANJE.md",
    ROOT / "docs" / "PODRSKA.md",
    ROOT / "docs" / "TESTIRANJE.md",
    ROOT / "docs" / "OBAVIJESTI-TRECIH-STRANA.md",
    ROOT / "docs" / "IZDAVANJE-NA-GITHUBU.md",
    ROOT / "docs" / "SHARED-HOSTING.md",
]

LEGACY_ROOT_DOCS = [
    "ARCHITECTURE.md", "CONTRIBUTING.md", "PRIVACY.md", "RELEASE-CHECKLIST.md",
    "ROADMAP.md", "SECURITY.md", "SIGNING.md", "SUPPORT.md", "TESTING.md",
    "THIRD-PARTY-NOTICES.md",
]

FORBIDDEN = {
    ROOT / "README.md": [
        "Current release", "## Download", "## Features", "## Requirements",
        "## Build from source", "## Quality gates", "## Repository structure",
        "## Authorized contributions", "## Documentation", "## Security reports",
    ],
    ROOT / ".github" / "ISSUE_TEMPLATE" / "bug_report.yml": [
        "Bug report", "What happened?", "Steps to reproduce", "Expected behavior",
        "Privacy check", "Not connection-related",
    ],
    ROOT / ".github" / "ISSUE_TEMPLATE" / "feature_request.yml": [
        "Feature request", "Problem or workflow", "Proposed behavior", "Project constraints",
    ],
    ROOT / ".github" / "pull_request_template.md": [
        "## Authorization", "## Summary", "Security & privacy checklist", "## Validation",
    ],
    ROOT / "internal" / "desktop" / "protocols_windows.go": ["FTPS (explicit)", "FTPS (implicit)"],
    ROOT / "internal" / "desktop" / "ui_windows.go": ["Poslužitelj / Host"],
    ROOT / "internal" / "brand" / "brand.go": ["ByFTP Client"],
    ROOT / "scripts" / "pe_resources.py": ["ByFTP Client", "Secure FTP, FTPS and SFTP client"],
    ROOT / ".github" / "workflows" / "release.yml": [
        "Build, Release & Package", "Create GitHub Release", "Upload complete release artifact",
        "Build GitHub Package", "Publish GitHub Package", "Official packaged Windows distribution",
    ],
    ROOT / ".github" / "workflows" / "ci.yml": [
        "Unit tests", "Race detector", "Privacy audit", "Release metadata validation",
        "Windows production build",
    ],
}


def fail(message: str) -> None:
    raise SystemExit("HRVATSKI_AUDIT_NIJE_PROSAO: " + message)


def main() -> int:
    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    for path in REQUIRED:
        if not path.is_file():
            fail(f"nedostaje dokument: {path.relative_to(ROOT)}")
    for name in LEGACY_ROOT_DOCS:
        if (ROOT / name).exists():
            fail(f"stari dokument još je u rootu: {name}")
    for path, phrases in FORBIDDEN.items():
        if not path.is_file():
            fail(f"nedostaje datoteka za provjeru: {path.relative_to(ROOT)}")
        text = path.read_text(encoding="utf-8")
        for phrase in phrases:
            if phrase in text:
                fail(f"pronađen je engleski korisnički tekst {phrase!r} u {path.relative_to(ROOT)}")
    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    for required in [
        f"Trenutačno izdanje: {version}",
        "## Preuzimanje",
        "## Instalacija, nadogradnja i uklanjanje",
        "## Sigurnost transfera",
        "## Dokumentacija",
    ]:
        if required not in readme:
            fail(f"README nema obavezni hrvatski tekst: {required}")
    protocols = (ROOT / "internal" / "desktop" / "protocols_windows.go").read_text(encoding="utf-8")
    for label in ["FTPS (eksplicitni)", "FTPS (implicitni)"]:
        if label not in protocols:
            fail(f"nedostaje hrvatska oznaka protokola: {label}")
    print(f"HRVATSKI_AUDIT=PROSAO ({version})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
