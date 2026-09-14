#!/usr/bin/env python3
"""Static regression checks for Ghost FTP architecture-selecting Linux bundles."""

from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
BUILD = (ROOT / "linux" / "BUILD-DISTROS.sh").read_text(encoding="utf-8")
WORKFLOW = (ROOT / ".github" / "workflows" / "linux-distro-packages.yml").read_text(encoding="utf-8")
VERSION = (ROOT / "VERSION").read_text(encoding="utf-8").strip()


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


require(re.fullmatch(r"\d+\.\d+\.\d+", VERSION) is not None, "VERSION must be X.Y.Z")

for mapping in (
    "build_arch amd64 amd64",
    "build_arch arm64 arm64",
    "build_arch 386 i386",
):
    require(mapping in BUILD, f"missing architecture mapping: {mapping}")

for uname_case in (
    "x86_64|amd64) arch=amd64",
    "aarch64|arm64) arch=arm64",
    "i386|i486|i586|i686|x86) arch=i386",
):
    require(BUILD.count(uname_case) >= 2, f"launcher/installer architecture dispatch missing: {uname_case}")

require("for distro in Debian Ubuntu Fedora; do" in BUILD, "Debian/Ubuntu/Fedora build loop missing")
require('make_installer "$distro"' in BUILD, "data-driven installer build call missing")
require('make_portable "$distro"' in BUILD, "data-driven portable build call missing")
for distro in ("Debian", "Ubuntu", "Fedora"):
    require(f'make_installer "{distro}"' not in BUILD, f"{distro} installer must not be hard-coded outside the distro loop")
    require(f'make_portable "{distro}"' not in BUILD, f"{distro} portable bundle must not be hard-coded outside the distro loop")

require('Ghost-FTP-${VERSION}-Linux-${distro}-Installer.run' in BUILD, "universal installer naming contract missing")
require('Ghost-FTP-${VERSION}-Linux-${distro}-Portable' in BUILD, "universal portable naming contract missing")
require("__GHOSTFTP_PAYLOAD_BELOW__" in BUILD, "self-extracting installer marker missing")
require("gzip -n -9" in BUILD, "deterministic gzip contract missing")
require("--sort=name" in BUILD and "--numeric-owner" in BUILD, "deterministic tar contract missing")
require("GHOSTFTP_PREFIX" in BUILD, "installer must support an explicit installation prefix")
require("unsupported Linux CPU architecture" in BUILD, "unsupported architectures must fail closed")
require('exec "$base_dir/bin/$arch/ghostftp" "$@"' in BUILD, "portable launcher must dispatch to selected native payload")
require('install -m 0755 "$base_dir/bin/$arch/ghostftp" "$bin_dir/ghostftp"' in BUILD, "installer must install selected native payload")

for obsolete in ("dpkg-deb", "rpmbuild", ".deb\"", ".rpm\""):
    require(obsolete not in BUILD, f"obsolete architecture-specific package tooling remains in builder: {obsolete}")

require("python scripts/test_linux_distro_packaging_contract.py" in WORKFLOW, "workflow must run contract regression test")
require("Installer.run" in WORKFLOW, "workflow must verify installer artifacts")
require("Portable.tar.gz" in WORKFLOW, "workflow must verify portable artifacts")
require("bin/amd64/ghostftp" in WORKFLOW, "workflow must verify amd64 payload")
require("bin/arm64/ghostftp" in WORKFLOW, "workflow must verify arm64 payload")
require("bin/i386/ghostftp" in WORKFLOW, "workflow must verify i386 payload")
require("GHOSTFTP_PREFIX" in WORKFLOW, "workflow must exercise installer without modifying runner system directories")

print("LINUX_DISTRO_PACKAGING_CONTRACT=PASS")
