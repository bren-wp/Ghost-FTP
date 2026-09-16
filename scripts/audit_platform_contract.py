#!/usr/bin/env python3
"""Fail closed when unsupported or retired application surfaces enter the active product tree."""
from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
RETIRED_ROOTS = ("ios/", "GhostFTP WEB/")
RETIRED_WEB_ROOTS = ("web/", "web-ftp/", "webftp/", "pwa/", "ghostftp-web/")
RETIRED_SCRIPTS = {
    "scripts/audit_android.py",
    "scripts/audit_android_localization.py",
    "scripts/audit_ios.py",
    "scripts/package_android.py",
    "scripts/package_ios.py",
    "scripts/check_web_contract.py",
    "scripts/test_web_contract.py",
}
RETIRED_EXACT = {
    ".github/workflows/web.yml",
    "docs/WEB.md",
    "docs/prompts/GHOST-FTP-WEB-APP-PROMPT.md",
    "docs/prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md",
}
LINUX_PLATFORM_STUBS = {
    "internal/platform/delete_other.go",
    "internal/platform/language_other.go",
    "internal/platform/other.go",
    "internal/platform/prompt_other.go",
    "internal/platform/registry_other.go",
    "internal/platform/shortcut_other.go",
}
ANDROID_REQUIRED = {
    "android/app/build.gradle",
    "android/app/src/main/AndroidManifest.xml",
    "android/app/src/main/java/app/ghostftp/client/MainActivity.java",
    "android/app/src/main/java/app/ghostftp/client/FtpSession.java",
    ".github/workflows/android-apk.yml",
    "scripts/test_android_contract.py",
    "scripts/test_android_release_signing_contract.py",
}
MACOS_REQUIRED = {
    "macos/README.md",
    "macos/PARITY.md",
    "macos/BUILD.sh",
    "macos/Sources/GhostFTPApp/main.swift",
    ".github/workflows/macos-app.yml",
    "scripts/test_macos_windows_parity_contract.py",
}
BROWSER_REQUIRED = {
    "extensions/BRAND.json",
    "extensions/chrome/manifest.json",
    "extensions/edge/manifest.json",
    "extensions/firefox/manifest.json",
    "extensions/opera/manifest.json",
    "extensions/shared/core.js",
    "extensions/shared/core.test.mjs",
    "extensions/shared/popup.js",
    "cmd/ghostftp/launch_init.go",
    "cmd/ghostftp/launch_target.go",
    "cmd/ghostftp/launch_target_test.go",
    "cmd/ghostftp/uninstall_mode_windows.go",
    "cmd/installer/protocol_registration.go",
    "cmd/installer/protocol_registration_test.go",
    "cmd/installer/registry_snapshot.go",
    "cmd/installer/uninstall_registration_windows.go",
    "internal/desktop/startup_target.go",
    "internal/desktop/startup_target_test.go",
    "internal/desktop/startup_target_windows.go",
    "internal/desktop/startup_target_handoff_windows.go",
    "internal/desktop/startup_target_handoff_other.go",
    "internal/platform/browser_protocol_windows.go",
    "scripts/build_browser_extensions.py",
    "scripts/test_browser_extensions_contract.py",
    "scripts/test_browser_desktop_launch_contract.py",
    ".github/workflows/browser-extensions.yml",
}
LINUX_DISTRIBUTION_REQUIRED = {
    "linux/BUILD-DISTROS.sh",
    ".github/workflows/linux-distro-packages.yml",
    "scripts/test_linux_distro_packaging_contract.py",
}


def fail(message: str) -> None:
    raise SystemExit("PLATFORM_CONTRACT_AUDIT_FAILED: " + message)


