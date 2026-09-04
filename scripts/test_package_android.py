from __future__ import annotations

import tempfile
import unittest
import zipfile
from pathlib import Path

import package_android


class PackageAndroidTests(unittest.TestCase):
    def make_apk(self, path: Path, extra_entries: tuple[str, ...] = ()) -> None:
        with zipfile.ZipFile(path, "w") as archive:
            archive.writestr("AndroidManifest.xml", b"manifest")
            archive.writestr("classes.dex", b"dex")
            archive.writestr("resources.arsc", b"resources")
            for entry in extra_entries:
                archive.writestr(entry, b"x")

    def test_stages_versioned_debug_and_unsigned_release_apks(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            debug = root / "app-debug.apk"
            release = root / "app-release-unsigned.apk"
            out = root / "dist"
            self.make_apk(debug)
            self.make_apk(release)

            debug_out, release_out = package_android.stage_apks(debug, release, out, "1.1.1")

            self.assertEqual(debug_out.name, "GhostFTP-1.1.1-Android-debug.apk")
            self.assertEqual(release_out.name, "GhostFTP-1.1.1-Android-release-unsigned.apk")
            package_android.validate_apk(debug_out)
            package_android.validate_apk(release_out)

    def test_rejects_missing_required_apk_entry(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            apk = Path(tmp) / "broken.apk"
            with zipfile.ZipFile(apk, "w") as archive:
                archive.writestr("AndroidManifest.xml", b"manifest")
                archive.writestr("classes.dex", b"dex")
            with self.assertRaises(package_android.PackageError):
                package_android.validate_apk(apk)

    def test_rejects_unsafe_zip_entry(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            apk = Path(tmp) / "unsafe.apk"
            self.make_apk(apk, ("../escape",))
            with self.assertRaises(package_android.PackageError):
                package_android.validate_apk(apk)

    def test_rejects_non_semantic_version(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            version_file = Path(tmp) / "VERSION"
            version_file.write_text("1.1\n", encoding="utf-8")
            with self.assertRaises(package_android.PackageError):
                package_android.read_version(version_file)


if __name__ == "__main__":
    unittest.main()
