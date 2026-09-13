#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        raise AssertionError(f"missing required macOS parity file: {relative}")
    return path.read_text(encoding="utf-8")


def retired_roots_declaration(source: str) -> str:
    return source.split("RETIRED_ROOTS =", 1)[1].split(")", 1)[0]


WINDOWS_PARITY_ACTIONS = (
    "Connect",
    "Disconnect",
    "Site Manager",
    "Bookmarks",
    "Private Key",
    "Save Profile",
    "Remove Profile",
    "Settings",
    "About",
    "Diagnostics",
    "Local Refresh",
    "Local Choose Folder",
    "Local Up",
    "Local New Folder",
    "Local Rename",
    "Local Delete",
    "Local Filter",
    "Local Recursive Search",
    "Remote Refresh",
    "Remote Up",
    "Remote New Folder",
    "Remote Rename",
    "Remote Delete",
    "Remote Permissions",
    "Remote Edit",
    "Remote Filter",
    "Remote Recursive Search",
    "Directory Compare",
    "Upload",
    "Download",
    "Pause Queue",
    "Resume Queue",
    "Cancel Transfer",
    "Retry Transfer",
    "Clear Finished",
    "Move Top",
    "Move Up",
    "Move Down",
    "Move Bottom",
)

IMPLEMENTED_MACOS_ACTIONS = set(WINDOWS_PARITY_ACTIONS)


class MacOSWindowsParityContractTests(unittest.TestCase):
    def test_macos_is_an_active_separate_native_surface(self) -> None:
        platform_audit = read("scripts/audit_platform_contract.py")
        desktop_audit = read("scripts/audit_desktop_surface.py")
        for source in (platform_audit, desktop_audit):
            self.assertIn("WINDOWS,LINUX,ANDROID,MACOS", source)
            self.assertNotIn("macos", retired_roots_declaration(source).lower())
            self.assertNotIn("IOS,MACOS", source)
        self.assertNotIn("DARWIN_SOURCE=BLOCKED", platform_audit)

    def test_macos_has_source_development_and_production_distribution_surfaces(self) -> None:
        for relative in (
            "macos/README.md",
            "macos/PARITY.md",
            "macos/BUILD.sh",
            "macos/SIGN_AND_NOTARIZE.sh",
            ".github/workflows/macos-app.yml",
            ".github/workflows/macos-production.yml",
            "scripts/test_macos_distribution_contract.py",
        ):
            self.assertTrue((ROOT / relative).is_file(), relative)

        workflow = read(".github/workflows/macos-app.yml")
        self.assertIn("runs-on: macos-", workflow)
        self.assertIn("bash macos/BUILD.sh", workflow)
        self.assertIn("ghostftp-macos-development", workflow)
        self.assertIn("permissions:\n  contents: read", workflow)

        production = read(".github/workflows/macos-production.yml")
        self.assertIn("workflow_dispatch:", production)
        self.assertIn("environment: macos-production", production)
        self.assertIn("bash macos/SIGN_AND_NOTARIZE.sh", production)
        self.assertIn("permissions:\n  contents: read", production)

    def test_windows_is_the_visual_and_behavior_reference(self) -> None:
        readme = read("macos/README.md")
        parity = read("macos/PARITY.md")
        combined = readme + "\n" + parity
        for marker in (
            "Windows desktop remains the canonical visual and behavior reference",
            "same typed `internal/api.Engine`",
            "no decorative or dead controls",
            "Classic Light",
            "#EEF1F5",
            "#F6F8FB",
            "#FAFBFD",
            "Dark",
            "#0B0F17",
            "#121824",
            "#161D2A",
            "24 languages",
            "FTP",
            "FTPS",
            "SFTP",
            "no telemetry",
        ):
            self.assertIn(marker, combined)

    def test_complete_windows_action_inventory_is_recorded_truthfully(self) -> None:
        parity = read("macos/PARITY.md")
        self.assertEqual(set(WINDOWS_PARITY_ACTIONS), IMPLEMENTED_MACOS_ACTIONS)
        for action in WINDOWS_PARITY_ACTIONS:
            self.assertIn(f"- [x] {action}", parity)
            self.assertNotIn(f"- [ ] {action}", parity)

    def test_production_preparation_does_not_silently_publish_a_public_release(self) -> None:
        readme = read("macos/README.md")
        self.assertIn("Source/native functionality is complete", readme)
        self.assertIn("Developer ID", readme)
        self.assertIn("Apple notarization", readme)
        self.assertIn("does **not** modify or upload to an existing public GitHub Release", readme)

        release = read(".github/workflows/release.yml")
        self.assertNotIn("macOS-notarized.app.zip", release)
        self.assertNotIn("SIGN_AND_NOTARIZE.sh", release)

    def test_build_contract_binds_to_root_version_and_app_bundle(self) -> None:
        build = read("macos/BUILD.sh")
        self.assertIn("../VERSION", build)
        self.assertIn("Ghost FTP.app", build)
        self.assertIn("CFBundleShortVersionString", build)
        self.assertIn("CFBundleVersion", build)
        self.assertIn("app.ghostftp.client", build)
        self.assertIn("Ghost-FTP-${VERSION}-macOS.app.zip", build)
        self.assertNotIn("curl ", build)
        self.assertNotIn("wget ", build)

    def test_distribution_contract_is_separate_and_fail_closed(self) -> None:
        signer = read("macos/SIGN_AND_NOTARIZE.sh")
        self.assertIn("Developer ID Application:", signer)
        self.assertIn("--options runtime", signer)
        self.assertIn("--timestamp", signer)
        self.assertIn("xcrun notarytool submit", signer)
        self.assertIn("xcrun stapler staple", signer)
        self.assertIn("spctl --assess", signer)
        self.assertIn("accepted-stapled", signer)


if __name__ == "__main__":
    unittest.main()
