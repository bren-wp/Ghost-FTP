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
    def test_release_version_remains_007(self) -> None:
        self.assertEqual(read(ROOT / "VERSION").strip(), "0.0.7")
        self.assertIn('var version = "0.0.7"', read(ROOT / "cmd" / "ghostftp" / "main.go"))
        self.assertIn('var version = "0.0.7"', read(ROOT / "cmd" / "installer" / "main.go"))

    def test_extension_payload_is_explicit_and_non_secret(self) -> None:
        core = read(SHARED / "core.js")
        start = core.index("function buildDesktopLaunchURL")
        builder = core[start:core.index("root.GhostFTPConnection", start)]

        self.assertIn("//connect?", builder)
        for allowed in ("protocol", "host", "port", "username", "path"):
            self.assertIn(f"params.set('{allowed}'", builder)
        for forbidden in ("password", "passphrase", "privatekey", "token", "parsed.search", "parsed.hash", "parsed.href"):
            self.assertNotIn(forbidden, builder.lower())
        self.assertNotIn("browser.storage", core)
        self.assertNotIn("localStorage", core)
        self.assertNotIn("sessionStorage", core)

    def test_popup_launch_is_explicit_user_action(self) -> None:
        html = read(SHARED / "popup.html")
        js = read(SHARED / "popup.js")
        css = read(SHARED / "popup.css")

        self.assertIn('id="open-desktop"', html)
        self.assertIn("Open in Ghost FTP", html)
        self.assertIn("Credentials must be entered there.", html)
        self.assertIn("openDesktopButton.addEventListener('click'", js)
        self.assertIn("buildDesktopLaunchURL(parsedConnection)", js)
        self.assertIn("window.location.href = launchURL", js)
        self.assertNotIn("min-width: 420px", css)
        self.assertIn("@media (max-width: 360px)", css)
        self.assertIn("@media (prefers-color-scheme: light)", css)

    def test_manifests_gain_no_permissions_or_network_hooks(self) -> None:
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

    def test_desktop_parser_installer_handoff_and_uninstall_share_contract(self) -> None:
        parser = read(ROOT / "internal" / "desktop" / "startup_target.go")
        startup = read(ROOT / "cmd" / "ghostftp" / "launch_init.go")
        handoff = read(ROOT / "internal" / "desktop" / "startup_target_handoff_windows.go")
        main = read(ROOT / "cmd" / "ghostftp" / "main.go")
        installer = read(ROOT / "cmd" / "installer" / "protocol_registration.go")
        uninstall = read(ROOT / "cmd" / "ghostftp" / "uninstall_mode_windows.go")

        self.assertIn('strings.EqualFold(u.Scheme, "ghostftp")', parser)
        self.assertIn('strings.EqualFold(u.Hostname(), "connect")', parser)
        self.assertIn('strings.EqualFold(u.Hostname(), "open")', parser)
        self.assertIn('u.User != nil', parser)
        self.assertIn('u.Fragment != ""', parser)
        self.assertIn("security.ValidateHost(host)", parser)
        self.assertIn("security.ValidateRemotePath(path)", parser)
        self.assertIn("url.ParseQuery(u.RawQuery)", parser)
        for forbidden in ('case "password"', 'case "passphrase"', 'case "token"'):
            self.assertNotIn(forbidden, parser)

        self.assertIn("desktop.SetStartupTarget", startup)
        self.assertIn("desktop.ForwardStartupTarget(desktopLaunchURI)", main)
        self.assertIn("desktop.StartStartupTargetReceiver()", main)
        self.assertIn("WM_COPYDATA", handoff)
        self.assertIn("ParseStartupTarget(string(payload))", handoff)
        self.assertIn("smtoAbortIfHung", handoff)
        self.assertNotIn("net.Listen", handoff)
        self.assertNotIn("os.WriteFile", handoff)
        self.assertIn(r"Software\Classes\ghostftp".replace("\\\\", "\\"), installer)
        self.assertIn('"URL Protocol"', installer)
        self.assertIn('`" "%1"`', installer)
        self.assertIn("RemoveOwnedGhostFTPProtocolRegistration", uninstall)

    def test_removed_web_runtime_stays_removed(self) -> None:
        for relative in ("web", "web-ftp", "webftp"):
            self.assertFalse((ROOT / relative).exists(), relative)


if __name__ == "__main__":
    unittest.main()
