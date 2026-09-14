#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class OfficialDestinationsContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_support_current_release_follows_version(self) -> None:
        version = self.read("VERSION").strip()
        support = self.read("docs/SUPPORT.md")
        self.assertIn(f"Ghost FTP **{version}** is the current supported public release", support)

    def test_generic_product_destinations_are_ghostftp_only(self) -> None:
        brand = self.read("internal/brand/brand.go")
        self.assertIn('Website = "ghostftp.com"', brand)
        self.assertIn("Support = Website", brand)
        self.assertNotIn("brendigo", brand.lower())

        support = self.read("docs/SUPPORT.md")
        self.assertIn("https://ghostftp.com", support)
        self.assertNotIn("brendigo", support.lower())

    def test_about_keeps_author_identity_separate(self) -> None:
        identity = self.read("internal/desktop/about_identity_windows.go")
        self.assertIn('aboutPublisher     = "BRENDIGO LTD"', identity)
        self.assertIn('aboutAuthorWebsite = "brendigo.com"', identity)
        self.assertIn('aboutSupport       = "brendigo.com/kontakt"', identity)

    def test_linux_distribution_uses_product_only_identity(self) -> None:
        desktop = self.read("linux/ghost-ftp.desktop")
        build = self.read("linux/BUILD-DISTROS.sh")
        linux_docs = self.read("linux/README.md")

        self.assertIn("Name=Ghost FTP", desktop)
        self.assertIn("GenericName=FTP, FTPS and SFTP Client", desktop)
        self.assertNotIn("brendigo", desktop.lower())

        self.assertIn("Ghost FTP", build)
        self.assertIn("LICENSE", build)
        self.assertNotIn("brendigo.com", build.lower())

        self.assertIn("https://ghostftp.com", linux_docs)
        self.assertNotIn("Homepage: https://github.com/", linux_docs)


if __name__ == "__main__":
    unittest.main()
