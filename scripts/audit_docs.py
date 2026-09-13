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
MARKDOWN_IMAGE_RE = re.compile(r"!\[[^\]]*\]\(([^)\n]+)\)")
HTML_LINK_RE = re.compile(r"\b(?:href|src)\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)
HTML_IMAGE_RE = re.compile(r"<img\b[^>]*\bsrc\s*=\s*[\"']([^\"']+)[\"']", re.IGNORECASE)
CURRENT_RELEASE_RE = re.compile(r"\*\*Current Ghost FTP release:\s*(\d+\.\d+\.\d+)\*\*")
PRODUCT_VERSION_RE = re.compile(r"Ghost FTP\s+\*\*(\d+\.\d+\.\d+)\*\*", re.IGNORECASE)
VERSION_SCOPE_RE = re.compile(r"(?:Version scope|Current source line|Current root source line):\s*\*\*?(\d+\.\d+\.\d+)\*\*?", re.IGNORECASE)
IGNORED_PREFIXES = ("http://", "https://", "mailto:", "data:", "//", "#")
REMOTE_MEDIA_PREFIXES = ("http://", "https://", "data:", "//")
RETIRED_ACTIVE_MARKERS = ("ios/", "ghostftp web/", "web companion", "pwa")

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
    "android/README.md",
    "android/UI-UX.md",
    "android/TRANSFER-PROGRESS.md",
    "macos/README.md",
    "macos/PARITY.md",
    "ekstenzije/README.md",
    "ekstenzije/PRIVACY.md",
    "scripts/README.md",
)

SIGNING_CONTRACT_DOCS = (
    "README.md",
    "docs/SIGNING.md",
    "docs/GITHUB-RELEASES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/PACKAGES.md",
    "docs/SUPPORT.md",
    "docs/PLATFORM-PARITY.md",
    "docs/ARCHITECTURE.md",
    "docs/INSTALLATION.md",
    "docs/TESTING.md",
)

STALE_OFFICIAL_SIGNING_MARKERS = (
    "production authenticode is optional",
    "windows_authenticode=unsigned",
    "official file is explicitly `unsigned`",
    "publication remains truthfully unsigned",
    "explicit unsigned metadata when no production certificate is configured",
    "supports windows authenticode signing as an optional production hardening layer",
    "unsigned publication is never relabeled as signed",
    "exercises optional authenticode policy",
)

STALE_RELEASE_SHAPES = (
    "12 platform artifacts / 15 public files",
    "9 platform artifacts / 12 public files",
    "6 platform artifacts / 9 public files",
)

