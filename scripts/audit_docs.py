#!/usr/bin/env python3
"""Validate maintained Ghost FTP documentation against the active 0.0.6 contract."""
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
    "android/TRANSFER-PROGRESS.md",
    "macos/README.md",
    "macos/PARITY.md",
    "extensions/README.md",
    "extensions/PRIVACY.md",
    "scripts/README.md",
)
RELEASE_FACING_DOCS = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/GITHUB-RELEASES.md",
    "docs/RELEASE-VERIFICATION.md",
)
CURRENT_VERSION_DOCS = (
    "README.md",
    "docs/README.md",
    "docs/INSTALLATION.md",
    "docs/GITHUB-RELEASES.md",
    "docs/PACKAGES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/VERSIONING.md",
    "docs/SUPPORT.md",
    "docs/ARCHITECTURE.md",
    "docs/PLATFORM-PARITY.md",
    "docs/SIGNING.md",
    "docs/TESTING.md",
    "linux/README.md",
    "android/README.md",
)
VISUAL_ASSETS = (
    "build/icon.png",
    "docs/images/0.0.6/ghost-ftp-main-workspace.png",
    "docs/images/0.0.6/ghost-ftp-site-manager.png",
    "docs/images/0.0.6/ghost-ftp-settings.png",
    "docs/images/0.0.6/ghost-ftp-about.png",
    "docs/images/0.0.6/ghost-ftp-linux-main-workspace.png",
    "docs/images/0.0.6/ghost-ftp-android-files.png",
)
RETIRED_WEB_PATHS = (
    "web",
    ".github/workflows/web.yml",
    "docs/WEB.md",
    "scripts/check_web_contract.py",
    "scripts/test_web_contract.py",
    "docs/prompts/GHOST-FTP-WEB-APP-PROMPT.md",
    "docs/prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md",
)


def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        fail(f"missing active document: {relative}")
    return path.read_text(encoding="utf-8")


