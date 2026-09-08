#!/usr/bin/env python3
"""Static regression checks for Ghost FTP distro packaging contracts.

The workflow also performs real package builds and metadata/binary parity checks.
This test protects the source-level invariants that make those runtime checks
meaningful: one canonical binary per Go architecture, explicit distro metadata,
canonical public artifact names, and the expected dependency contracts.
"""

from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
BUILD = (ROOT / "linux" / "BUILD-DISTROS.sh").read_text(encoding="utf-8")
RPM = (ROOT / "linux" / "rpm" / "ghost-ftp.spec.in").read_text(encoding="utf-8")
DEB = (ROOT / "linux" / "debian" / "control.in").read_text(encoding="utf-8")
WORKFLOW = (ROOT / ".github" / "workflows" / "linux-distro-packages.yml").read_text(encoding="utf-8")
VERSION = (ROOT / "VERSION").read_text(encoding="utf-8").strip()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


require(re.fullmatch(r"\d+\.\d+\.\d+", VERSION) is not None, "VERSION must be X.Y.Z")
require(BUILD.count("go build -trimpath") == 1, "distro builder must compile one canonical payload")
require('cp "$binary" "$portable_root/ghostftp"' in BUILD, "portable package must use canonical binary")
require('cp "$binary" "$deb_root/usr/bin/ghostftp"' in BUILD, "DEB packages must use canonical binary")
require('cp "$binary" "$rpm_top/SOURCES/ghostftp"' in BUILD, "RPM package must use canonical binary")

for mapping in (
    "build_distro_arch amd64 amd64 x86_64",
    "build_distro_arch arm64 arm64 aarch64",
    "build_distro_arch 386 i386 i686",
):
    require(mapping in BUILD, f"missing architecture mapping: {mapping}")

require("for distro in Debian Ubuntu; do" in BUILD, "Debian and Ubuntu package loop missing")
require('printf \'X-GhostFTP-Distribution: %s\\n\' "$distro"' in BUILD, "DEB distro metadata missing")
require('Ghost-FTP-${VERSION}-Linux-${slug}-${debarch}.deb' in BUILD, "DEB artifact naming contract missing")

require("rpmbuild" in BUILD and "-bb" in BUILD, "Fedora package must be built with rpmbuild")
require('Ghost-FTP-${VERSION}-Linux-Fedora-${rpmarch}.rpm' in BUILD, "Fedora artifact naming contract missing")
for field in (
    "Name:           ghost-ftp",
    "Version:        @VERSION@",
    "Distribution:   Fedora",
    "URL:            https://ghostftp.com",
    "Vendor:         Ghost FTP",
    "Packager:       Ghost FTP <https://ghostftp.com>",
    "Requires:       ca-certificates, curl, openssh-clients",
):
    require(field in RPM, f"RPM metadata contract missing: {field}")
require("BRENDIGO" not in RPM.upper() and "brendigo.com" not in RPM.lower(), "RPM metadata must be Ghost FTP-only")

for field in (
    "Package: ghost-ftp",
    "Version: @VERSION@",
    "Architecture: @ARCH@",
    "Maintainer: Ghost FTP <https://ghostftp.com>",
    "Depends: ca-certificates, curl, openssh-client",
    "Homepage: https://ghostftp.com",
):
    require(field in DEB, f"DEB metadata contract missing: {field}")
require("BRENDIGO" not in DEB.upper() and "brendigo.com" not in DEB.lower(), "DEB metadata must be Ghost FTP-only")

for payload in (
    'cp "$binary" "$portable_root/ghostftp"',
    'cp linux/ghost-ftp.desktop "$portable_root/ghost-ftp.desktop"',
    'cp build/icon.png "$portable_root/ghost-ftp.png"',
    'cp LICENSE "$portable_root/LICENSE"',
    'cp linux/README.md "$portable_root/README.md"',
):
    require(payload in BUILD, f"portable payload contract missing: {payload}")
require('portable_name="Ghost-FTP-${VERSION}-Linux-Portable-${debarch}"' in BUILD, "portable base naming contract missing")
require('portable_out="dist/${portable_name}.tar.gz"' in BUILD, "portable archive naming contract missing")

require("python scripts/test_linux_distro_packaging_contract.py" in WORKFLOW, "workflow must run contract regression test")
require(WORKFLOW.count("cmp ") >= 2, "workflow must compare DEB/Ubuntu/RPM binaries with portable payload")
require("dpkg-deb -f" in WORKFLOW, "workflow must inspect DEB metadata")
require("rpm -qp" in WORKFLOW, "workflow must inspect RPM metadata")

print("LINUX_DISTRO_PACKAGING_CONTRACT=PASS")
