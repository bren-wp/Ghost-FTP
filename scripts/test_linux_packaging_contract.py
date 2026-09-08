#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class LinuxPackagingContractTests(unittest.TestCase):
    def test_build_emits_portable_archives_without_forcing_deb_tooling(self) -> None:
        build = read("linux/BUILD.sh")
        self.assertIn('portable_name="Ghost-FTP-${VERSION}-Linux-${debarch}"', build)
        self.assertIn('portable_out="dist/${portable_name}.tar.gz"', build)
        self.assertIn("command -v dpkg-deb", build)
        self.assertIn("GHOSTFTP_REQUIRE_DEB", build)
        self.assertIn("tar --sort=name --owner=0 --group=0 --numeric-owner", build)
        self.assertIn("gzip -n -9", build)
        self.assertIn('cp "$binary" "$portable_root/ghostftp"', build)
        self.assertIn('cp "$binary" "$deb_root/usr/bin/ghostftp"', build)

    def test_ci_proves_deb_and_portable_binary_parity(self) -> None:
        workflow = read(".github/workflows/ci.yml")
        self.assertIn("Verify DEB and portable packages", workflow)
        self.assertIn("GHOSTFTP_REQUIRE_DEB: '1'", workflow)
        self.assertIn("Ghost-FTP-${version}-Linux-${arch}.tar.gz", workflow)
        self.assertIn('cmp "$work/deb/usr/bin/ghostftp" "$root/ghostftp"', workflow)
        self.assertIn("dist/Ghost-FTP-*-Linux-*.tar.gz", workflow)

    def test_docs_do_not_retroactively_claim_tarballs_for_116(self) -> None:
        linux_readme = read("linux/README.md")
        parity = read("docs/PLATFORM-PARITY.md")
        self.assertIn("already published Ghost FTP 1.1.6 release is immutable", linux_readme)
        self.assertIn("is **not** retroactively claimed as a 1.1.6 release asset", linux_readme)
        self.assertIn("source/CI outputs only until a separate release-contract change", parity)


if __name__ == "__main__":
    unittest.main()
