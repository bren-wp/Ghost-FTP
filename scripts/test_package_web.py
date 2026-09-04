#!/usr/bin/env python3
"""Regression coverage for the deployable Ghost FTP web release archive."""

from __future__ import annotations

import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]


class WebPackageTests(unittest.TestCase):
    def test_package_contains_only_safe_tracked_web_files(self) -> None:
        version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
        with tempfile.TemporaryDirectory() as temp_dir:
            subprocess.run(
                [sys.executable, str(ROOT / "scripts/package_web.py"), "--output-dir", temp_dir],
                cwd=ROOT,
                check=True,
            )
            package = Path(temp_dir) / f"Ghost-FTP-{version}-Web.zip"
            self.assertTrue(package.is_file())
            self.assertGreater(package.stat().st_size, 0)

            tracked = subprocess.run(
                ["git", "ls-files", "-z", "--", "ByFTP WEB"],
                cwd=ROOT,
                check=True,
                stdout=subprocess.PIPE,
            ).stdout
            expected = sorted(
                (
                    raw.decode("utf-8").removeprefix("ByFTP WEB/")
                    for raw in tracked.split(b"\0")
                    if raw
                ),
                key=str.casefold,
            )

            with zipfile.ZipFile(package, "r") as zf:
                names = zf.namelist()
                self.assertEqual(expected, names)
                self.assertEqual(version, zf.read("VERSION").decode("utf-8").strip())
                self.assertIn(f"ghostftp-static-v{version}", zf.read("service-worker.js").decode("utf-8"))
                composer = zf.read("composer.json").decode("utf-8")
                self.assertIn('"name": "brendigo/ghost-ftp-web"', composer)
                self.assertIn("storage/.htaccess", names)
                self.assertNotIn("storage/users.json", names)
                self.assertNotIn("storage/config.json", names)
                for name in names:
                    pure = PurePosixPath(name)
                    self.assertFalse(pure.is_absolute(), name)
                    self.assertNotIn("..", pure.parts, name)
                    self.assertNotIn("\\", name)


if __name__ == "__main__":
    unittest.main()
