#!/usr/bin/env python3
from __future__ import annotations

import json
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
EXTENSIONS = ROOT / "extensions"
SHARED = EXTENSIONS / "shared"
PACKAGES = ("chrome", "edge", "firefox", "opera")


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


class BrowserDesktopLaunchContractTests(unittest.TestCase):
    def test_release_version_remains_006(self) -> None:
        self.assertEqual(read(ROOT / "VERSION").strip(), "0.0.6")
        self.assertIn('var version = "0.0.6"', read(ROOT / "cmd" / "ghostftp" / "main.go"))
        self.assertIn('var version = "0.0.6"', read(ROOT / "cmd" / "installer" / "main.go"))

    def test_extension_launch_payload_is_explicit_and_non_secret(self) -> None:
        core = read(SHARED / "core.js")
        start = core.index("function buildDesktopLaunchTarget")
        end = core.index("function parseConnectionTarget", start)
        builder = core[start:end]

        self.assertIn("ghostftp://open?", builder)
        for allowed in ("protocol", "host", "port", "username", "path"):
            self.assertIn(f"params.set('{allowed}'", builder)
        for forbidden in ("password", "passphrase", "parsed.search", "parsed.hash", "parsed.href"):
            self.assertNotIn(forbidden, builder.lower())

        self.assertIn("desktopLaunchTarget", core)
        self.assertIn("passwordDetected", core)
        self.assertNotIn("browser.storage", core)
        self.assertNotIn("localStorage", core)
        self.assertNotIn("sessionStorage", core)

    def test_popup_requires_explicit_user_launch_action(self) -> None:
        html = read(SHARED / "popup.html")
        js = read(SHARED / "popup.js")
        css = read(SHARED / "popup.css")

        self.assertIn('id="desktop-launch"', html)
        self.assertIn("Open in Ghost FTP", html)
        self.assertIn("Enter credentials in Ghost FTP", html)
        self.assertIn("desktopLaunch.href = parsed.desktopLaunchTarget", js)
        self.assertIn("desktopLaunch.addEventListener('click'", js)
        self.assertNotIn("min-width: 420px", css)
        self.assertIn("@media (max-width: 360px)", css)
        self.assertIn("@media (prefers-color-scheme: light)", css)

    def test_manifests_do_not_gain_permissions_or_network_access(self) -> None:
        for package in PACKAGES:
            manifest = json.loads(read(EXTENSIONS / package / "manifest.json"))
            self.assertEqual(manifest.get("permissions"), [], package)
            for forbidden in (
                "host_permissions",
                "optional_host_permissions",
                "optional_permissions",
                "background",
                "content_scripts",
                "externally_connectable",
            ):
                self.assertNotIn(forbidden, manifest, package)

    def test_desktop_parser_and_windows_registration_match_extension_contract(self) -> None:
        parser = read(ROOT / "internal" / "desktop" / "launch_target.go")
        installer = read(ROOT / "cmd" / "installer" / "protocol_registration.go")
        startup = read(ROOT / "cmd" / "ghostftp" / "launch_init.go")
        uninstall = read(ROOT / "cmd" / "ghostftp" / "uninstall_mode_windows.go")

        for allowed in ('"protocol": true', '"host":     true', '"port":     true', '"username": true', '"path":     true'):
            self.assertIn(allowed, parser)
        self.assertIn('strings.EqualFold(u.Scheme, "ghostftp")', parser)
        self.assertIn('strings.EqualFold(u.Hostname(), "open")', parser)
        self.assertIn('u.User != nil', parser)
        self.assertIn('u.Fragment != ""', parser)

        self.assertIn(r"Software\Classes\ghostftp", installer)
        self.assertIn('"URL Protocol"', installer)
        self.assertIn('`" "%1"`', installer)
        self.assertIn("ConfigureInitialLaunchTarget", startup)
        self.assertIn("RemoveOwnedGhostFTPProtocolRegistration", uninstall)

    def test_removed_web_runtime_stays_removed(self) -> None:
        for relative in ("web", "web-ftp", "webftp"):
            self.assertFalse((ROOT / relative).exists(), relative)


if __name__ == "__main__":
    unittest.main()
