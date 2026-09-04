from __future__ import annotations

import os
import plistlib
import stat
import tempfile
import unittest
import zipfile
from pathlib import Path

import package_ios


class IOSPackageTests(unittest.TestCase):
    def make_app(self, root: Path) -> tuple[Path, str]:
        version = package_ios.canonical_version()
        app = root / "GhostFTP.app"
        app.mkdir()
        with (app / "Info.plist").open("wb") as handle:
            plistlib.dump(
                {
                    "CFBundleIdentifier": "com.GhostFTP.client",
                    "CFBundleShortVersionString": version,
                    "CFBundleExecutable": "GhostFTP",
                },
                handle,
            )
        executable = app / "GhostFTP"
        executable.write_bytes(b"\xcf\xfa\xed\xfe" + b"\x00" * 128)
        executable.chmod(stat.S_IRUSR | stat.S_IWUSR | stat.S_IXUSR)
        (app / "asset.txt").write_text("asset", encoding="utf-8")
        return app, version

    def test_valid_app_creates_ipa_and_app_zip(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            app, _ = self.make_app(root)
            package_ios.validate_app(app, package_ios.canonical_version())
            ipa = root / "out.ipa"
            app_zip = root / "app.zip"
            package_ios.archive_tree(app, ipa, "Payload/GhostFTP.app")
            package_ios.archive_tree(app, app_zip, "GhostFTP.app")
            package_ios.validate_archive(ipa, {"Payload/GhostFTP.app/Info.plist", "Payload/GhostFTP.app/GhostFTP"})
            package_ios.validate_archive(app_zip, {"GhostFTP.app/Info.plist", "GhostFTP.app/GhostFTP"})
            with zipfile.ZipFile(ipa) as archive:
                self.assertIn("Payload/GhostFTP.app/asset.txt", archive.namelist())

    def test_version_mismatch_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            app, _ = self.make_app(Path(temp))
            with self.assertRaises(SystemExit):
                package_ios.validate_app(app, "99.99.99")

    @unittest.skipIf(os.name == "nt", "symlink semantics differ on Windows")
    def test_symlink_in_app_is_rejected(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            app, _ = self.make_app(root)
            (app / "link").symlink_to(app / "asset.txt")
            with self.assertRaises(SystemExit):
                package_ios.validate_app(app, package_ios.canonical_version())


if __name__ == "__main__":
    unittest.main()
