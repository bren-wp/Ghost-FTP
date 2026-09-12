#!/usr/bin/env python3
import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
EXT_ROOT = ROOT / "ekstenzije"
PACKAGES = ("chromium", "firefox")
SHARED_FILES = ("core.js", "popup.js", "popup.css", "popup.html")


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


class BrowserExtensionsContractTests(unittest.TestCase):
    def manifest(self, package: str) -> dict:
        return json.loads(read(EXT_ROOT / package / "manifest.json"))

    def test_supported_packages_and_product_version(self) -> None:
        version = read(ROOT / "VERSION").strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+$")
        for package in PACKAGES:
            manifest = self.manifest(package)
            self.assertEqual(manifest["manifest_version"], 3)
            self.assertEqual(manifest["name"], "Ghost FTP Connection Helper")
            self.assertEqual(manifest["version"], version)
            self.assertEqual(manifest["action"]["default_popup"], "popup.html")

    def test_chromium_family_is_one_shared_package(self) -> None:
        chromium_readme = read(EXT_ROOT / "README.md")
        for browser in ("Google Chrome", "Microsoft Edge", "Opera", "Brave", "Vivaldi"):
            self.assertIn(browser, chromium_readme)
        self.assertIn("Mozilla Firefox", chromium_readme)

    def test_manifests_are_permission_minimal_and_have_no_background_access(self) -> None:
        forbidden_keys = {
            "background",
            "content_scripts",
            "externally_connectable",
            "host_permissions",
            "optional_host_permissions",
            "web_accessible_resources",
        }
        for package in PACKAGES:
            manifest = self.manifest(package)
            self.assertEqual(manifest.get("permissions", []), [])
            self.assertTrue(forbidden_keys.isdisjoint(manifest))

    def test_extension_sources_are_local_only(self) -> None:
        for package in PACKAGES:
            html = read(EXT_ROOT / package / "popup.html")
            js = read(EXT_ROOT / package / "popup.js") + read(EXT_ROOT / package / "core.js")
            self.assertNotRegex(html, r"https?://")
            self.assertNotRegex(html, r"<(?:script|link)[^>]+(?:src|href)=[\"']//")
            for marker in ("fetch(", "XMLHttpRequest", "WebSocket", "sendBeacon", "analytics", "telemetry"):
                self.assertNotIn(marker, js)

    def test_connection_parser_is_fail_closed_and_strips_sensitive_url_parts(self) -> None:
        for package in PACKAGES:
            core = read(EXT_ROOT / package / "core.js")
            for scheme in ("ftp:", "ftps:", "sftp:"):
                self.assertIn(repr(scheme), core)
            self.assertIn("new URL(", core)
            self.assertIn("parsed.hostname", core)
            self.assertIn("parsed.username", core)
            self.assertIn("parsed.password", core)
            self.assertIn("passwordDetected", core)
            self.assertIn("parsed.host", core)
            self.assertIn("parsed.pathname", core)
            self.assertNotIn("localStorage", core)
            self.assertNotIn("sessionStorage", core)

    def test_safe_target_never_includes_url_credentials_query_or_fragment(self) -> None:
        for package in PACKAGES:
            core = read(EXT_ROOT / package / "core.js")
            self.assertIn("safeTarget", core)
            self.assertNotIn("parsed.username + '@'", core)
            self.assertNotIn("parsed.password + '@'", core)
            self.assertNotIn("parsed.search", core)
            self.assertNotIn("parsed.hash", core)

    def test_chromium_and_firefox_runtime_sources_stay_identical(self) -> None:
        for filename in SHARED_FILES:
            self.assertEqual(
                (EXT_ROOT / "chromium" / filename).read_bytes(),
                (EXT_ROOT / "firefox" / filename).read_bytes(),
                f"browser runtime source drifted: {filename}",
            )

    def test_popup_has_no_inline_script_or_style(self) -> None:
        for package in PACKAGES:
            html = read(EXT_ROOT / package / "popup.html")
            self.assertNotIn("<style", html.lower())
            self.assertNotRegex(html.lower(), r"<script(?![^>]*\bsrc=)")
            self.assertIn('href="popup.css"', html)
            self.assertIn('src="core.js"', html)
            self.assertIn('src="popup.js"', html)

    def test_privacy_documentation_is_explicit(self) -> None:
        readme = read(EXT_ROOT / "README.md").lower()
        for statement in (
            "no telemetry",
            "no tracking",
            "no remote code",
            "does not store",
            "does not read the active tab",
            "does not connect to your ftp, ftps, or sftp server",
        ):
            self.assertIn(statement, readme)


if __name__ == "__main__":
    unittest.main()
