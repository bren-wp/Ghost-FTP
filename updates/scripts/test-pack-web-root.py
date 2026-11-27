#!/usr/bin/env python3
"""Synthetic unit fixtures only. Never publish or distribute these test bytes."""
import hashlib
import json
import re
import importlib.util
import tempfile
import unittest
from pathlib import Path
from zipfile import ZipFile

spec = importlib.util.spec_from_file_location("pack_web_root", Path(__file__).with_name("pack-web-root.py"))
packer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(packer)

class RootPackTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.assets = self.root / "assets"
        self.assets.mkdir()
        self.version = "0.99.92"
        self.output = self.root / "root.zip"
        self.sums = self.root / "sums.txt"
        self.names = packer.names(self.version)
        lines = []
        for name in self.names:
            data = packer.package_magic(name) + b"unit fixture only"
            (self.assets / name).write_bytes(data)
            lines.append(f"{hashlib.sha256(data).hexdigest()}  {name}\n")
        self.sums.write_text("".join(lines), encoding="utf-8")
        (self.assets / "GhostFTP-Portable.exe").write_bytes(b"not included")

    def pack(self):
        packer.pack(self.output, self.version, self.assets, self.sums)

    def test_exact_local_inventory_and_determinism(self):
        self.pack()
        first = self.output.read_bytes()
        with ZipFile(self.output) as z:
            self.assertEqual(set(z.namelist()), set(packer.PUBLIC) | {
                *("downloads/" + n for n in self.names),
                "downloads/catalog.json", "downloads/SHA256SUMS.txt"
            })
            rules = z.read(".htaccess").decode()
            self.assertNotIn("__GHOSTFTP_RELEASE_INSTALLERS__", rules)
            for name in self.names:
                self.assertIn(re.escape(name), rules)
            self.assertIn("RewriteRule ^downloads/ - [R=404,END]", rules)
            catalog = json.loads(z.read("downloads/catalog.json"))
            self.assertEqual(catalog["version"], self.version)
            self.assertEqual(set(catalog["packages"]), set(self.names))
            for name in self.names:
                data = (self.assets / name).read_bytes()
                self.assertEqual(z.read("downloads/" + name), data)
                self.assertEqual(catalog["packages"][name], {
                    "size": len(data), "sha256": hashlib.sha256(data).hexdigest(),
                })
        self.pack()
        self.assertEqual(first, self.output.read_bytes())

    def test_missing_installer_fails_without_archive(self):
        (self.assets / self.names[0]).unlink()
        with self.assertRaises(ValueError):
            self.pack()
        self.assertFalse(self.output.exists())

    def test_corrupt_installer_does_not_replace_good_archive(self):
        self.pack()
        original = self.output.read_bytes()
        with (self.assets / self.names[0]).open("ab") as f:
            f.write(b"corruption")
        with self.assertRaisesRegex(ValueError, "SHA-256 mismatch"):
            self.pack()
        self.assertEqual(original, self.output.read_bytes())

    def test_legacy_inventory_and_new_aliases(self):
        self.assertEqual(len(packer.names("0.99.91")), 4)
        self.assertEqual(len(self.names), 7)
        alias = self.names[4]
        data = b"MZdifferent update fixture"
        (self.assets / alias).write_bytes(data)
        lines = self.sums.read_text().splitlines()
        self.sums.write_text("\n".join(
            f"{hashlib.sha256(data).hexdigest()}  {alias}" if line.endswith("  " + alias)
            else line for line in lines) + "\n")
        with self.assertRaisesRegex(ValueError, "differs from original"):
            self.pack()
        self.assertFalse(self.output.exists())

    def test_rejects_traversal_version(self):
        with self.assertRaises(ValueError):
            packer.names("../0.99.90")

if __name__ == "__main__":
    unittest.main()
