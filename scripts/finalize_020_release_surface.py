#!/usr/bin/env python3
from __future__ import annotations

import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel: str, text: str) -> None:
    (ROOT / rel).write_text(text, encoding="utf-8", newline="\n")


def replace_once(rel: str, old: str, new: str) -> None:
    text = read(rel)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"expected one match in {rel}, found {count}: {old[:80]!r}")
    write(rel, text.replace(old, new, 1))


# Release only the actual Windows/Linux desktop product. NuGet was a duplicate
# publication surface and is not part of the application artifact contract.
replace_once(".github/workflows/release.yml", "  packages: write\n", "")
replace_once(
    ".github/workflows/release.yml",
    "          python scripts/audit_web.py\n",
    "          python scripts/audit_desktop_surface.py\n",
)
replace_once(
    ".github/workflows/release.yml",
    "      - name: Assemble public release and NuGet package\n",
    "      - name: Assemble public Windows and Linux release\n",
)
replace_once(
    ".github/workflows/release.yml",
    "          mkdir -p release packages-out\n",
    "          mkdir -p release\n",
)
replace_once(
    ".github/workflows/release.yml",
    '''          python scripts/package_nuget.py \\
            --x64 "staging/windows/Ghost-FTP-${VERSION}-Portable-x64.exe" \\
            --x86 "staging/windows/Ghost-FTP-${VERSION}-Portable-x86.exe" \\
            --output-dir packages-out
          test -s "packages-out/GhostFTP.${VERSION}.nupkg"

''',
    "",
)
replace_once(
    ".github/workflows/release.yml",
    "      - name: Publish GitHub Release and GhostFTP package\n",
    "      - name: Publish verified GitHub Release\n",
)
replace_once(
    ".github/workflows/release.yml",
    "          command -v dotnet >/dev/null || { echo 'dotnet CLI is required for NuGet publication' >&2; exit 1; }\n\n",
    "",
)
nuget_publish = '''          dotnet nuget push "packages-out/GhostFTP.${VERSION}.nupkg" \\
            --api-key "$GH_TOKEN" \\
            --source "https://nuget.pkg.github.com/${GITHUB_REPOSITORY_OWNER}/index.json" \\
            --skip-duplicate

          package_versions="$(gh api "/users/${GITHUB_REPOSITORY_OWNER}/packages/nuget/GhostFTP/versions" --jq '.[].name')"
          grep -Fx "$VERSION" <<< "$package_versions"
          echo "GITHUB_PACKAGE_READBACK=PASS (GhostFTP $VERSION; channel=$RELEASE_CHANNEL)"

'''
replace_once(".github/workflows/release.yml", nuget_publish, "")
replace_once(
    ".github/workflows/release.yml",
    '''          path: |
            release/*
            packages-out/GhostFTP.*.nupkg
''',
    '''          path: release/*
''',
)

