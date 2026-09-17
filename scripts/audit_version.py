#!/usr/bin/env python3
"""Verify canonical Ghost FTP 0.0.7 production identity across maintained source surfaces."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
GO_TOOLCHAIN = "1.27.1"
RETIRED_ROOTS = ("ios", "GhostFTP WEB", "ekstenzije", "web", "web-ftp")
CURRENT_LINE_DOCS = (
    "README.md",
    "CHANGELOG.md",
    "docs/README.md",
    "docs/GITHUB-RELEASES.md",
    "docs/INSTALLATION.md",
    "docs/PACKAGES.md",
    "docs/REFERENCE-UI.md",
    "docs/RELEASE-HISTORY.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/SETTINGS.md",
    "docs/SUPPORT.md",
    "docs/TESTING.md",
    "docs/VERSIONING.md",
)
RETIRED_PUBLIC_VERSION_PATTERNS = (
    re.compile(r"Ghost FTP(?:\s+\*\*)?\s*1\.\d+\.\d+"),
    re.compile(r"Ghost-FTP-1\.\d+\.\d+"),
    re.compile(r"ghostftp-v1\.\d+\.\d+"),
    re.compile(r"ghcr\.io/bren-wp/ghost-ftp:1\.\d+\.\d+"),
    re.compile(r"\bVERSION=1\.\d+\.\d+\b"),
    re.compile(r"\bTAG=ghostftp-v1\.\d+\.\d+\b"),
)


def fail(message: str) -> None:
    raise SystemExit("VERSION_AUDIT_FAILED: " + message)


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        fail(f"missing {path}")
    return target.read_text(encoding="utf-8")


def require(text: str, markers: tuple[str, ...], where: str) -> None:
    for marker in markers:
        if marker not in text:
            fail(f"{where} is missing version/platform binding: {marker}")


def forbid(text: str, markers: tuple[str, ...], where: str) -> None:
    for marker in markers:
        if marker in text:
            fail(f"{where} contains retired production marker: {marker}")


def main() -> int:
    version = read("VERSION").strip()
    if not VERSION_RE.fullmatch(version):
        fail(f"VERSION is not semantic: {version!r}")
    if version != "0.0.7":
        fail(f"active release candidate must remain 0.0.7, got {version!r}")

    if f"go {GO_TOOLCHAIN}" not in read("go.mod"):
        fail(f"go.mod must use Go {GO_TOOLCHAIN}")

    # Release builds still inject VERSION with -X. Source entry points also carry
    # the same current release identity so locally built user interfaces never
    # expose a development-channel label when linker metadata is absent.
    for rel in ("cmd/ghostftp/main.go", "cmd/installer/main.go", "cmd/windowsbootstrap/main.go"):
        text = read(rel)
        match = re.search(r'var\s+version\s*=\s*"([^"]+)"', text)
        if match is None:
            fail(f"{rel} is missing the build-time version binding")
        if match.group(1) != version:
            fail(f"{rel} fallback version {match.group(1)!r} does not match VERSION {version!r}")
        if '"dev"' in text:
            fail(f"{rel} exposes a development version marker")

    brand_version = read("internal/brand/version.go")
    require(brand_version, ('strings.TrimSpace(version)', 'return version'), "internal/brand/version.go")
    if 'return "dev"' in brand_version:
        fail("product display version must not expose a development fallback")
    if 'return version + " Beta"' in brand_version or 'strings.HasPrefix(version, "0.")' in brand_version:
        fail("product display version must not infer prerelease status from major version 0")

    readme = read("README.md")
    require(
        readme,
        (
            f"Current source version: **{version}**",
            "Release channel: **Current**",
            "Product status: **Current**",
            "Last actually published GitHub Release: **0.0.6**",
            f"ghostftp-v{version}",
            "Prerelease: **false**",
            "13 platform artifacts / 16 public files",
            f"Ghost-FTP-{version}-Linux-Debian-Installer.run",
            f"Ghost-FTP-{version}-Linux-Fedora-Portable.tar.gz",
            f"Ghost-FTP-{version}-Android.apk",
            f"Ghost-FTP-{version}-Opera-Extension.zip",
            "GHOSTFTP_ANDROID_CERT_SHA256",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "The retired website and Web FTP implementation are intentionally not part of this repository or product runtime.",
        ),
        "README.md",
    )

    if f"## {version}" not in read("CHANGELOG.md"):
        fail("CHANGELOG does not contain a section for VERSION")

    versioning = read("docs/VERSIONING.md")
    require(
        versioning,
        (
            f"Current source candidate: **{version}**",
            f"VERSION={version}",
            f"TAG=ghostftp-v{version}",
            "CHANNEL=Current",
            "PRERELEASE=false",
            f"ghcr.io/bren-wp/ghost-ftp:{version}",
            "major version `0` does not imply prerelease",
            "latest public version",
            "release-retention.yml",
            "required public-release trust boundary",
            "WINDOWS_AUTHENTICODE=signed",
            "Absence of the production Authenticode identity is a release failure.",
        ),
        "docs/VERSIONING.md",
    )

    for rel in CURRENT_LINE_DOCS:
        text = read(rel)
        for pattern in RETIRED_PUBLIC_VERSION_PATTERNS:
            match = pattern.search(text)
            if match:
                fail(f"active current-line documentation contains retired public identity {match.group(0)!r}: {rel}")

    windows_build = read("BUILD-WINDOWS.ps1")
    require(
        windows_build,
        (
            "Get-Content -LiteralPath $versionFile",
            "-X main.version=$version",
            "WINDOWS_PUBLIC_SETUP=UNIVERSAL_X86_X64_ARM64",
            "WINDOWS_PUBLIC_PORTABLE=UNIVERSAL_X86_X64_ARM64",
            "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
            "WINDOWS_PUBLIC_EXECUTABLES=2",
        ),
        "BUILD-WINDOWS.ps1",
    )

    linux_build = read("linux/BUILD-DISTROS.sh")
    require(
        linux_build,
        (
            "< VERSION",
            "-X main.version=${VERSION}",
            "build_arch amd64 amd64",
            "build_arch arm64 arm64",
            "build_arch 386 i386",
            "for distro in Debian Ubuntu Fedora; do",
            'Ghost-FTP-${VERSION}-Linux-${distro}-Installer.run',
            'Ghost-FTP-${VERSION}-Linux-${distro}-Portable',
            "ghostftp-uninstall",
        ),
        "linux/BUILD-DISTROS.sh",
    )
    forbid(linux_build, ("dpkg-deb", "rpmbuild"), "linux/BUILD-DISTROS.sh")

    android_build = read("android/app/build.gradle")
    require(
        android_build,
        (
            "rootProject.file('../VERSION').text.trim()",
            "versionCode ghostFtpVersionCode",
            "versionName ghostFtpVersion",
            "namespace 'app.ghostftp.client'",
            "applicationId 'app.ghostftp.client'",
            "applicationIdSuffix '.debug'",
        ),
        "android/app/build.gradle",
    )
    forbid(
        android_build,
        (
            "versionNameSuffix '-dev'",
            "packageGhostFtpApk",
            "Ghost-FTP-Android-dev.apk",
        ),
        "android/app/build.gradle",
    )

    android_workflow = read(".github/workflows/android-apk.yml")
    require(
        android_workflow,
        (
            ":app:testDebugUnitTest",
            ":app:lintRelease",
            ":app:assembleRelease",
            "app-release-unsigned.apk",
            '"$build_tools/apksigner" sign',
            '"$build_tools/apksigner" verify --verbose --print-certs',
            "ANDROID_RELEASE_SIGNING_PIPELINE_SMOKE_IDENTITY=EPHEMERAL_CI_ONLY",
        ),
        ".github/workflows/android-apk.yml",
    )
    forbid(
        android_workflow,
        (
            "ISOLATED_CI_ONLY",
            "Ghost-FTP-Android-dev.apk",
            "ghostftp-android-dev-apk",
            "Upload Android development APK",
        ),
        ".github/workflows/android-apk.yml",
    )

    macos_build = read("macos/BUILD.sh")
    require(
        macos_build,
        (
            'VERSION="$(tr -d \'\\r\\n\' < "$SCRIPT_DIR/../VERSION")"',
            "CFBundleShortVersionString",
            "CFBundleVersion",
            "app.ghostftp.client",
            'Ghost-FTP-${VERSION}-macOS.app.zip',
            "MACOS_SIGNING=adhoc-validation",
        ),
        "macos/BUILD.sh",
    )
    forbid(macos_build, ("adhoc-development", "MACOS_DEVELOPMENT_ARTIFACT"), "macos/BUILD.sh")

    extension_brand = read("extensions/BRAND.json")
    require(
        extension_brand,
        (
            '"product": "Ghost FTP"',
            '"extension": "Ghost FTP Connection Helper"',
            '"official_packages": ["chrome", "edge", "firefox", "opera"]',
        ),
        "extensions/BRAND.json",
    )
    for browser in ("chrome", "edge", "firefox", "opera"):
        manifest = read(f"extensions/{browser}/manifest.json")
        if f'"version": "{version}"' not in manifest:
            fail(f"{browser} extension manifest is not bound to VERSION {version}")

    for retired in RETIRED_ROOTS:
        if (ROOT / retired).exists():
            fail(f"retired source surface must be removed: {retired}/")
    if (ROOT / ".github/workflows/web.yml").exists():
        fail("retired website workflow must be removed")
    if (ROOT / "docs/WEB.md").exists():
        fail("retired Web FTP documentation must be removed")

    for workflow_rel in (".github/workflows/ci.yml", ".github/workflows/release.yml"):
        workflow = read(workflow_rel)
        if f"go-version: '{GO_TOOLCHAIN}'" not in workflow:
            fail(f"{workflow_rel} does not pin Go {GO_TOOLCHAIN}")
        require(workflow, ("windows:", "linux:", "bash linux/BUILD-DISTROS.sh"), workflow_rel)
        lowered = workflow.lower()
        for marker in ("ios/", "ghostftp web/", "runs-on: macos"):
            if marker in lowered:
                fail(f"{workflow_rel} references non-public application marker: {marker}")

    release_workflow = read(".github/workflows/release.yml")
    if re.search(r"(?m)^\s*default:\s*['\"]?\d+\.\d+\.\d+", release_workflow):
        fail("release workflow contains a hard-coded production version")
    require(
        release_workflow,
        (
            "manual='${{ inputs.version }}'",
            "source_version=\"$(tr -d '\\r\\n' < VERSION)\"",
            "RELEASE_TAG=ghostftp-v$version",
            "release_channel='current'",
            "release_title=\"Ghost FTP $version\"",
            "packages: write",
            "Publish verified bundle to GitHub Packages",
            "test \"$remote_prerelease\" = 'false'",
            "Require protected Authenticode identity",
            "state=signed",
            "test \"$WINDOWS_SIGNING_STATE\" = 'signed'",
            "Require protected Android production signing identity",
            "GHOSTFTP_ANDROID_KEYSTORE_BASE64",
            "GHOSTFTP_ANDROID_CERT_SHA256",
            "ANDROID_APK=production-signed",
            "BROWSER_EXTENSION_PACKAGES=Chrome,Edge,Firefox,Opera",
            "WINDOWS_SETUP=universal-x86-x64-arm64",
            "WINDOWS_PORTABLE=universal-x86-x64-arm64",
            "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
            "LINUX_DEBIAN_INSTALLER=universal-amd64-arm64-i386",
            "LINUX_DEBIAN_PORTABLE=universal-amd64-arm64-i386",
            "LINUX_UBUNTU_INSTALLER=universal-amd64-arm64-i386",
            "LINUX_UBUNTU_PORTABLE=universal-amd64-arm64-i386",
            "LINUX_FEDORA_INSTALLER=universal-amd64-arm64-i386",
            "LINUX_FEDORA_PORTABLE=universal-amd64-arm64-i386",
            "PUBLIC_PLATFORM_ARTIFACTS=13",
            "PUBLIC_RELEASE_FILES=16",
        ),
        ".github/workflows/release.yml",
    )
    forbid(release_workflow, ("state=unsigned", "keytool -genkeypair", "--prerelease"), ".github/workflows/release.yml")

    retention = read(".github/workflows/release-retention.yml")
    require(
        retention,
        (
            "Publish Ghost FTP",
            "test \"$release_prerelease\" = 'false'",
            "test \"$asset_count\" -eq 16",
            "gh release delete",
            "--cleanup-tag",
            "packages/container/ghost-ftp/versions",
            "GHOSTFTP_RELEASE_RETENTION=PASS",
            "GHOSTFTP_PACKAGE_RETENTION=PASS (current=$version)",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
        ),
        ".github/workflows/release-retention.yml",
    )

    release_audit = read("scripts/audit_release.py")
    require(
        release_audit,
        (
            "PUBLIC_RELEASE_CHANNEL=CURRENT",
            "CURRENT_RELEASE_PRERELEASE_FLAG=FALSE",
            "LATEST_ONLY_RELEASE_RETENTION=YES",
            "PUBLIC_PLATFORM_ARTIFACTS={PUBLIC_PLATFORM_ARTIFACTS}",
            "PUBLIC_RELEASE_FILES={PUBLIC_RELEASE_FILES}",
            "WINDOWS_SETUP=UNIVERSAL_X86_X64_ARM64",
            "LINUX_BUNDLE_ARCHITECTURES=AMD64,ARM64,I386",
            "ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED",
            "BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX,OPERA",
            "GHCR_CURRENT_BUNDLE=REQUIRED",
            "PUBLIC_WINDOWS_AUTHENTICODE=REQUIRED_AND_VERIFIED",
            "ANDROID_PRODUCTION_SIGNING_IDENTITY=REQUIRED_AND_VERIFIED",
        ),
        "scripts/audit_release.py",
    )

    print(f"VERSION_AUDIT=PASS ({version}; channel=current; product=current)")
    print(f"GO_TOOLCHAIN={GO_TOOLCHAIN}")
    print("PUBLIC_BRAND=Ghost FTP")
    print("LAST_PUBLISHED_GITHUB_RELEASE=0.0.6")
    print("NEXT_PUBLIC_RELEASE=0.0.7")
    print("PUBLIC_RELEASE_CHANNEL=CURRENT")
    print("CURRENT_RELEASE_PRERELEASE_FLAG=FALSE")
    print("PUBLIC_PLATFORM_ARTIFACTS=13")
    print("PUBLIC_RELEASE_FILES=16")
    print("WINDOWS_SETUP=UNIVERSAL_X86_X64_ARM64")
    print("LINUX_BUNDLE_ARCHITECTURES=AMD64,ARM64,I386")
    print("ANDROID_PUBLIC_RELEASE_ARTIFACT=YES_PRODUCTION_SIGNED")
    print("BROWSER_PUBLIC_RELEASE_PACKAGES=CHROME,EDGE,FIREFOX,OPERA")
    print("RETIRED_WEB_SURFACES=ABSENT")
    return 0


if __name__ == "__main__":
    sys.exit(main())