VISUAL_ASSETS = (
    "build/icon.png",
    "docs/images/ghost-ftp-main-workspace.png",
    "docs/images/ghost-ftp-site-manager.png",
    "docs/images/ghost-ftp-settings.png",
    "docs/images/ghost-ftp-about.png",
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


def check_media(source: Path, raw: str) -> None:
    destination = clean_destination(raw)
    if not destination:
        fail(f"empty documentation media source: {source.relative_to(ROOT)}")
    if destination.lower().startswith(REMOTE_MEDIA_PREFIXES):
        fail(f"remote/data documentation media is blocked: {source.relative_to(ROOT)} -> {raw}")
    check_link(source, raw)


def require_markers(label: str, text: str, markers: tuple[str, ...]) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{label} missing marker: {marker}")


def check_active_version(relative: str, text: str, version: str) -> None:
    for pattern in (PRODUCT_VERSION_RE, VERSION_SCOPE_RE):
        for match in pattern.finditer(text):
            found = match.group(1)
            if found != version:
                fail(f"stale active product version in {relative}: {found} (expected {version})")

    lowered = text.lower()
    for marker in STALE_RELEASE_SHAPES:
        if marker in lowered:
            fail(f"stale public release shape in {relative}: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    if tuple(int(part) for part in version.split(".")) < (0, 0, 1):
        fail("documentation public version must be 0.0.1 or newer")
    prerelease = "false"

    files = sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts)
    if not files or not INDEX.is_file():
        fail("documentation set is incomplete")

    for path in files:
        text = path.read_text(encoding="utf-8")
        for match in MARKDOWN_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in HTML_LINK_RE.finditer(text):
            check_link(path, match.group(1))
        for match in MARKDOWN_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))
        for match in HTML_IMAGE_RE.finditer(text):
            check_media(path, match.group(1))
        for match in CURRENT_RELEASE_RE.finditer(text):
            if match.group(1) != version:
                fail(f"stale release marker in {path.relative_to(ROOT)}: {match.group(1)}")

    for relative in VISUAL_ASSETS:
        path = ROOT / relative
        if not path.is_file() or path.stat().st_size <= 0:
            fail(f"missing maintained local documentation visual: {relative}")

    for relative in ACTIVE_DOCS:
        text = read(relative)
        lowered = text.lower()
        check_active_version(relative, text, version)
        for marker in RETIRED_ACTIVE_MARKERS:
            if marker in lowered:
                fail(f"retired application surface appears in active guidance: {relative} -> {marker}")

    for relative in SIGNING_CONTRACT_DOCS:
        lowered = read(relative).lower()
        for marker in STALE_OFFICIAL_SIGNING_MARKERS:
            if marker in lowered:
                fail(f"stale official unsigned-release policy appears in {relative}: {marker}")

    readme = read("README.md")
    index = read("docs/README.md")
    if not readme.startswith("# Ghost FTP\n"):
        fail("README public title must be Ghost FTP")
    if not index.startswith("# Ghost FTP documentation\n"):
        fail("documentation index title is invalid")

    require_markers(
        "README product identity",
        readme,
        (
            f"Current Ghost FTP version: **{version}**",
            "Development status: **Active**",
            "Release channel: **Current**",
            "Public release platforms: **Windows and Linux**",
            "Active native source platforms: **Windows, Linux, Android and macOS**",
            "24 selectable desktop languages",
            "FTP", "FTPS", "SFTP",
            f"ghostftp-v{version}", f"prerelease={prerelease}",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
        ),
    )
    require_markers(
        "README marketing/visual contract",
        readme,
        (
            "## ⚡ Why Ghost FTP",
            "## 🖥️ See the product",
            "## 🔐 Security is a product boundary",
            "## 🛡️ Privacy without a hidden backend",
            'src="build/icon.png"',
            "docs/images/ghost-ftp-main-workspace.png",
            "docs/images/ghost-ftp-site-manager.png",
            "docs/images/ghost-ftp-settings.png",
            "docs/images/ghost-ftp-about.png",
            "repository-local assets",
        ),
    )
    require_markers(
        "documentation index current release",
        index,
        (
            f"**Current Ghost FTP release: {version}**",
            "Development status: **Active**",
            "Release channel: **Current**",
            f"PRERELEASE={prerelease}",
            "latest release only",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            'src="../build/icon.png"',
            "images/ghost-ftp-main-workspace.png",
            "repository-local",
        ),
    )

    release_contract = (
        "14 platform artifacts / 17 public files",
        f"Ghost-FTP-{version}-Setup.exe",
        f"Ghost-FTP-{version}-Portable.exe",
        f"Ghost-FTP-{version}-Linux-Debian-amd64.deb",
        f"Ghost-FTP-{version}-Linux-Ubuntu-amd64.deb",
        f"Ghost-FTP-{version}-Linux-Fedora-x86_64.rpm",
        f"Ghost-FTP-{version}-Linux-Portable-amd64.tar.gz",
    )
    require_markers("README release contract", readme, release_contract)
    require_markers("documentation index release contract", index, (release_contract[0],))

    installation = read("docs/INSTALLATION.md")
    require_markers(
        "installation release contract",
        installation,
        (
            f"Ghost FTP **{version}** is the current published release",
            f"Ghost-FTP-{version}-Setup.exe",
            f"Ghost-FTP-{version}-Portable.exe",
            "Canonical release packages",
            "Debian 13 amd64", "Ubuntu 26.04 LTS amd64", "Fedora 44 x86_64",
            "14 platform artifacts / 17 public files",
            "Official Windows publication requires trusted Authenticode.",
            "WINDOWS_AUTHENTICODE=signed",
            "macOS development app",
        ),
    )

    linux_readme = read("linux/README.md")
    require_markers(
        "linux distro contract",
        linux_readme,
        (
            f"Ghost FTP **{version}** is the current public release line",
            f"Canonical {version} release artifacts",
            "linux/BUILD-DISTROS.sh",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "Debian 13 amd64", "Ubuntu 26.04 LTS amd64", "Fedora 44 x86_64",
            "14 platform artifacts / 17 public files",
            "distro-specific artifacts are no longer supplemental",
        ),
    )

    architecture = read("docs/ARCHITECTURE.md")
    require_markers(
        "architecture documentation",
        architecture,
        (
            f"Ghost FTP **{version}**",
            "Android and macOS development/source clients",
            "### macOS client",
            "WINDOWS_AUTHENTICODE=signed",
            "14 platform artifacts / 17 public files",
            "publicly releases Windows and Linux",
        ),
    )

    parity = read("docs/PLATFORM-PARITY.md")
    require_markers(
        "platform parity documentation",
        parity,
        (
            "Windows and Linux platform parity",
            f"Ghost FTP **{version}**",
            "same typed `internal/api.Engine`",
            "macOS source parity boundary",
            "Official public Windows publication requires trusted Authenticode.",
            "WINDOWS_AUTHENTICODE=signed",
            "14 platform artifacts / 17 public files",
        ),
    )

    signing = read("docs/SIGNING.md")
    require_markers(
        "signing documentation",
        signing,
        (
            f"Ghost FTP **{version}**",
            "Official Windows publication is **signed-only**.",
            "GHOSTFTP_SIGNING_PFX_BASE64",
            "GHOSTFTP_SIGNING_PASSWORD",
            "WINDOWS_AUTHENTICODE=signed",
            "no supported `state=unsigned` continuation path",
            "production workflow never creates its own long-lived publisher key",
            "## macOS development signing",
            "## macOS production signing and notarization",
            "Developer ID Application",
            "macos/SIGN_AND_NOTARIZE.sh",
            ".github/workflows/macos-production.yml",
            "14 platform artifacts / 17 public files",
        ),
    )

    testing = read("docs/TESTING.md")
    require_markers(
        "testing documentation",
        testing,
        (
            f"Ghost FTP **{version}**",
            "Bandwidth regression contract",
            ".github/workflows/android-apk.yml",
            "authentication-error redaction",
            ".github/workflows/macos-app.yml",
            "Ghost FTP macOS Development App",
            ".github/workflows/linux-distro-packages.yml",
            ".github/workflows/linux-distro-install.yml",
            "Exact-head and post-merge rule",
            "WINDOWS_SIGNING_STATE=signed",
            "14 platform artifacts / 17 public files",
        ),
    )

    android_readme = read("android/README.md")
    require_markers(
        "Android source documentation",
        android_readme,
        (
            "Android is an active source-development platform",
            "repository root `VERSION`",
            "development/debug-signed APK",
            "FTP and explicit FTPS Quick Connect",
            "SFTP is intentionally not exposed",
            "Storage Access Framework",
            "not a production-signed public Android release",
        ),
    )

    android_ui = read("android/UI-UX.md")
    require_markers(
        "Android UI/UX documentation",
        android_ui,
        (
            "Ghost FTP Android is a mobile client",
            "Files", "Sites", "Bookmarks", "Transfers", "Settings", "About",
            "FTP and explicit FTPS",
            "BuildConfig.VERSION_NAME",
            "## SFTP status",
            "does not offer an SFTP connection route",
        ),
    )

    android_progress = read("android/TRANSFER-PROGRESS.md")
    require_markers(
        "Android transfer progress documentation",
        android_progress,
        (
            "bytes that actually pass through the active upload/download stream",
            "does not invent a total size",
            "staged remote commit contract",
            "generation token",
            "emit no telemetry",
        ),
    )

    macos_readme = read("macos/README.md")
    require_markers(
        "macOS source documentation",
        macos_readme,
        (
            "same product and shared `internal/api.Engine`",
            "The macOS action inventory in `PARITY.md` is complete.",
            "universal Intel + Apple Silicon development app",
            "development artifact is deliberately ad-hoc signed",
            "Developer ID Application",
            "Apple reports an accepted notarization",
            ".github/workflows/macos-production.yml",
            "deliberately does **not** modify or upload to an existing public GitHub Release",
        ),
    )

    macos_parity = read("macos/PARITY.md")
    require_markers(
        "macOS parity documentation",
        macos_parity,
        (
            "same typed `internal/api.Engine`",
            "Root `VERSION` is the only product version source of truth.",
            "[x] Connect", "[x] Remote Edit", "[x] Move Bottom",
            "Public distribution is a separate release gate",
            "Apple Developer signing/notarization",
        ),
    )

    browser_helper = read("ekstenzije/README.md")
    require_markers(
        "browser helper documentation",
        browser_helper,
        (
            "Ghost FTP Connection Helper",
            "No telemetry.",
            "does not store",
            "does not read the active tab",
            "does not connect to your FTP, FTPS, or SFTP server",
            "does not launch the desktop client directly",
            "does not currently expose a supported browser-to-desktop URI or native-messaging contract",
        ),
    )

    reference_ui = read("docs/REFERENCE-UI.md")
    require_markers(
        "reference UI visual contract",
        reference_ui,
        (
            "images/ghost-ftp-main-workspace.png",
            "images/ghost-ftp-site-manager.png",
            "images/ghost-ftp-settings.png",
            "images/ghost-ftp-about.png",
            "Mockups, image-generation output and manually composed approximations are not accepted",
            "15 runtime images",
            "macOS native development frontend",
        ),
    )

    privacy = read("docs/PRIVACY.md")
    require_markers(
        "privacy documentation media contract",
        privacy,
        (
            f"Ghost FTP **{version}**",
            "Documentation media is repository-local.",
            "remote badge images", "tracking pixels", "remote icon resources", "remote webfonts",
            "browser connection helper",
        ),
    )

    releases = read("docs/GITHUB-RELEASES.md")
    require_markers(
        "GitHub Releases documentation",
        releases,
        (
            f"Ghost FTP **{version}** is the current published release",
            f"ghostftp-v{version}",
            "Prerelease: false",
            "14 platform artifacts / 17 public files",
            "only the latest public Ghost FTP version remains",
            "Official Windows publication requires a protected trusted Authenticode identity.",
            "WINDOWS_AUTHENTICODE=signed",
            "no supported unsigned-publication fallback",
        ),
    )

    verification = read("docs/RELEASE-VERIFICATION.md")
    require_markers(
        "release verification documentation",
        verification,
        (
            f"current maintained release is **{version}**",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            f"PRERELEASE={prerelease}",
            "14 platform artifacts / 17 public files",
            "Official Windows publication requires trusted Authenticode.",
            "WINDOWS_AUTHENTICODE=signed",
            "Local development and ordinary CI Windows builds may be unsigned.",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
        ),
    )

    packages = read("docs/PACKAGES.md")
    require_markers(
        "packages documentation",
        packages,
        (
            f"Ghost FTP **{version}**",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "distribution bundle",
            "not a runtime container",
            "/ghostftp-release/",
            "SHA256.txt",
            "14 platform artifacts / 17 public files",
            "WINDOWS_AUTHENTICODE=signed",
        ),
    )

    support = read("docs/SUPPORT.md")
    require_markers(
        "support documentation",
        support,
        (
            f"Ghost FTP **{version}**",
            "Official public Windows artifacts require trusted Authenticode",
            "WINDOWS_AUTHENTICODE=signed",
            "release-integrity issue",
            "Unsigned local/development or ordinary CI builds are allowed",
        ),
    )

    history = read("docs/RELEASE-HISTORY.md")
    require_markers(
        "release history",
        history,
        (f"## {version}", "latest public Ghost FTP version", "release-retention.yml", "14 platform artifacts / 17 public files"),
    )

    transition = read("docs/PACKAGING-TRANSITION.md")
    require_markers(
        "packaging transition record",
        transition,
        ("0.0.3 source candidate", "historical 0.0.2 tag/release is not rewritten", "PUBLIC_PLATFORM_ARTIFACTS=14", "PUBLIC_RELEASE_FILES=17"),
    )

    ui_workflow = read(".github/workflows/ui-screenshots.yml")
    require_markers(
        "authentic UI immutable evidence workflow",
        ui_workflow,
        (
            "permissions:\n  contents: read",
            "Verify exact-head cross-platform evidence bundle",
            "AUTHENTIC_UI_SOURCE_SHA=",
            "AUTHENTIC_UI_EVIDENCE=VERIFIED",
            "ghostftp-authentic-ui-verified-bundle",
            "scripts/assemble_ui_evidence.py",
        ),
    )
    for forbidden in (
        "contents: write", "git push", "git commit", "github-actions[bot]",
        "AUTHENTIC_UI_SCREENSHOTS=PERSISTED", "Persist authentic screenshots in repository",
    ):
        if forbidden in ui_workflow:
            fail(f"authentic UI workflow must not mutate the tested PR head: {forbidden}")

    print(f"DOCS_AUDIT=PASS ({version}; channel=current; {len(files)} Markdown files)")
    print("PUBLIC_BRAND=Ghost FTP")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("BROWSER_CONNECTION_HELPER=LOCAL_ONLY_NO_DESKTOP_HANDOFF")
    print("PUBLIC_RELEASE_CHANNEL=CURRENT")
    print("CURRENT_RELEASE_PRERELEASE_FLAG=FALSE")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("LATEST_ONLY_RELEASE_RETENTION=YES")
    print("CURRENT_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=YES")
    print("PUBLIC_WINDOWS_AUTHENTICODE=REQUIRED_AND_VERIFIED")
    print("DEVELOPMENT_WINDOWS_BUILDS_MAY_BE_UNSIGNED=YES")
    print("MACOS_DEVELOPMENT_BUILD_IS_NOT_NOTARIZATION_PROOF=YES")
    print("ANDROID_DEVELOPMENT_APK_IS_NOT_PUBLIC_RELEASE=YES")
    print("SELF_SIGNED_PRODUCTION_IDENTITY=BLOCKED")
    print("PUBLIC_PLATFORM_ARTIFACTS=14")
    print("PUBLIC_RELEASE_FILES=17")
    print("CANONICAL_DISTRO_PACKAGING=DEBIAN,UBUNTU,FEDORA,PORTABLE")
    print("NATIVE_DISTRO_INSTALL_COVERAGE=DEBIAN13_AMD64,UBUNTU26.04_AMD64,FEDORA44_X86_64")
    print("DOCS_LOCAL_VISUALS=PASS")
    return 0


if __name__ == "__main__":
    sys.exit(main())
