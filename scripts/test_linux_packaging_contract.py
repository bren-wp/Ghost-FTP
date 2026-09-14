#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class LinuxPackagingContractTests(unittest.TestCase):
    def test_universal_builder_emits_one_installer_and_portable_per_distro(self) -> None:
        build = read("linux/BUILD-DISTROS.sh")
        self.assertIn("for distro in Debian Ubuntu Fedora; do", build)
        self.assertIn('Ghost-FTP-${VERSION}-Linux-${distro}-Installer.run', build)
        self.assertIn('Ghost-FTP-${VERSION}-Linux-${distro}-Portable', build)
        self.assertIn("build_arch amd64 amd64", build)
        self.assertIn("build_arch arm64 arm64", build)
        self.assertIn("build_arch 386 i386", build)
        self.assertIn("__GHOSTFTP_PAYLOAD_BELOW__", build)
        self.assertIn("ghostftp-uninstall", build)
        self.assertIn("GHOSTFTP_PREFIX", build)
        self.assertNotIn("dpkg-deb", build)
        self.assertNotIn("rpmbuild", build)

    def test_ci_proves_universal_bundle_architecture_and_installer_parity(self) -> None:
        workflow = read(".github/workflows/ci.yml")
        self.assertIn("Linux universal distro bundles with amd64 arm64 i386 payloads", workflow)
        self.assertIn("bash linux/BUILD-DISTROS.sh", workflow)
        self.assertIn("Ghost-FTP-${version}-Linux-${distro}-Installer.run", workflow)
        self.assertIn("Ghost-FTP-${version}-Linux-${distro}-Portable.tar.gz", workflow)
        self.assertIn("bin/amd64/ghostftp", workflow)
        self.assertIn("bin/arm64/ghostftp", workflow)
        self.assertIn("bin/i386/ghostftp", workflow)
        self.assertIn("ghostftp-uninstall", workflow)
        self.assertIn('test "$count" = \'6\'', workflow)

    def test_release_workflow_publishes_exact_universal_linux_set(self) -> None:
        workflow = read(".github/workflows/release.yml")
        self.assertIn("Build universal distro bundles", workflow)
        self.assertIn("Verify universal distro bundle parity", workflow)
        self.assertIn("bash linux/BUILD-DISTROS.sh", workflow)
        for distro in ("Debian", "Ubuntu", "Fedora"):
            self.assertIn(f"Ghost-FTP-${{VERSION}}-Linux-{distro}-Installer.run", workflow)
            self.assertIn(f"Ghost-FTP-${{VERSION}}-Linux-{distro}-Portable.tar.gz", workflow)
        self.assertIn("LINUX_DEBIAN_INSTALLER=universal-amd64-arm64-i386", workflow)
        self.assertIn("LINUX_DEBIAN_PORTABLE=universal-amd64-arm64-i386", workflow)
        self.assertIn("LINUX_UBUNTU_INSTALLER=universal-amd64-arm64-i386", workflow)
        self.assertIn("LINUX_UBUNTU_PORTABLE=universal-amd64-arm64-i386", workflow)
        self.assertIn("LINUX_FEDORA_INSTALLER=universal-amd64-arm64-i386", workflow)
        self.assertIn("LINUX_FEDORA_PORTABLE=universal-amd64-arm64-i386", workflow)
        self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=13", workflow)
        self.assertIn("PUBLIC_RELEASE_FILES=16", workflow)
        self.assertIn('test "$count" = \'16\'', workflow)
        self.assertIn("Ghost-FTP-${VERSION}-Android.apk", workflow)
        self.assertIn("Ghost-FTP-${VERSION}-Chrome-Extension.zip", workflow)
        self.assertIn("Ghost-FTP-${VERSION}-Edge-Extension.zip", workflow)
        self.assertIn("Ghost-FTP-${VERSION}-Firefox-Extension.zip", workflow)
        self.assertIn("Ghost-FTP-${VERSION}-Opera-Extension.zip", workflow)
        self.assertNotIn("Linux-Debian-amd64.deb", workflow)
        self.assertNotIn("Linux-Fedora-x86_64.rpm", workflow)

    def test_published_docs_and_next_release_contract_are_not_conflated(self) -> None:
        version = read("VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        self.assertNotEqual(version, "0.0.0")

        # Until the explicit 0.0.7 version bump, public documentation continues
        # to describe the actually published 0.0.6 asset set. The next release
        # workflow is validated independently above.
        for rel in (
            "README.md",
            "docs/INSTALLATION.md",
            "docs/GITHUB-RELEASES.md",
            "docs/RELEASE-VERIFICATION.md",
        ):
            self.assertIn("18 platform artifacts", read(rel))
        self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=13", read(".github/workflows/release.yml"))
        self.assertIn("PUBLIC_RELEASE_FILES=16", read(".github/workflows/release.yml"))


if __name__ == "__main__":
    unittest.main()