def require(label: str, text: str, *markers: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{label} missing marker: {marker}")


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


def main() -> int:
    version = read("VERSION").strip()
    if version != "0.0.6":
        fail(f"active documentation contract expects VERSION 0.0.6, got {version!r}")

    for retired in RETIRED_WEB_PATHS:
        if (ROOT / retired).exists():
            fail(f"retired web surface is still present: {retired}")

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

    for rel in CURRENT_VERSION_DOCS:
        if version not in read(rel):
            fail(f"current-version document does not mention {version}: {rel}")

    for rel in ACTIVE_DOCS:
        text = read(rel)
        lower = text.lower()
        if "0.0.7" in text:
            fail(f"future 0.0.7 identity leaked into active 0.0.6 documentation: {rel}")
        if "ghostftp web/" in lower or "ios/" in lower:
            fail(f"retired application surface appears in active guidance: {rel}")
        if "ghostftp_android_signer_sha256" in lower:
            fail(f"legacy Android signer secret appears in active guidance: {rel}")

    for rel in RELEASE_FACING_DOCS:
        text = read(rel)
        for stale in (
            "18 platform artifacts / 21 public files",
            f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
            f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
            f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
        ):
            if stale in text:
                fail(f"stale 0.0.6 release shape in {rel}: {stale}")

    readme = read("README.md")
    if not readme.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    require(
        "README 0.0.6 identity",
        readme,
        "Current source version: **0.0.6**",
        "Last actually published GitHub Release: **0.0.5**",
        "13 platform artifacts / 16 public files",
        "Ghost-FTP-0.0.6-Linux-Debian-Installer.run",
        "Ghost-FTP-0.0.6-Linux-Fedora-Portable.tar.gz",
        "Ghost-FTP-0.0.6-Android.apk",
        "Ghost-FTP-0.0.6-Opera-Extension.zip",
        "GHOSTFTP_ANDROID_CERT_SHA256",
        "sanitized browser-to-desktop handoff",
        "Brendigo LTD",
        "proprietary commercial",
        "repository-local",
        "exact final head SHA",
    )

    index = read("docs/README.md")
    require(
        "documentation index",
        index,
        "Current source version: **0.0.6**",
        "Last actually published GitHub Release: **0.0.5**",
        "13 platform artifacts / 16 public files",
        "Chrome, Edge, Firefox and Opera",
        "proprietary commercial software",
        "../extensions/README.md",
    )

    installation = read("docs/INSTALLATION.md")
    require(
        "installation",
        installation,
        "Ghost FTP **0.0.6** is the active release candidate",
        "13 platform artifacts / 16 public files",
        "Ghost-FTP-0.0.6-Linux-Debian-Installer.run",
        "Ghost-FTP-0.0.6-Linux-Ubuntu-Portable.tar.gz",
        "Ghost-FTP-0.0.6-Linux-Fedora-Installer.run",
        "Ghost-FTP-0.0.6-Opera-Extension.zip",
        "ghostftp-uninstall",
        "GHOSTFTP_ANDROID_CERT_SHA256",
        "SFTP remains intentionally hidden",
    )

    releases = read("docs/GITHUB-RELEASES.md")
    require(
        "GitHub release documentation",
        releases,
        "Ghost FTP **0.0.6** is the active release candidate",
        "last actually published GitHub Release is **0.0.5**",
        "ghostftp-v0.0.6",
        "13 platform artifacts / 16 public files",
        "Ghost-FTP-0.0.6-Opera-Extension.zip",
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
        "release/ghostftp-vX.Y.Z",
        "does not publish a release directly",
    )

    verification = read("docs/RELEASE-VERIFICATION.md")
    require(
        "release verification",
        verification,
        "Ghost FTP **0.0.6** is the active release candidate",
        "VERSION=0.0.6",
        "TAG=ghostftp-v0.0.6",
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
        "Ghost-FTP-0.0.6-Opera-Extension.zip",
        "GHOSTFTP_ANDROID_CERT_SHA256",
        "release/ghostftp-vX.Y.Z",
        "must never publish a release directly",
    )

    signing = read("docs/SIGNING.md")
    require(
        "signing documentation",
        signing,
        "Ghost FTP **0.0.6**",
        "Official Windows publication is **signed-only**.",
        "GHOSTFTP_SIGNING_PFX_BASE64",
        "GHOSTFTP_ANDROID_KEYSTORE_BASE64",
        "GHOSTFTP_ANDROID_CERT_SHA256",
        "Developer ID Application",
    )

    browser = read("extensions/README.md")
    require(
        "browser helper documentation",
        browser,
        "Google Chrome",
        "Microsoft Edge",
        "Mozilla Firefox",
        "Opera",
        "ghostftp://connect",
        "Open in Ghost FTP",
        "Zero browser permissions and zero host permissions",
    )

    android = read("android/README.md")
    require(
        "Android documentation",
        android,
        "Ghost FTP **0.0.6**",
        "Ghost-FTP-0.0.6-Android.apk",
        "FTP and explicit FTPS",
        "SFTP is intentionally not exposed",
        "Storage Access Framework",
    )

    reference = read("docs/REFERENCE-UI.md")
    require(
        "authentic UI evidence",
        reference,
        "Windows — 5 images",
        "Linux — 3 images",
        "Android — 7 images",
        "exactly **15 runtime images**",
        "Mockups, image-generation output and manually composed approximations are not accepted",
    )

    print(f"DOCS_AUDIT=PASS ({version}; 13 platform artifacts / 16 public files)")
    print("LAST_PUBLISHED_GITHUB_RELEASE=0.0.5")
    print("NEXT_PUBLIC_RELEASE=0.0.6")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("ACTIVE_WEB_SURFACE=NONE")
    print("ANDROID_SFTP=HIDDEN_UNTIL_STRICT_HOST_KEY_VERIFICATION")
    print("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX,OPERA")
    print("BROWSER_DESKTOP_HANDOFF=WINDOWS_SANITIZED_PROTOCOL")
    print("MACOS_PUBLIC_RELEASE=NO")
    return 0


if __name__ == "__main__":
    sys.exit(main())