# Rewrite release audit around one publication surface: immutable GitHub Release
# with 9 platform artifacts plus notes/metadata/checksums = 12 public files.
write(
    "scripts/audit_release.py",
    '''#!/usr/bin/env python3
"""Fail-closed validation of the Ghost FTP Windows/Linux release contract."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("RELEASE_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def require(rel: str, *markers: str) -> str:
    text = read(rel)
    for marker in markers:
        if marker not in text:
            fail(f"{rel} is missing required marker: {marker}")
    return text


def run(rel: str) -> None:
    try:
        subprocess.run([sys.executable, str(ROOT / rel)], cwd=ROOT, check=True)
    except subprocess.CalledProcessError as exc:
        fail(f"{rel} failed with exit code {exc.returncode}")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\\d+\\.\\d+\\.\\d+", version):
        fail(f"invalid VERSION: {version!r}")
    parts = tuple(int(part) for part in version.split("."))
    if parts < (0, 1, 0):
        fail("active release baseline must not precede 0.1.0")

    run("scripts/audit_brand_hardcut.py")
    run("scripts/audit_repository.py")
    run("scripts/audit_platform_contract.py")
    run("scripts/audit_desktop_surface.py")

    workflow = require(
        ".github/workflows/release.yml",
        "name: Publish Ghost FTP",
        "contents: write",
        "needs: [quality, windows, linux]",
        "RELEASE_TAG=ghostftp-v$version",
        "release_title=\\\"Ghost FTP $version Beta\\\"",
        "release_channel='beta'",
        "prerelease_args+=(--prerelease)",
        "GHOSTFTP_SIGNING_PFX_BASE64",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
        "WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}",
        "Stable Windows releases require a configured trusted Authenticode identity.",
        "python scripts/audit_platform_contract.py",
        "python scripts/audit_desktop_surface.py",
        "Ghost-FTP-${VERSION}-Portable-x64.exe",
        "Ghost-FTP-${VERSION}-Portable-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x64.exe",
        "Ghost-FTP-${VERSION}-Setup-x86.exe",
        "Ghost-FTP-${VERSION}-Setup-x32.exe",
        "Ghost-FTP-${VERSION}-Linux-amd64.deb",
        "Ghost-FTP-${VERSION}-Linux-arm64.deb",
        "Ghost-FTP-${VERSION}-Linux-i386.deb",
        "Ghost-FTP-${VERSION}-Linux-multiarch.zip",
        "PUBLIC_PLATFORM_ARTIFACTS=9",
        "PUBLIC_RELEASE_FILES=12",
        "main moved from release commit",
        "refusing to rewrite it",
        "RELEASE_ASSET_READBACK=PASS",
    )
    lowered = workflow.lower()
    for forbidden in (
        "packages: write", "package_nuget.py", "dotnet nuget", "nuget.pkg.github.com",
        "package_web.py", "audit_web.py", "android/", "ios/", "macos/", "runs-on: macos",
    ):
        if forbidden in lowered:
            fail(f"release workflow contains retired publication/platform marker: {forbidden}")

    require(
        ".github/workflows/ci.yml",
        "name: Ghost FTP CI",
        "python scripts/audit_platform_contract.py",
        "python scripts/audit_desktop_surface.py",
        "go test -race ./...",
        "Windows x64 and x86 production build",
        "Linux amd64 arm64 i386 production build",
        "Authenticode private-key pipeline smoke test",
        "New-DevCodeSigningCertificate.ps1",
        "Sign-WindowsArtifacts.ps1",
    )
    require(
        "BUILD-WINDOWS.ps1",
        "function Build-GhostFTPArchitecture",
        "function Sign-WindowsTarget",
        "GHOSTFTP_SIGNING_PFX_PATH",
        "GHOSTFTP_SIGNING_PASSWORD",
        "GHOSTFTP_SIGNING_TIMESTAMP_URL",
        '"Ghost-FTP-$version-Portable-$Label.exe"',
        '"Ghost-FTP-$version-Setup-$Label.exe"',
        "Sign-WindowsTarget -Path $portable",
        "scripts/make_payload.py",
        "Sign-WindowsTarget -Path $setup",
        "scripts/verify_release.py",
    )
    require(
        "cmd/installer/main.go",
        'uninstallKey = `Software\\\\Microsoft\\\\Windows\\\\CurrentVersion\\\\Uninstall\\\\GhostFTP`',
        'appPathsKey  = `Software\\\\Microsoft\\\\Windows\\\\CurrentVersion\\\\App Paths\\\\GhostFTP.exe`',
        'appPath := filepath.Join(dir, "GhostFTP.exe")',
        "registerIntegratedUninstall(appPath, version)",
    )
    require(
        "cmd/installer/uninstall_registration_windows.go",
        '"UninstallString"', '"QuietUninstallString"', '"DisplayVersion"', '"NoModify"', '"NoRepair"',
    )
    require("scripts/make_payload.py", "PAYLOAD_SCHEMA = 2", 'add(zf, args.app, "GhostFTP.exe")')
    require("linux/BUILD.sh", '"$root/usr/bin/ghostftp"', "Ghost-FTP-${VERSION}-Linux-${debarch}.deb")
    require("linux/debian/control.in", "Package: ghost-ftp")

    for retired in ("android", "ios", "macos", "GhostFTP WEB"):
        if (ROOT / retired).exists():
            fail(f"retired application directory exists: {retired}/")
    for retired_file in (
        "scripts/package_nuget.py", "scripts/package_web.py", "scripts/test_package_web.py", "scripts/audit_web.py",
    ):
        if (ROOT / retired_file).exists():
            fail(f"retired release/tooling file exists: {retired_file}")

    channel = "beta" if parts[0] == 0 else "stable"
    print(f"RELEASE_AUDIT=PASS ({version}; channel={channel})")
    print("PUBLIC_BRAND=Ghost FTP")
    print("TECHNICAL_IDENTITY=GhostFTP")
    print("RELEASE_TAG_NAMESPACE=ghostftp-vX.Y.Z")
    print("ACTIVE_APPLICATION_PLATFORMS=WINDOWS,LINUX")
    print("PUBLICATION_SURFACE=GITHUB_RELEASE")
    print("PRE_1_0_CHANNEL=BETA")
    print("FIRST_STABLE_VERSION=1.0.0")
    print("AUTHENTICODE_PRIVATE_KEY_IN_REPOSITORY=BLOCKED")
    print("STABLE_WINDOWS_RELEASE_REQUIRES_TRUSTED_AUTHENTICODE=YES")
    print("PUBLIC_PLATFORM_ARTIFACTS=9")
    print("PUBLIC_RELEASE_FILES=12")
    print("WINDOWS_PORTABLE=x64,x86")
    print("WINDOWS_X32_ALIAS_OF_X86=REQUIRED")
    print("LINUX_DEB=amd64,arm64,i386")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
''',
)

# Remove stale Web/NuGet packaging and its tests. The desktop-only audit remains
# the fail-closed guard against those surfaces returning.
for rel in (
    "scripts/package_nuget.py",
    "scripts/package_web.py",
    "scripts/test_package_web.py",
    "scripts/audit_web.py",
    "docs/SHARED-HOSTING.md",
):
    path = ROOT / rel
    if path.exists():
        path.unlink()

