#!/usr/bin/env python3
"""Validate Ghost FTP documentation and the Windows/Linux-only product contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
DOCS = ROOT / "docs"
INDEX = DOCS / "README.md"
MARKDOWN_LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)\n]+)\)")
HTML_LINK_RE = re.compile(r"\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)
CURRENT_RELEASE_RE = re.compile(r"\*\*Current Ghost FTP release:\s*(\d+\.\d+\.\d+)\*\*")
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")
RETIRED_ACTIVE_MARKERS = ("android/", "ios/", "macos/", "ghostftp web/", "web companion", "pwa")
ACTIVE_DOCS = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/ARCHITECTURE.md",
    "docs/ROADMAP.md",
    "docs/GITHUB-RELEASES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/CONTRIBUTING.md",
    "docs/PLATFORM-PARITY.md",
    "docs/VERSIONING.md",
)


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)


def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1:value.index(">")]
    elif value:
        value = value.split(maxsplit=1)[0]
    value = unquote(value).split("#", 1)[0].split("?", 1)[0]
    return value.strip()


def check_link(source: Path, raw: str) -> None:
    destination = clean_destination(raw)
    if not destination or destination.lower().startswith(IGNORED_PREFIXES):
        return
    target = (source.parent / destination).resolve()
    try:
        target.relative_to(ROOT.resolve())
    except ValueError:
        fail(f"link escapes repository: {source.relative_to(ROOT)} -> {raw}")
    if not target.exists():
        fail(f"missing local link: {source.relative_to(ROOT)} -> {raw}")


def main() -> int:
    version_path = ROOT / "VERSION"
    if not version_path.is_file():
        fail("VERSION is missing")
    version = version_path.read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    files = sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts)
    if not files or not INDEX.is_file():
        fail("documentation set is incomplete")

    for path in files:
        text = path.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in CURRENT_RELEASE_RE.finditer(text):
            if match.group(1) != version:
                fail(f"stale release marker in {path.relative_to(ROOT)}: {match.group(1)}")

    readme = (ROOT / "README.md").read_text(encoding="utf-8")
    index = INDEX.read_text(encoding="utf-8")
    if not readme.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    if not index.startswith("# Ghost FTP documentation\n"):
        fail("documentation index title is invalid")
    if f"Current Ghost FTP version: **{version}**" not in readme:
        fail("README VERSION marker is stale")
    if f"**Current Ghost FTP release: {version}**" not in index:
        fail("docs/README release marker is stale")

    if version.startswith("0."):
        for rel, text in (("README.md", readme), ("docs/README.md", index)):
            if "Development status: **Beta**" not in text:
                fail(f"{rel} must mark pre-1.0 releases as Beta")

    for rel in ACTIVE_DOCS:
        path = ROOT / rel
        if not path.is_file():
            fail(f"missing active document: {rel}")
        lowered = path.read_text(encoding="utf-8").lower()
        for marker in RETIRED_ACTIVE_MARKERS:
            if marker in lowered:
                fail(f"retired application surface appears in active guidance: {rel} -> {marker}")

    for required in ("docs/PLATFORM-PARITY.md", "docs/VERSIONING.md", "linux/README.md"):
        if not (ROOT / required).is_file():
            fail(f"missing required Windows/Linux documentation: {required}")

    for marker in ("Windows", "Linux", "24", "FTP", "FTPS", "SFTP"):
        if marker not in readme:
            fail(f"README is missing product marker: {marker}")

    for marker in ("9 platform artifacts", "12 public files"):
        if marker not in readme or marker not in index:
            fail(f"release contract marker missing: {marker}")

    parity = (DOCS / "PLATFORM-PARITY.md").read_text(encoding="utf-8")
    for marker in ("Windows and Linux platform parity", "SFTP password", "SFTP key passphrase", "24-language", "same typed `internal/api.Engine`"):
        if marker not in parity:
            fail(f"platform parity documentation missing marker: {marker}")

    versioning = (DOCS / "VERSIONING.md").read_text(encoding="utf-8")
    for marker in ("0.1.0", "0.x.y", "Beta", "1.0.0", "Setup", "Portable"):
        if marker not in versioning:
            fail(f"versioning documentation missing marker: {marker}")

    print(f"DOCS_AUDIT=PASS ({version}; {len(files)} Markdown files)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("RETIRED_APPLICATION_SURFACES=ANDROID,IOS,MACOS,WEB,PWA")
    print("PRE_1_0_CHANNEL=BETA")
    print("PUBLIC_PLATFORM_ARTIFACTS=9")
    print("PUBLIC_RELEASE_FILES=12")
    return 0


if __name__ == "__main__":
    sys.exit(main())
