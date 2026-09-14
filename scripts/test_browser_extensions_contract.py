#!/usr/bin/env python3
from __future__ import annotations

import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import zipfile

ROOT = Path(__file__).resolve().parents[1]
EXT_ROOT = ROOT / "ekstenzije"
MANIFESTS = EXT_ROOT / "manifests"
SHARED = EXT_ROOT / "shared"
PACKAGES = ("chrome", "edge", "firefox")
OFFICIAL_EXTENSION = "Ghost FTP Connection Helper"
OFFICIAL_SHORT_NAME = "Ghost FTP"
OFFICIAL_HOMEPAGE = "https://ghostftp.com"
OFFICIAL_FIREFOX_ID = "ghostftp-connection-helper@ghostftp.com"


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def load_manifest(package: str) -> dict:
    return json.loads(read(MANIFESTS / f"{package}.json"))


def load_builder():
    path = ROOT / "scripts" / "build_browser_extensions.py"
    spec = importlib.util.spec_from_file_location("ghostftp_browser_builder", path)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load browser extension builder")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class BrowserExtensionsContractTests(unittest.TestCase):
    def test_brand_contract_is_exact_and_not_manifest_configurable(self) -> None:
        brand = json.loads(read(EXT_ROOT / "BRAND.json"))
        self.assertEqual(brand["product"], "Ghost FTP")
        self.assertEqual(brand["extension"], OFFICIAL_EXTENSION)
        self.assertEqual(brand["short_name"], OFFICIAL_SHORT_NAME)
        self.assertEqual(brand["homepage"], OFFICIAL_HOMEPAGE)
        self.assertEqual(brand["firefox_id"], OFFICIAL_FIREFOX_ID)
        self.assertEqual(brand["official_packages"], list(PACKAGES))

        builder = read(ROOT / "scripts" / "build_browser_extensions.py")
        for exact in ("OFFICIAL_PRODUCT = \"Ghost FTP\"", f'OFFICIAL_EXTENSION = "{OFFICIAL_EXTENSION}"'):
            self.assertIn(exact, builder)

    def test_all_official_manifests_match_root_version_and_brand(self) -> None:
        version = read(ROOT / "VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        for package in PACKAGES:
            manifest = load_manifest(package)
            self.assertEqual(manifest["manifest_version"], 3)
            self.assertEqual(manifest["name"], OFFICIAL_EXTENSION)
            self.assertEqual(manifest["short_name"], OFFICIAL_SHORT_NAME)
            self.assertEqual(manifest["version"], version)
            self.assertEqual(manifest["homepage_url"], OFFICIAL_HOMEPAGE)
            self.assertEqual(manifest["permissions"], [])
            self.assertEqual(manifest["action"]["default_title"], OFFICIAL_EXTENSION)
            self.assertEqual(manifest["action"]["default_popup"], "popup.html")

    def test_firefox_manifest_declares_signing_identity_and_no_collection(self) -> None:
        firefox = load_manifest("firefox")
        self.assertEqual(
            firefox["browser_specific_settings"],
            {
                "gecko": {
                    "id": OFFICIAL_FIREFOX_ID,
                    "data_collection_permissions": {"required": ["none"]},
                }
            },
        )
        self.assertNotIn("browser_specific_settings", load_manifest("chrome"))
        self.assertNotIn("browser_specific_settings", load_manifest("edge"))

    def test_manifests_are_permission_minimal(self) -> None:
        forbidden = {
            "background",
            "content_scripts",
            "externally_connectable",
            "host_permissions",
            "optional_host_permissions",
            "optional_permissions",
            "web_accessible_resources",
        }
        for package in PACKAGES:
            manifest = load_manifest(package)
            self.assertTrue(forbidden.isdisjoint(manifest), package)

    def test_shared_runtime_is_local_only_and_fail_closed(self) -> None:
        html = read(SHARED / "popup.html")
        core = read(SHARED / "core.js")
        popup = read(SHARED / "popup.js")

        self.assertIn("Ghost FTP", html)
        self.assertIn("Connection Helper", html)
        self.assertNotRegex(html, r"https?://")
        self.assertNotIn("<style", html.lower())
        self.assertNotRegex(html.lower(), r"<script(?![^>]*\bsrc=)")
        self.assertIn('src="core.js"', html)
        self.assertIn('src="popup.js"', html)

        runtime = core + popup
        for marker in (
            "fetch(",
            "XMLHttpRequest",
            "WebSocket",
            "sendBeacon",
            "localStorage",
            "sessionStorage",
            "indexedDB",
            "analytics",
            "telemetry",
        ):
            self.assertNotIn(marker, runtime)
        for marker in (
            "MAX_INPUT_LENGTH",
            "CONTROL_CHARACTERS",
            "new URL(",
            "parsed.hostname",
            "parsed.username",
            "parsed.password",
            "passwordDetected",
            "safeTarget",
        ):
            self.assertIn(marker, core)
        for scheme in ("'ftp:'", "'ftps:'", "'sftp:'"):
            self.assertIn(scheme, core)
        self.assertNotIn("parsed.search", core)
        self.assertNotIn("parsed.hash", core)

    def test_browser_builder_outputs_only_three_official_deterministic_packages(self) -> None:
        builder = load_builder()
        with tempfile.TemporaryDirectory() as first_tmp, tempfile.TemporaryDirectory() as second_tmp:
            first = Path(first_tmp)
            second = Path(second_tmp)
            first_paths = builder.build(first)
            second_paths = builder.build(second)
            self.assertEqual([path.name for path in first_paths], [path.name for path in second_paths])
            self.assertEqual(len(first_paths), 3)

            version = read(ROOT / "VERSION").strip()
            expected_names = [f"Ghost-FTP-{version}-{name.capitalize()}-Extension.zip" for name in PACKAGES]
            self.assertEqual([path.name for path in first_paths], expected_names)

            for left, right in zip(first_paths, second_paths):
                self.assertEqual(hashlib.sha256(left.read_bytes()).hexdigest(), hashlib.sha256(right.read_bytes()).hexdigest())
                with zipfile.ZipFile(left, "r") as archive:
                    self.assertEqual(
                        archive.namelist(),
                        ["manifest.json", "core.js", "popup.js", "popup.css", "popup.html", "icons/icon.png"],
                    )
                    manifest = json.loads(archive.read("manifest.json"))
                    self.assertEqual(manifest["name"], OFFICIAL_EXTENSION)
                    self.assertEqual(manifest["version"], version)

    def test_privacy_documentation_is_explicit(self) -> None:
        text = (read(EXT_ROOT / "README.md") + "\n" + read(EXT_ROOT / "PRIVACY.md")).lower()
        for statement in (
            "no telemetry",
            "no tracking",
            "no remote code",
            "does not store",
            "does not read the active tab",
            "does not connect to your ftp, ftps, or sftp server",
        ):
            self.assertIn(statement, text)


if __name__ == "__main__":
    unittest.main()
