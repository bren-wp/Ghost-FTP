#!/usr/bin/env python3
"""Validate Ghost FTP documentation against the active Windows/Linux/Android product contract."""
from __future__ import annotations

import re
import sys
from pathlib import Path
from urllib.parse import unquote

ROOT = Path(__file__).resolve().parents[1]
MARKDOWN_LINK_RE = re.compile(r"!?\[[^\]]*\]\(([^)\n]+)\)")
MARKDOWN_IMAGE_RE = re.compile(r"!\[[^\]]*\]\(([^)\n]+)\)")
HTML_LINK_RE = re.compile(r"\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']", re.I)
HTML_IMAGE_RE = re.compile(r"<img\b[^>]*\bsrc\s*=\s*[\"']([^\"']+)[\"']", re.I)
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")
REMOTE_MEDIA_PREFIXES = ("http://", "https://", "data:", "//")

ACTIVE_DOCS = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/ARCHITECTURE.md",
    "docs/ROADMAP.md",
    "docs/GITHUB-RELEASES.md",
    "docs/PACKAGES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/CONTRIBUTING.md",
    "docs/PLATFORM-PARITY.md",
    "docs/VERSIONING.md",
    "docs/SECURITY.md",
    "docs/PRIVACY.md",
    "docs/SIGNING.md",
    "docs/LOCALIZATION.md",
    "docs/DEPENDENCIES.md",
    "docs/SETTINGS.md",
    "docs/TESTING.md",
    "docs/SUPPORT.md",
    "docs/REFERENCE-UI.md",
    "docs/NAVIGATION-BOOKMARKS.md",
    "docs/QUEUE-PRIORITY.md",
    "docs/THIRD-PARTY-NOTICES.md",
    "linux/README.md",
    "android/README.md",
    "android/UI-UX.md",
    "extensions/README.md",
    "extensions/PRIVACY.md",
    "scripts/README.md",
)

VISUAL_ASSETS = (
    "build/icon.png",
    "docs/images/ghost-ftp-main-workspace.png",
    "docs/images/ghost-ftp-linux-main-workspace.png",
    "docs/images/ghost-ftp-android-files.png",
    "docs/images/readme/transfer.svg",
    "docs/images/readme/security.svg",
    "docs/images/readme/privacy.svg",
)

RETIRED_PATHS = (
    "macos",
    ".github/workflows/macos-app.yml",
    ".github/workflows/macos-production.yml",
    "web",
    "web-ftp",
    "webftp",
    "pwa",
)

def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)

def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        fail(f"missing active document: {relative}")
    return path.read_text(encoding="utf-8")

def clean_destination(raw: str) -> str:
    value = raw.strip()
    if value.startswith("<") and ">" in value:
        value = value[1:value.index(">")]
    elif value:
        value = value.split(maxsplit=1)[0]
    return unquote(value).split("#", 1)[0].split("?", 1)[0].strip()

def check_link(source: Path, raw: str) -> None:
    dest = clean_destination(raw)
    if not dest or dest.lower().startswith(IGNORED_PREFIXES):
        return
    target = (source.parent / dest).resolve()
    try:
        target.relative_to(ROOT.resolve())
    except ValueError:
        fail(f"link escapes repository: {source.relative_to(ROOT)} -> {raw}")
    if not target.exists():
        fail(f"missing local link: {source.relative_to(ROOT)} -> {raw}")

def check_media(source: Path, raw: str) -> None:
    dest = clean_destination(raw)
    if not dest:
        fail(f"empty documentation media source: {source.relative_to(ROOT)}")
    if dest.lower().startswith(REMOTE_MEDIA_PREFIXES):
        fail(f"remote/data documentation media is blocked: {source.relative_to(ROOT)} -> {raw}")
    check_link(source, raw)

def require(label: str, text: str, *markers: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{label} missing marker: {marker}")

def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    for retired in RETIRED_PATHS:
        if (ROOT / retired).exists():
            fail(f"retired application surface is still present: {retired}")

    for rel in ACTIVE_DOCS:
        read(rel)

    for path in sorted(x for x in ROOT.rglob("*.md") if ".git" not in x.parts):
        text = path.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in MARKDOWN_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))
        for match in HTML_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))

    for rel in VISUAL_ASSETS:
        path = ROOT / rel
        if not path.is_file() or path.stat().st_size <= 0:
            fail(f"missing maintained local documentation visual: {rel}")

    readme = read("README.md")
    require(
        "README",
        readme,
        "<h1>Ghost FTP</h1>",
        "Windows · Linux · Android",
        "docs/images/ghost-ftp-main-workspace.png",
        "docs/images/ghost-ftp-linux-main-workspace.png",
        "docs/images/ghost-ftp-android-files.png",
        "build/icon.png",
        "macOS is retired",
        "Published tags are immutable",
    )

    index = read("docs/README.md")
    require("documentation index", index, "Windows, Linux and Android", "macOS is retired")

    packages = read("docs/PACKAGES.md")
    require(
        "package contract",
        packages,
        "PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER",
        "ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID",
        "13 platform artifacts / 16 public files",
    )

    verification = read("docs/RELEASE-VERIFICATION.md")
    require(
        "release verification",
        verification,
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
        "Windows, Linux and Android",
    )

    reference = read("docs/REFERENCE-UI.md")
    require(
        "reference UI",
        reference,
        "Back",
        "Forward",
        "Refresh",
        "New Folder",
        "Upload",
        "Download",
        "Bookmarks",
        "More",
        "Generated images and visual references are design targets, not proof of execution.",
    )

    release = read(".github/workflows/release.yml")
    for retired in ("macos:", "Ghost-FTP-${VERSION}-macOS", "MACOS_APP=", "APPLE_NOTARY"):
        if retired in release:
            fail(f"release workflow contains retired macOS marker: {retired}")
    require(
        "release workflow",
        release,
        "needs: [quality, windows, linux, android, browser]",
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
    )

    print(f"DOCS_AUDIT=PASS ({version}; Windows/Linux/Android active)")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID")
    print("MACOS_SOURCE_SURFACE=RETIRED")
    return 0

if __name__ == "__main__":
    sys.exit(main())
