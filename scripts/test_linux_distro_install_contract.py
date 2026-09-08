#!/usr/bin/env python3
"""Static regression contract for Linux package installation verification."""

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERIFY = (ROOT / "scripts" / "verify_linux_distro_install.sh").read_text(encoding="utf-8")
WORKFLOW = (ROOT / ".github" / "workflows" / "linux-distro-install.yml").read_text(encoding="utf-8") if (ROOT / ".github" / "workflows" / "linux-distro-install.yml").exists() else ""


def require(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


# The verifier must fail closed when the actual container OS/version differs
# from the public distro target that the package claims to support.
for token in (
    "source /etc/os-release",
    '[[ "${ID:-}" == "debian" ]]',
    '[[ "${ID:-}" == "ubuntu" ]]',
    '[[ "${ID:-}" == "fedora" ]]',
    '[[ "${VERSION_ID:-}" == "$expected_os_version" ]]',
):
    require(token in VERIFY, f"missing distro identity guard: {token}")

# Real install/remove state is mandatory; file-existence-only checks are not.
for token in (
    'apt-get install -y --no-install-recommends "$package_path"',
    "dpkg-query -W -f='${Status}' ghost-ftp",
    "dpkg-query -L ghost-ftp",
    "apt-get remove -y ghost-ftp",
    'dnf install -y "$package_path"',
    "rpm -q --qf '%{VERSION}' ghost-ftp",
    "rpm -ql ghost-ftp",
    "dnf remove -y ghost-ftp",
):
    require(token in VERIFY, f"missing package lifecycle verification: {token}")

# Dependency contracts and installed runtime tools must be proved inside the
# clean target distribution after package installation. Fedora may satisfy the
# `curl` dependency with an implementation-specific provider such as
# curl-minimal, so bind the proof to the actual executable and its RPM owner.
for token in (
    "ca-certificates, curl, openssh-client",
    "openssh-clients",
    'curl_path="$(command -v curl)"',
    'rpm -qf "$curl_path"',
    "command -v ssh",
    "command -v sftp",
):
    require(token in VERIFY, f"missing dependency/runtime verification: {token}")
require("rpm -q --whatprovides curl" not in VERIFY, "Fedora verifier must not assume a literal curl capability provider name")

# Fedora CA trust layout must be derived from the installed ca-certificates RPM,
# not from one historical /etc/pki path that may change between Fedora releases.
for token in (
    "verify_fedora_ca_bundle()",
    "rpm -ql ca-certificates",
    "*/tls-ca-bundle.pem|*/ca-bundle.crt|*/ca-certificates.crt",
    'rpm -qf "$ca_bundle"',
    "GHOSTFTP_FEDORA_CA_BUNDLE=",
):
    require(token in VERIFY, f"missing Fedora CA-bundle contract: {token}")
require("/etc/pki/tls/certs/ca-bundle.crt" not in VERIFY, "Fedora verifier must not hardcode one CA bundle path")

# Fedora assertions should identify the exact failing contract instead of being
# silent under set -e. This keeps future packaging regressions deterministic.
for token in (
    "GHOSTFTP_FEDORA_VERIFY_FAIL=",
    "fedora_fail installed-version",
    "fedora_fail curl-rpm-owner",
    "fedora_fail ca-bundle",
    "fedora_fail ghostftp-executable",
    "fedora_fail owns-executable",
    "fedora_fail package-still-installed",
):
    require(token in VERIFY, f"missing Fedora diagnostic contract: {token}")

# The installed GUI must actually start under an isolated local X server. The
# test reproduces a normal Linux data-root precondition instead of weakening the
# production safe-path validator, and it provides a private runtime directory.
for token in (
    "Xvfb :99",
    "-nolisten tcp",
    "mktemp -d /var/lib/ghostftp-ci-home.XXXXXX",
    'chmod 0700 "$smoke_home"',
    'mkdir -p "$smoke_home/.local/share"',
    'chmod 0700 "$smoke_home/.local" "$smoke_home/.local/share"',
    'data_root="$smoke_home/.local/share"',
    'runtime_dir="$smoke_home/runtime"',
    'chmod 0700 "$runtime_dir"',
    "HOME=\"$smoke_home\" XDG_RUNTIME_DIR=\"$runtime_dir\" DISPLAY=:99",
    "/usr/bin/ghostftp",
    'kill -0 "$app_pid"',
    "GHOSTFTP_INSTALLED_GUI_SMOKE=PASS",
):
    require(token in VERIFY, f"missing installed GUI smoke contract: {token}")
require('smoke_home="$(mktemp -d)"' not in VERIFY, "GUI smoke HOME must not be created under default /tmp")
require("XDG_DATA_HOME=" not in VERIFY, "smoke should exercise the default $HOME/.local/share LocalAppData path")

# Uninstall verification is scoped to package-owned system locations; user data
# must never be deleted merely to make an uninstall test pass.
for path in (
    "/usr/bin/ghostftp",
    "/usr/share/applications/ghost-ftp.desktop",
    "/usr/share/icons/hicolor/512x512/apps/ghost-ftp.png",
):
    require(f"test ! -e {path}" in VERIFY, f"missing uninstall residue assertion: {path}")
require("rm -rf /root" not in VERIFY, "verifier must not delete user home data")

# The workflow pins concrete supported distro generations and rebuilds package
# artifacts from the exact checked-out commit before entering clean containers.
for image in ("debian:13-slim", "ubuntu:26.04", "fedora:44"):
    require(image in WORKFLOW, f"missing pinned install-test image: {image}")
for token in (
    "GHOSTFTP_REQUIRE_DEB: '1'",
    "GHOSTFTP_REQUIRE_RPM: '1'",
    "bash linux/BUILD-DISTROS.sh",
    ":/workspace:ro",
    "scripts/verify_linux_distro_install.sh debian",
    "scripts/verify_linux_distro_install.sh ubuntu",
    "scripts/verify_linux_distro_install.sh fedora",
):
    require(token in WORKFLOW, f"missing workflow install contract: {token}")
require("--privileged" not in WORKFLOW, "install verification containers must not be privileged")

print("LINUX_DISTRO_INSTALL_CONTRACT=PASS")
