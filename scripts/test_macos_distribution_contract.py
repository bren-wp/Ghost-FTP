#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        raise AssertionError(f"missing required macOS distribution file: {relative}")
    return path.read_text(encoding="utf-8")


class MacOSDistributionContractTests(unittest.TestCase):
    def test_signing_script_is_fail_closed_and_uses_current_apple_distribution_primitives(self) -> None:
        script = read("macos/SIGN_AND_NOTARIZE.sh")
        for marker in (
            "MACOS_DEVELOPER_IDENTITY is required",
            "MACOS_NOTARY_KEYCHAIN_PROFILE is required",
            "MACOS_DEVELOPER_IDENTITY must be a Developer ID Application identity",
            "security find-identity -v -p codesigning",
            "--options runtime",
            "--timestamp",
            "codesign --verify --deep --strict",
            "xcrun notarytool submit",
            "--keychain-profile",
            "--wait",
            "xcrun stapler staple",
            "xcrun stapler validate",
            "spctl --assess --type execute",
            "com.apple.security.get-task-allow",
            "MACOS_PRODUCTION_NOTARIZATION=accepted-stapled",
        ):
            self.assertIn(marker, script)

        # Raw signing/notary credential material belongs outside the reusable
        # production signer. It receives only Keychain references.
        for forbidden in (
            "APPLE_ID_PASSWORD",
            "APP_SPECIFIC_PASSWORD",
            "APPLE_NOTARY_API_KEY_P8",
            "MACOS_DEVELOPER_ID_P12_BASE64",
            "MACOS_DEVELOPER_ID_P12_PASSWORD",
            "altool",
        ):
            self.assertNotIn(forbidden, script)

        # Explicit inside-out signing remains reviewable; never hide nested
        # code discovery behind codesign --deep during signing.
        sign_section = script.split("sign_one()", 1)[1].split("codesign --verify", 1)[0]
        for target in ('sign_one "$ENGINE"', 'sign_one "$ASKPASS"', 'sign_one "$EXECUTABLE"', 'sign_one "$APP"'):
            self.assertIn(target, sign_section)
        self.assertNotIn("--deep", sign_section)

    def test_manual_production_workflow_keeps_credentials_ephemeral(self) -> None:
        workflow = read(".github/workflows/macos-production.yml")
        for marker in (
            "workflow_dispatch:",
            "environment: macos-production",
            "permissions:\n  contents: read",
            "MACOS_DEVELOPER_ID_P12_BASE64",
            "MACOS_DEVELOPER_ID_P12_PASSWORD",
            "MACOS_DEVELOPER_IDENTITY",
            "APPLE_NOTARY_API_KEY_P8",
            "APPLE_NOTARY_API_KEY_ID",
            "APPLE_NOTARY_ISSUER_ID",
            "security create-keychain",
            "security delete-keychain",
            "notarytool store-credentials",
            "bash macos/SIGN_AND_NOTARIZE.sh",
            "Ghost-FTP-*-macOS-notarized.app.zip",
            "MACOS_PRODUCTION_ARTIFACT_VERIFIED=PASS",
            "if: always()",
        ):
            self.assertIn(marker, workflow)

        # Signing/notarization can create a verified artifact but does not
        # silently mutate or publish an existing public GitHub release.
        for forbidden in (
            "gh release",
            "softprops/action-gh-release",
            "contents: write",
            "release.yml",
        ):
            self.assertNotIn(forbidden, workflow)

    def test_development_build_stays_distinct_from_production_distribution(self) -> None:
        build = read("macos/BUILD.sh")
        self.assertIn("MACOS_SIGNING=adhoc-development", build)
        self.assertNotIn("notarytool", build)
        self.assertNotIn("Developer ID Application", build)

        dev_workflow = read(".github/workflows/macos-app.yml")
        self.assertIn("Ghost FTP macOS Development App", dev_workflow)
        self.assertIn("MACOS_APP_SIGNING=adhoc-development", dev_workflow)
        self.assertNotIn("MACOS_DEVELOPER_ID_P12_BASE64", dev_workflow)

    def test_public_release_remains_separate_until_credentialed_artifact_is_explicitly_promoted(self) -> None:
        release = read(".github/workflows/release.yml")
        self.assertNotIn("macOS-notarized.app.zip", release)
        self.assertNotIn("SIGN_AND_NOTARIZE.sh", release)


if __name__ == "__main__":
    unittest.main()
