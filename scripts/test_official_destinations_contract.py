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
        self.assertIn(f"Ghost FTP **{version} Stable**", support)

    def test_product_and_author_sites_are_distinct_and_canonical(self) -> None:
        brand = self.read("internal/brand/brand.go")
        self.assertIn('Website       = "ghostftp.com"', brand)
        self.assertIn('AuthorWebsite = "brendigo.com"', brand)
        self.assertIn('Support       = "brendigo.com/kontakt"', brand)

        support = self.read("docs/SUPPORT.md")
        self.assertIn("https://ghostftp.com", support)
        self.assertIn("https://brendigo.com", support)

    def test_linux_package_uses_product_homepage_and_publisher_identity(self) -> None:
        control = self.read("linux/debian/control.in")
        self.assertIn("Maintainer: BRENDIGO LTD <https://brendigo.com>", control)
        self.assertIn("Homepage: https://ghostftp.com", control)
        self.assertNotIn("Homepage: https://github.com/", control)


if __name__ == "__main__":
    unittest.main()