def tracked_paths() -> list[str]:
    result = subprocess.run(
        ["git", "ls-files", "-z"], cwd=ROOT, check=True, stdout=subprocess.PIPE
    )
    return [item.decode("utf-8", "strict") for item in result.stdout.split(b"\0") if item]


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def main() -> int:
    paths = tracked_paths()
    path_set = set(paths)

    for path in paths:
        normalized = path.replace("\\", "/")
        if normalized.startswith(RETIRED_ROOTS + RETIRED_WEB_ROOTS):
            fail(f"retired application platform/surface is tracked: {path}")
        if normalized in RETIRED_SCRIPTS or normalized in RETIRED_EXACT:
            fail(f"retired platform tooling/surface is tracked: {path}")
        if normalized.startswith("ekstenzije/"):
            fail(f"retired non-English extension source root is tracked: {path}")
        if normalized.startswith(("linux/debian/", "linux/rpm/")):
            fail(f"retired architecture-specific Linux packaging source is tracked: {path}")

    for label, required in (
        ("Android", ANDROID_REQUIRED),
        ("macOS", MACOS_REQUIRED),
        ("browser helper", BROWSER_REQUIRED),
        ("Linux distribution", LINUX_DISTRIBUTION_REQUIRED),
    ):
        missing = sorted(required - path_set)
        if missing:
            fail(f"active {label} source contract is incomplete: " + ", ".join(missing))

    if "internal/platform/filemove_other.go" in path_set:
        fail("generic unsupported-OS filemove fallback must not be restored")

    for rel in sorted(LINUX_PLATFORM_STUBS):
        if rel not in path_set:
            fail(f"required Linux platform stub is not tracked: {rel}")
        lines = read(rel).splitlines()
        if not lines or lines[0] != "//go:build linux":
            fail(f"Linux platform stub has a broad/non-Linux build contract: {rel}")

    ci = read(".github/workflows/ci.yml")
    ci_lower = ci.lower()
    for marker in ("runs-on: macos", "ios/", "macos/", "ghostftp web/", "web/"):
        if marker in ci_lower:
            fail(f"ci.yml desktop production contract unexpectedly references: {marker}")
    for marker in ("windows:", "linux:"):
        if marker not in ci:
            fail(f"Windows/Linux desktop CI contract is incomplete: missing {marker}")

    linux_builder = read("linux/BUILD-DISTROS.sh")
    for marker in (
        "build_arch amd64 amd64",
        "build_arch arm64 arm64",
        "build_arch 386 i386",
        "Ghost-FTP-${VERSION}-Linux-${distro}-Installer.run",
        "Ghost-FTP-${VERSION}-Linux-${distro}-Portable",
        "for distro in Debian Ubuntu Fedora; do",
    ):
        if marker not in linux_builder:
            fail(f"Linux universal distribution contract is incomplete: missing {marker}")
    for retired in ("dpkg-deb", "rpmbuild"):
        if retired in linux_builder:
            fail(f"Linux public builder still contains retired architecture-specific packager: {retired}")

    release = read(".github/workflows/release.yml")
    release_lower = release.lower()
    for marker in ("runs-on: macos", "ios/", "macos/", "ghostftp web/", "web/"):
        if marker in release_lower:
            fail(f"release.yml public contract unexpectedly references unsupported public surface: {marker}")
    for marker in (
        "windows:",
        "linux:",
        "android:",
        "browser:",
        "Production signed Android APK",
        "Chrome Edge Firefox Opera release packages",
        "Ghost-FTP-${VERSION}-Android.apk",
        "Ghost-FTP-${VERSION}-Chrome-Extension.zip",
        "Ghost-FTP-${VERSION}-Opera-Extension.zip",
        "Ghost-FTP-${VERSION}-Linux-Debian-Installer.run",
        "Ghost-FTP-${VERSION}-Linux-Debian-Portable.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-Ubuntu-Installer.run",
        "Ghost-FTP-${VERSION}-Linux-Ubuntu-Portable.tar.gz",
        "Ghost-FTP-${VERSION}-Linux-Fedora-Installer.run",
        "Ghost-FTP-${VERSION}-Linux-Fedora-Portable.tar.gz",
        "PUBLIC_PLATFORM_ARTIFACTS=13",
        "PUBLIC_RELEASE_FILES=16",
    ):
        if marker not in release:
            fail(f"cross-platform public release contract is incomplete: missing {marker}")

    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")
    if tuple(int(part) for part in version.split(".")) < (0, 0, 1):
        fail("active product baseline must not precede 0.0.1")

    print(f"PLATFORM_CONTRACT_AUDIT=PASS ({version})")
    print("PUBLIC_RELEASE_PLATFORMS=WINDOWS,LINUX,ANDROID,BROWSER_HELPER")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS")
    print("ACTIVE_WEB_SURFACE=NONE")
    print("WINDOWS_PUBLIC_RELEASE_PACKAGES=SETUP,PORTABLE")
    print("LINUX_PUBLIC_RELEASE_PACKAGES=DEBIAN_INSTALLER,DEBIAN_PORTABLE,UBUNTU_INSTALLER,UBUNTU_PORTABLE,FEDORA_INSTALLER,FEDORA_PORTABLE")
    print("LINUX_BUNDLE_ARCHITECTURES=AMD64,ARM64,I386")
    print("ANDROID_SOURCE_SURFACE=ACTIVE")
    print("ANDROID_RELEASE_BUILD_AND_SIGNING_SMOKE=ACTIVE")
    print("ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED")
    print("ANDROID_SFTP_PUBLIC_SUPPORT=NO_STRICT_HOST_KEY_BOUNDARY")
    print("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX,OPERA")
    print("BROWSER_DESKTOP_HANDOFF=WINDOWS_SANITIZED_PROTOCOL")
    print("MACOS_SOURCE_SURFACE=ACTIVE")
    print("MACOS_PUBLIC_RELEASE_ARTIFACT=NO")
    print("RETIRED_APPLICATION_PLATFORMS=IOS")
    print("RETIRED_APPLICATION_SURFACES=PWA,WEB,WEB_FTP")
    print("LINUX_PLATFORM_STUBS=EXPLICIT")
    print("MINIMUM_PUBLIC_VERSION=0.0.1")
    print("VERSIONING_PLATFORM_INDEPENDENT=YES")
    return 0


if __name__ == "__main__":
    sys.exit(main())
