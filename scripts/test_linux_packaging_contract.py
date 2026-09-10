#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class LinuxPackagingContractTests(unittest.TestCase):
    def test_legacy_generic_build_remains_ci_compatible(self) -> None:
        build = read("linux/BUILD.sh")
        self.assertIn('portable_name="Ghost-FTP-${VERSION}-Linux-${debarch}"', build)
        self.assertIn('portable_out="dist/${portable_name}.tar.gz"', build)
        self.assertIn("command -v dpkg-deb", build)
        self.assertIn("GHOSTFTP_REQUIRE_DEB", build)
        self.assertIn("tar --sort=name --owner=0 --group=0 --numeric-owner", build)
        self.assertIn("gzip -n -9", build)
        self.assertIn('cp "$binary" "$portable_root/ghostftp"', build)
        self.assertIn('cp "$binary" "$deb_root/usr/bin/ghostftp"', build)

    def test_ci_proves_generic_deb_and_portable_binary_parity(self) -> None:
        workflow = read(".github/workflows/ci.yml")
        self.assertIn("Verify DEB and portable packages", workflow)
        self.assertIn("GHOSTFTP_REQUIRE_DEB: '1'", workflow)
        self.assertIn("Ghost-FTP-${version}-Linux-${arch}.tar.gz", workflow)
        self.assertIn('cmp "$work/deb/usr/bin/ghostftp" "$root/ghostftp"', workflow)

    def test_release_workflow_publishes_verified_distro_and_portable_archives(self) -> None:
        workflow = read(".github/workflows/release.yml")
        self.assertIn("Verify distro package metadata and binary parity", workflow)
        self.assertIn("GHOSTFTP_REQUIRE_DEB: '1'", workflow)
        self.assertIn("GHOSTFTP_REQUIRE_RPM: '1'", workflow)
        self.assertIn("bash linux/BUILD-DISTROS.sh", workflow)
        self.assertIn('cmp "$work/${distro,,}/usr/bin/ghostftp" "$portable_root/ghostftp"', workflow)
        self.assertIn('cmp "$work/fedora/usr/bin/ghostftp" "$portable_root/ghostftp"', workflow)
        self.assertIn("dist/Ghost-FTP-*-Linux-Debian-*.deb", workflow)
        self.assertIn("dist/Ghost-FTP-*-Linux-Ubuntu-*.deb", workflow)
        self.assertIn("dist/Ghost-FTP-*-Linux-Fedora-*.rpm", workflow)
        self.assertIn("dist/Ghost-FTP-*-Linux-Portable-*.tar.gz", workflow)
        self.assertIn("LINUX_DEBIAN_DEB=amd64,arm64,i386", workflow)
        self.assertIn("LINUX_UBUNTU_DEB=amd64,arm64,i386", workflow)
        self.assertIn("LINUX_FEDORA_RPM=x86_64,aarch64,i686", workflow)
        self.assertIn("LINUX_PORTABLE=amd64,arm64,i386", workflow)
        self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=14", workflow)
        self.assertIn("PUBLIC_RELEASE_FILES=17", workflow)
        self.assertIn('test "$count" = \'17\'', workflow)
        for arch in ("amd64", "arm64", "i386"):
            self.assertIn(f"Ghost-FTP-${{VERSION}}-Linux-Debian-{arch}.deb", workflow)
            self.assertIn(f"Ghost-FTP-${{VERSION}}-Linux-Ubuntu-{arch}.deb", workflow)
            self.assertIn(f"Ghost-FTP-${{VERSION}}-Linux-Portable-{arch}.tar.gz", workflow)
        for arch in ("x86_64", "aarch64", "i686"):
            self.assertIn(f"Ghost-FTP-${{VERSION}}-Linux-Fedora-{arch}.rpm", workflow)

    def test_active_docs_describe_canonical_0_0_3_packaging(self) -> None:
        version = read("VERSION").strip()
        self.assertEqual(version, "0.0.3")
        linux_readme = read("linux/README.md")
        parity = read("docs/PLATFORM-PARITY.md")
        releases = read("docs/GITHUB-RELEASES.md")
        verification = read("docs/RELEASE-VERIFICATION.md")
        transition = read("docs/PACKAGING-TRANSITION.md")

        self.assertIn(f"Ghost FTP **{version}** is the current public release line", linux_readme)
        self.assertIn(f"Canonical {version} release artifacts", linux_readme)
        self.assertIn("distro-specific artifacts are no longer supplemental", linux_readme)
        self.assertIn(f"ghcr.io/bren-wp/ghost-ftp:{version}", linux_readme)
        self.assertIn("14 platform artifacts / 17 public files", parity)
        self.assertIn("14 platform artifacts", releases)
        self.assertIn("17 public files", releases)
        self.assertIn("14 platform artifacts", verification)
        self.assertIn("17 public files", verification)
        self.assertIn("0.0.3 source candidate", transition)
        self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=14", transition)
        self.assertIn("PUBLIC_RELEASE_FILES=17", transition)


if __name__ == "__main__":
    unittest.main()
