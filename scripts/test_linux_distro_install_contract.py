#!/usr/bin/env python3
"""Static regression contract for universal Linux installer verification."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERIFY = (ROOT / "scripts" / "verify_linux_distro_install.sh").read_text(encoding="utf-8")
WORKFLOW = (ROOT / ".github" / "workflows" / "linux-distro-install.yml").read_text(encoding="utf-8")
BUILD = (ROOT / "linux" / "BUILD-DISTROS.sh").read_text(encoding="utf-8")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


for token in (
    "source /etc/os-release",
    '[[ "${ID:-}" == "debian" ]]',
    '[[ "${ID:-}" == "ubuntu" ]]',
    '[[ "${ID:-}" == "fedora" ]]',
    '[[ "${VERSION_ID:-}" == "$expected_os_version" ]]',
):
    require(token in VERIFY, f"missing distro identity guard: {token}")

for token in (
    "ca-certificates curl openssh-client xvfb",
    "ca-certificates curl openssh-clients xorg-x11-server-Xvfb",
    "command -v \"$tool\"",
    "/etc/ssl/certs/ca-certificates.crt",
    "/etc/pki/tls/certs/ca-bundle.crt",
    "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem",
):
    require(token in VERIFY, f"missing clean-container runtime dependency verification: {token}")

for token in (
    'GHOSTFTP_PREFIX="$prefix" "$installer_path"',
    'test -x "$prefix/bin/ghostftp"',
    'test -x "$prefix/bin/ghostftp-uninstall"',
    'test -f "$prefix/share/applications/ghost-ftp.desktop"',
    'test -f "$prefix/share/icons/hicolor/512x512/apps/ghost-ftp.png"',
    'test -f "$prefix/share/doc/ghost-ftp/LICENSE"',
    'test -f "$prefix/share/doc/ghost-ftp/README.md"',
    "grep -Fx 'Exec=ghostftp'",
):
    require(token in VERIFY, f"missing universal installer payload verification: {token}")

for token in (
    "Xvfb :99",
    "-nolisten tcp",
    "mktemp -d /var/lib/ghostftp-ci-home.XXXXXX",
    'chmod 0700 "$smoke_home"',
    'mkdir -p "$smoke_home/.local/share"',
    'runtime_dir="$smoke_home/runtime"',
    'chmod 0700 "$runtime_dir"',
    'HOME="$smoke_home" XDG_RUNTIME_DIR="$runtime_dir" DISPLAY=:99',
    '"$prefix/bin/ghostftp"',
    'kill -0 "$app_pid"',
    "GHOSTFTP_INSTALLED_GUI_SMOKE=PASS",
):
    require(token in VERIFY, f"missing installed GUI smoke contract: {token}")
require('smoke_home="$(mktemp -d)"' not in VERIFY, "GUI smoke HOME must not use default /tmp")
require("XDG_DATA_HOME=" not in VERIFY, "smoke must exercise the default $HOME/.local/share path")

for token in (
    '"$prefix/bin/ghostftp-uninstall"',
    'test ! -e "$prefix/bin/ghostftp"',
    'test ! -e "$prefix/bin/ghostftp-uninstall"',
    'test ! -e "$prefix/share/applications/ghost-ftp.desktop"',
    'test ! -e "$prefix/share/icons/hicolor/512x512/apps/ghost-ftp.png"',
    'test ! -e "$prefix/share/doc/ghost-ftp/LICENSE"',
    'test ! -e "$prefix/share/doc/ghost-ftp/README.md"',
):
    require(token in VERIFY, f"missing uninstall residue assertion: {token}")
require("rm -rf /root" not in VERIFY, "verifier must not delete user home data")

for token in (
    "ghostftp-uninstall",
    "User configuration and server data were not touched.",
    "Debian/Ubuntu: install ca-certificates curl openssh-client",
    "Fedora: install ca-certificates curl openssh-clients",
    'Exec=ghostftp',
):
    require(token in (BUILD + (ROOT / "linux" / "ghost-ftp.desktop").read_text(encoding="utf-8")), f"missing installer safety/usability contract: {token}")

for image in ("debian:13-slim", "ubuntu:26.04", "fedora:44"):
    require(image in WORKFLOW, f"missing pinned install-test image: {image}")
for token in (
    "bash linux/BUILD-DISTROS.sh",
    ":/workspace:ro",
    "Ghost-FTP-${version}-Linux-Debian-Installer.run",
    "Ghost-FTP-${version}-Linux-Ubuntu-Installer.run",
    "Ghost-FTP-${version}-Linux-Fedora-Installer.run",
    "scripts/verify_linux_distro_install.sh debian",
    "scripts/verify_linux_distro_install.sh ubuntu",
    "scripts/verify_linux_distro_install.sh fedora",
    "does not claim native ARM64/i386 runtime execution",
):
    require(token in WORKFLOW, f"missing workflow install contract: {token}")
for retired in ("GHOSTFTP_REQUIRE_DEB", "GHOSTFTP_REQUIRE_RPM", ".deb\"", ".rpm\""):
    require(retired not in WORKFLOW, f"retired architecture-specific package marker remains in workflow: {retired}")
require("--privileged" not in WORKFLOW, "install verification containers must not be privileged")

print("LINUX_DISTRO_INSTALL_CONTRACT=PASS")
