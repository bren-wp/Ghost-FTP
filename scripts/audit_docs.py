#!/usr/bin/env python3
"""Validate Ghost FTP active documentation against the current release contract."""

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
STALE_SIGNING_POLICY_MARKERS = (
    "trusted authenticode requirement for stable windows publication",
    "a stable windows release is blocked unless",
    "stable release whose windows signing state is not trusted/configured",
    "stable windows authenticode gate",
    "stable windows signing gate",
    "stable windows publication requires",
    "stable windows publication additionally requires",
)
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
    "linux/README.md",
    "scripts/README.md",
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


def require_markers(label: str, text: str, markers: tuple[str, ...]) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{label} missing marker: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    major = int(version.split(".", 1)[0])

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

    readme = read("README.md")
    index = read("docs/README.md")
    if not readme.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    if not index.startswith("# Ghost FTP documentation\n"):
        fail("documentation index title is invalid")
    require_markers(
        "README current release",
        readme,
        (
            f"Current Ghost FTP version: **{version}**",
            "Development status: **Stable**" if major >= 1 else "Development status: **Beta**",
            "Windows",
            "Linux",
            "24",
            "FTP",
            "FTPS",
            "SFTP",
        ),
    )
    require_markers(
        "documentation index current release",
        index,
        (
            f"**Current Ghost FTP release: {version}**",
            "prerelease=false" if major >= 1 else "Development status: **Beta**",
        ),
    )

    for relative in ACTIVE_DOCS:
        text = read(relative)
        lowered = text.lower()
        for marker in RETIRED_ACTIVE_MARKERS:
            if marker in lowered:
                fail(f"retired application surface appears in active guidance: {relative} -> {marker}")
        for marker in STALE_SIGNING_POLICY_MARKERS:
            if marker in lowered:
                fail(f"stale mandatory-signing policy appears in active guidance: {relative} -> {marker}")

    current_contract = (
        "12 platform artifacts / 15 public files",
        f"Ghost-FTP-{version}-Setup-x64.exe",
        f"Ghost-FTP-{version}-Portable-x64.exe",
        f"Ghost-FTP-{version}-Linux-amd64.deb",
        f"Ghost-FTP-{version}-Linux-amd64.tar.gz",
        f"ghcr.io/bren-wp/ghost-ftp:{version}",
    )
    require_markers("README release contract", readme, current_contract)
    require_markers("documentation index release contract", index, current_contract[:1])

    installation = read("docs/INSTALLATION.md")
    require_markers(
        "installation release contract",
        installation,
        (
            f"Ghost FTP **{version} Stable** is the current published stable release",
            f"Ghost-FTP-{version}-Setup-x64.exe",
            f"Ghost-FTP-{version}-Linux-amd64.deb",
            f"Ghost-FTP-{version}-Linux-amd64.tar.gz",
            "Canonical release packages",
            "Supplemental distro-specific source/CI packages",
            "Linux-Debian-amd64.deb",
            "Linux-Ubuntu-amd64.deb",
            "Linux-Fedora-x86_64.rpm",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "not yet part of the canonical release allow-list",
            "x86-64 only",
            "12 platform artifacts / 15 public files",
        ),
    )

    linux_readme = read("linux/README.md")
    require_markers(
        "linux distro contract",
        linux_readme,
        (
            "linux/BUILD-DISTROS.sh",
            "Linux-Debian-amd64.deb",
            "Linux-Ubuntu-amd64.deb",
            "Linux-Fedora-x86_64.rpm",
            "Linux-Portable-amd64.tar.gz",
            "`amd64` | `x86_64`",
            "`arm64` | `aarch64`",
            "`i386` | `i686`",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "x86-64 only",
            "12 platform artifacts / 15 public files",
        ),
    )

    for required_path in (
        "linux/BUILD-DISTROS.sh",
        ".github/workflows/linux-distro-packages.yml",
        ".github/workflows/linux-distro-install.yml",
    ):
        if not (ROOT / required_path).is_file():
            fail(f"documented Linux distro implementation missing: {required_path}")

    parity = read("docs/PLATFORM-PARITY.md")
    require_markers(
        "platform parity documentation",
        parity,
        (
            "Windows and Linux platform parity",
            "SFTP password",
            "SFTP key passphrase",
            "24-language",
            "same typed `internal/api.Engine`",
            "Production Authenticode is optional.",
            "WINDOWS_AUTHENTICODE=unsigned",
            "linux/BUILD-DISTROS.sh",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "x86-64 only",
            "12 platform artifacts / 15 public files",
        ),
    )

    testing = read("docs/TESTING.md")
    require_markers(
        "testing documentation",
        testing,
        (
            f"Ghost FTP **{version} Stable**",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "linux/BUILD-DISTROS.sh",
            "Debian 13 amd64",
            "Ubuntu 26.04 LTS amd64",
            "Fedora 44 x86_64",
            "Native package-manager/runtime coverage is deliberately limited to x86-64.",
            "12 platform artifacts / 15 public files",
            "Exact-head and post-merge rule",
        ),
    )

    releases = read("docs/GITHUB-RELEASES.md")
    require_markers(
        "GitHub Releases documentation",
        releases,
        (
            f"Ghost FTP **{version} Stable** is the current published stable release",
            f"ghostftp-v{version}",
            f"Ghost-FTP-{version}-Linux-amd64.tar.gz",
            "12 platform artifacts",
            "15 public files",
            "release/ghostftp-vX.Y.Z",
            "workflow_dispatch",
        ),
    )

    verification = read("docs/RELEASE-VERIFICATION.md")
    require_markers(
        "release verification documentation",
        verification,
        (
            f"current maintained release is **{version} Stable**",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"Ghost-FTP-{version}-Linux-amd64.tar.gz",
            "12 platform artifacts",
            "15 public files",
            "truthful supported publication state",
            "does not create a self-signed production identity",
            "explicit unsigned metadata when no production certificate is configured",
        ),
    )

    packages = read("docs/PACKAGES.md")
    require_markers(
        "packages documentation",
        packages,
        (
            f"Ghost FTP **{version} Stable is published**",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "distribution bundle",
            "not a runtime container",
            "/ghostftp-release/",
            "SHA256.txt",
            "12 platform artifacts / 15 public files",
            "Authenticode verification **when a trusted production certificate is configured**",
            "WINDOWS_AUTHENTICODE=unsigned",
        ),
    )

    support = read("docs/SUPPORT.md")
    require_markers(
        "support documentation",
        support,
        (
            f"Ghost FTP **{version} Stable**",
            "inspect `WINDOWS_AUTHENTICODE` in `BUILD-METADATA.txt`",
            "official file is explicitly `unsigned`",
            "if metadata says `signed` and Windows signature verification fails",
        ),
    )

    signing = read("docs/SIGNING.md")
    require_markers(
        "signing documentation",
        signing,
        (
            "supports Windows Authenticode signing as an optional production hardening layer",
            "WINDOWS_AUTHENTICODE=signed",
            "WINDOWS_AUTHENTICODE=unsigned",
            "production workflow never creates its own long-lived publisher key",
        ),
    )

    print(f"DOCS_AUDIT=PASS ({version}; {len(files)} Markdown files)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PRE_1_0_CHANNEL=BETA")
    print("FIRST_STABLE_VERSION=1.0.0")
    print("STABLE_GITHUB_RELEASE_PRERELEASE=FALSE")
    print("STABLE_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=NO")
    print("TRUSTED_AUTHENTICODE_WHEN_CONFIGURED=VERIFIED")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("PUBLIC_PLATFORM_ARTIFACTS=12")
    print("PUBLIC_RELEASE_FILES=15")
    print("SUPPLEMENTAL_DISTRO_PACKAGING=DEBIAN,UBUNTU,FEDORA,PORTABLE")
    print("NATIVE_DISTRO_INSTALL_COVERAGE=DEBIAN13_AMD64,UBUNTU26.04_AMD64,FEDORA44_X86_64")
    print("SUPPLEMENTAL_DISTRO_RELEASE_ASSETS=NO")
    print("STABLE_GITHUB_PACKAGE=ghcr.io/bren-wp/ghost-ftp")
    return 0


if __name__ == "__main__":
    sys.exit(main())
