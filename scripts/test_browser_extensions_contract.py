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
EXT_ROOT = ROOT / "extensions"
SHARED = EXT_ROOT / "shared"
PACKAGES = ("chrome", "edge", "firefox", "opera")
OFFICIAL_EXTENSION = "Ghost FTP Connection Helper"
OFFICIAL_SHORT_NAME = "Ghost FTP"
OFFICIAL_HOMEPAGE = "https://ghostftp.com"
OFFICIAL_FIREFOX_ID = "ghostftp-connection-helper@ghostftp.com"
EXPECTED_RUNTIME_FILES = [
    "background.js",
    "core.js",
    "popup.js",
    "popup.css",
    "popup.html",
    "icons/icon.png",
]


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def load_manifest(package: str) -> dict:
    return json.loads(read(EXT_ROOT / package / "manifest.json"))


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

    def test_source_tree_is_english_and_browser_specific(self) -> None:
        self.assertFalse((ROOT / "ekstenzije").exists())
        self.assertFalse((EXT_ROOT / "manifests").exists())
        for package in PACKAGES:
            self.assertTrue((EXT_ROOT / package / "manifest.json").is_file(), package)
        for name in EXPECTED_RUNTIME_FILES[:-1]:
            self.assertTrue((SHARED / name).is_file(), name)
        self.assertTrue((ROOT / "cmd" / "ghostftp-native-host" / "main.go").is_file())

    def test_all_official_manifests_match_root_version_brand_and_minimum_permission(self) -> None:
        version = read(ROOT / "VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        for package in PACKAGES:
            manifest = load_manifest(package)
            self.assertEqual(manifest["manifest_version"], 3)
            self.assertEqual(manifest["name"], OFFICIAL_EXTENSION)
            self.assertEqual(manifest["short_name"], OFFICIAL_SHORT_NAME)
            self.assertEqual(manifest["version"], version)
            self.assertEqual(manifest["homepage_url"], OFFICIAL_HOMEPAGE)
            self.assertEqual(manifest["permissions"], ["nativeMessaging"])
            self.assertEqual(manifest["action"]["default_title"], OFFICIAL_EXTENSION)
            self.assertEqual(manifest["action"]["default_popup"], "popup.html")

        self.assertEqual(load_manifest("firefox")["background"], {"scripts": ["background.js"]})
        for package in ("chrome", "edge", "opera"):
            self.assertEqual(load_manifest(package)["background"], {"service_worker": "background.js"})

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
        for package in ("chrome", "edge", "opera"):
            self.assertNotIn("browser_specific_settings", load_manifest(package))

    def test_manifests_do_not_gain_web_or_tab_access(self) -> None:
        forbidden = {
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
            self.assertNotIn("storage", manifest["permissions"])
            self.assertNotIn("tabs", manifest["permissions"])

    def test_shared_runtime_uses_only_local_native_bridge(self) -> None:
        html = read(SHARED / "popup.html")
        core = read(SHARED / "core.js")
        popup = read(SHARED / "popup.js")
        background = read(SHARED / "background.js")
        runtime = "\n".join((core, popup, background))

        self.assertIn("Ghost FTP", html)
        self.assertIn("Connection Helper", html)
        self.assertNotRegex(html, r"https?://")
        self.assertNotIn("<style", html.lower())
        self.assertNotRegex(html.lower(), r"<script(?![^>]*\bsrc=)")
        self.assertIn('src="core.js"', html)
        self.assertIn('src="popup.js"', html)
        self.assertNotIn('src="background.js"', html)

        for marker in (
            "fetch(",
            "XMLHttpRequest",
            "WebSocket",
            "sendBeacon",
            "localStorage",
            "sessionStorage",
            "indexedDB",
            "browser.storage",
            "chrome.storage",
            "analytics",
            "telemetry",
            "eval(",
            "new Function(",
            ".innerHTML",
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

        for marker in (
            "runtime.connectNative",
            "runtime.onConnect",
            "com.ghostftp.bridge",
            "transfer.events",
            "activeTransferCount",
            "closeNativeIfIdle",
        ):
            self.assertIn(marker, background)
        self.assertNotIn("setInterval(", background)

    def test_popup_exposes_real_engine_workflows_without_html_injection(self) -> None:
        popup = read(SHARED / "popup.js")
        html = read(SHARED / "popup.html")
        for operation in (
            "profiles.list",
            "profiles.save",
            "profiles.delete",
            "connect",
            "disconnect",
            "trust.cancel",
            "local.chooseRoot",
            "local.list",
            "local.mkdir",
            "local.rename",
            "local.delete",
            "remote.list",
            "remote.mkdir",
            "remote.rename",
            "remote.delete",
            "transfer.upload",
            "transfer.download",
            "transfer.events",
            "transfer.cancel",
            "transfer.retry",
            "transfer.pause",
            "transfer.resume",
            "transfer.clearFinished",
        ):
            self.assertIn(operation, popup)
        for surface in ("Connection", "Files", "Transfers", "SFTP host key verification"):
            self.assertIn(surface, html)
        self.assertIn("textContent", popup)
        self.assertNotIn("innerHTML", popup)

    def test_native_host_is_bounded_strict_and_uses_existing_engine(self) -> None:
        host = read(ROOT / "cmd" / "ghostftp-native-host" / "main.go")
        for marker in (
            "maxInboundMessage",
            "maxOutboundMessage",
            "DisallowUnknownFields",
            "hasControlCharacters",
            "filepath.EvalSymlinks",
            "pathWithin",
            "api.New",
            "engine.Connect",
            "engine.TransferEvents",
            "usererror.Message",
        ):
            self.assertIn(marker, host)
        self.assertNotIn("http.ListenAndServe", host)
        self.assertNotIn("log.Printf", host)

    def test_browser_builder_outputs_four_official_deterministic_packages(self) -> None:
        builder = load_builder()
        with tempfile.TemporaryDirectory() as first_tmp, tempfile.TemporaryDirectory() as second_tmp:
            first = Path(first_tmp)
            second = Path(second_tmp)
            first_paths = builder.build(first)
            second_paths = builder.build(second)
            self.assertEqual([path.name for path in first_paths], [path.name for path in second_paths])
            self.assertEqual(len(first_paths), 4)

            version = read(ROOT / "VERSION").strip()
            expected_names = [f"Ghost-FTP-{version}-{name.capitalize()}-Extension.zip" for name in PACKAGES]
            self.assertEqual([path.name for path in first_paths], expected_names)

            for left, right in zip(first_paths, second_paths):
                self.assertEqual(hashlib.sha256(left.read_bytes()).hexdigest(), hashlib.sha256(right.read_bytes()).hexdigest())
                with zipfile.ZipFile(left, "r") as archive:
                    self.assertEqual(archive.namelist(), ["manifest.json", *EXPECTED_RUNTIME_FILES])
                    manifest = json.loads(archive.read("manifest.json"))
                    self.assertEqual(manifest["name"], OFFICIAL_EXTENSION)
                    self.assertEqual(manifest["version"], version)
                    self.assertEqual(manifest["permissions"], ["nativeMessaging"])

    def test_privacy_documentation_describes_native_bridge_boundary(self) -> None:
        text = (read(EXT_ROOT / "README.md") + "\n" + read(EXT_ROOT / "PRIVACY.md")).lower()
        for statement in (
            "no telemetry",
            "no tracking",
            "no remote code",
            "does not store credentials in browser storage",
            "does not read the active tab",
            "native messaging",
            "directly",
        ):
            self.assertIn(statement, text)


if __name__ == "__main__":
    unittest.main()