# Remove any Markdown link to the retired shared-hosting guide to keep docs link-clean.
for path in ROOT.rglob("*.md"):
    if ".git" in path.parts:
        continue
    lines = path.read_text(encoding="utf-8").splitlines()
    filtered = [line for line in lines if "SHARED-HOSTING.md" not in line]
    if filtered != lines:
        path.write_text("\n".join(filtered) + "\n", encoding="utf-8", newline="\n")

# Update scripts index to describe only maintained desktop tooling.
write(
    "scripts/README.md",
    '''# Ghost FTP build and audit scripts

This directory contains the maintained Windows/Linux packaging, security, privacy and verification tooling used by Ghost FTP.

## Canonical release path

GitHub Releases are assembled by `.github/workflows/release.yml`. There is one public release publication surface: the immutable GitHub Release for `ghostftp-vX.Y.Z`. Keeping one path prevents version, tag and artifact drift.

Important maintained tools include:

- `release_notes.py` — generates release notes from `CHANGELOG.md`.
- `make_payload.py` — creates the verified Windows Setup payload.
- `verify_release.py` / `verify_bundle.py` — verify release/bundle structure and integrity.
- `audit_desktop_surface.py` — rejects retired Web/PWA application surfaces.
- `audit_platform_contract.py` — enforces Windows/Linux-only application targets.
- `audit_security.py` — security-policy regression checks.
- `audit_privacy.py` — privacy/telemetry regression checks.
- `audit_dependencies.py` — rejects unexpected dependency and tracking SDK drift.
- platform build/package helpers referenced by CI or documented local workflows.

NuGet, Web/PWA and mobile application packaging are not part of the maintained Ghost FTP desktop release contract.

## Release identity

Ghost FTP uses `VERSION` plus namespaced tags:

```text
ghostftp-vX.Y.Z
```

Historical GhostFTP tags are immutable and are not reused.

## Build invariants

Production workflows disable Go telemetry and use controlled dependency resolution. Windows and Linux artifacts are assembled only after all platform jobs pass, then checksummed in `SHA256.txt`.

Do not add another script or package registry that independently publishes Ghost FTP. New release logic belongs in the canonical GitHub Release workflow and must preserve tag immutability, checksum generation, read-back verification and explicit signing status.

## Security

Never embed production signing keys, tokens, FTP credentials, recovery secrets or private certificates in scripts. Signing credentials must remain outside the public repository.
''',
)

# Convert maintenance tests from the retired Web contract to the desktop-only contract.
maintenance = read("scripts/test_maintenance.py")
maintenance = maintenance.replace(
    '        self.assertEqual(version, read("GhostFTP WEB/VERSION").strip())\n',
    '        self.assertFalse((ROOT / "GhostFTP WEB").exists())\n',
)
start = maintenance.find("    def test_web_version_brand_and_fail_closed_boundaries(self) -> None:\n")
if start >= 0:
    end = maintenance.find("\n\n\nif __name__ == \"__main__\":", start)
    if end < 0:
        raise SystemExit("unable to locate retired Web maintenance-test boundary")
    replacement = '''    def test_retired_web_and_package_surfaces_remain_absent(self) -> None:
        self.assertFalse((ROOT / "GhostFTP WEB").exists())
        for rel in (
            "scripts/package_web.py",
            "scripts/test_package_web.py",
            "scripts/package_nuget.py",
            "scripts/audit_web.py",
        ):
            self.assertFalse((ROOT / rel).exists(), f"retired tooling unexpectedly exists: {rel}")
        release = read(".github/workflows/release.yml").lower()
        self.assertNotIn("nuget", release)
        self.assertNotIn("package_web.py", release)
        self.assertIn("audit_desktop_surface.py", release)
'''
    maintenance = maintenance[:start] + replacement + maintenance[end:]
write("scripts/test_maintenance.py", maintenance)

# Ensure no active tooling still names the retired package scripts.
for rel in (".github/workflows/release.yml", "scripts/audit_release.py", "scripts/README.md"):
    text = read(rel).lower()
    for forbidden in ("package_web.py", "package_nuget.py", "dotnet nuget", "nuget.pkg.github.com"):
        if forbidden in text:
            raise SystemExit(f"retired publication marker remains in {rel}: {forbidden}")

# Verify the changed policy before committing it.
subprocess.run(["python", "scripts/audit_desktop_surface.py"], cwd=ROOT, check=True)
subprocess.run(["python", "scripts/audit_release.py"], cwd=ROOT, check=True)
subprocess.run(["python", "scripts/audit_docs.py"], cwd=ROOT, check=True)
subprocess.run(["python", "-m", "unittest", "discover", "-s", "scripts", "-p", "test_*.py"], cwd=ROOT, check=True)

# Self-remove the migration mechanism from the product branch.
for rel in ("scripts/finalize_020_release_surface.py", ".github/workflows/finalize-020-release-surface.yml"):
    path = ROOT / rel
    if path.exists():
        path.unlink()
