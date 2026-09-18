#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class UpdateAndPremiumContractTests(unittest.TestCase):
    def test_shared_update_checker_is_manual_bounded_and_fail_closed(self) -> None:
        brand = read("internal/brand/brand.go")
        checker = read("internal/updatecheck/updatecheck.go")
        external = read("internal/external/open.go")

        for marker in (
            'ReleaseAPIURL     = "https://api.github.com/repos/bren-wp/Ghost-FTP/releases/latest"',
            'ReleaseURL        = "https://github.com/bren-wp/Ghost-FTP/releases/latest"',
            'PremiumURL        = "https://ghostftp.com/premium/"',
        ):
            self.assertIn(marker, brand)

        for marker in (
            "Timeout: 10 * time.Second",
            "io.LimitReader(resp.Body, maxResponseBytes)",
            "payload.Draft || payload.Prerelease",
            'host != "github.com"',
            '"/bren-wp/Ghost-FTP/releases/"',
            "parsed.RawQuery = \"\"",
            "parsed.Fragment = \"\"",
        ):
            self.assertIn(marker, checker)

        for marker in (
            'parsed.Scheme != "https"',
            "parsed.User != nil",
            'case "windows":',
            'case "darwin":',
            'case "linux":',
        ):
            self.assertIn(marker, external)

    def test_windows_exposes_real_update_and_premium_commands(self) -> None:
        navigation = read("internal/desktop/navigation_windows.go")
        commands = read("internal/desktop/commands_windows.go")
        rail = read("internal/desktop/sidebar_windows.go")
        update = read("internal/desktop/update_windows.go")
        for marker in (
            "idCheckUpdates     = 704",
            "idPremiumDownload  = 705",
        ):
            self.assertIn(marker, navigation)
        self.assertIn("a.checkForUpdates()", commands)
        self.assertIn("a.openPremiumDownload()", commands)
        self.assertIn('"Check for updates"', rail)
        self.assertIn('"Premium"', rail)
        self.assertIn("updatecheck.New().Check(ctx, a.version)", update)
        self.assertIn("external.OpenReleasePage(result.ReleaseURL)", update)
        self.assertIn("external.OpenPremiumPage()", update)
        self.assertNotIn("checkForUpdates()", read("internal/desktop/windows.go"))

    def test_linux_update_and_premium_are_user_initiated(self) -> None:
        rail = read("internal/desktop/linux_master_rail.go")
        update = read("internal/desktop/update_linux.go")
        gui = read("internal/desktop/gui_linux.go")
        self.assertIn('"Check for updates"', rail)
        self.assertIn('"Premium"', rail)
        self.assertIn("u.checkForUpdates()", rail)
        self.assertIn("u.openPremiumDownload()", rail)
        self.assertIn("linuxActionUpdateCheck", gui)
        self.assertIn("updatecheck.New().Check(ctx, u.version)", update)
        self.assertIn("external.OpenPremiumPage()", update)
        constructor = gui[gui.index("func newLinuxDesktop("):gui.index("func linuxTrimForUI")]
        self.assertNotIn("checkForUpdates()", constructor)

    def test_android_update_is_explicit_and_does_not_include_connection_data(self) -> None:
        activity = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        settings = activity[activity.index("private View buildSettingsSurface()"):activity.index("private View buildConnectionInfoSurface()")]
        for marker in (
            '"UPDATES & PREMIUM"',
            'button("Check for updates")',
            'primaryButton("Download Premium")',
            "checkForUpdates()",
            "openTrustedWebPage(PREMIUM_URL, \"ghostftp.com\")",
            "connection.setConnectTimeout(8000)",
            "connection.setReadTimeout(8000)",
            "connection.setInstanceFollowRedirects(false)",
            'path.startsWith("/bren-wp/Ghost-FTP/releases/")',
        ):
            self.assertIn(marker, activity)
        self.assertIn("Credentials, paths and transfer data are never sent.", settings)
        create = activity[activity.index("protected void onCreate(Bundle state)"):activity.index("protected void onDestroy()")]
        self.assertNotIn("checkForUpdates()", create)
        fetch_start = activity.index("private static UpdateInfo fetchLatestRelease()")
        fetch = activity[fetch_start:activity.index("private static int[] parseVersion", fetch_start)]
        for forbidden in ("host.getText()", "username.getText()", "password.getText()", "currentRemotePath", "currentDocumentId"):
            self.assertNotIn(forbidden, fetch)

    def test_macos_update_and_premium_use_bridge_and_background_queue(self) -> None:
        bridge = read("macos/Bridge/application.go")
        windows = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")
        for marker in (
            "//export GhostFTPCheckForUpdates",
            "updatecheck.New().Check(ctx, productVersion)",
            "//export GhostFTPOpenReleasePage",
            "//export GhostFTPOpenPremiumPage",
            "external.OpenReleasePage",
            "external.OpenPremiumPage",
        ):
            self.assertIn(marker, bridge)
        about = windows[windows.index("final class AboutWindowController"):windows.index("final class DiagnosticsWindowController")]
        for marker in (
            'NSButton(title: "Check for Updates"',
            'NSButton(title: "Download Premium"',
            'DispatchQueue(label: "app.ghostftp.update-check"',
            "GhostFTPCheckForUpdates()",
            "GhostFTPOpenReleasePage",
            "GhostFTPOpenPremiumPage",
        ):
            self.assertIn(marker, about)

    def test_public_release_requires_notarized_macos_and_17_files(self) -> None:
        release = read(".github/workflows/release.yml")
        retention = read(".github/workflows/release-retention.yml")
        digest = read("scripts/verify_release_digest_readback.py")
        for marker in (
            "needs: [quality, windows, linux, android, macos, browser]",
            "environment: macos-production",
            "bash macos/SIGN_AND_NOTARIZE.sh",
            "Ghost-FTP-${VERSION}-macOS-notarized.app.zip",
            "PUBLIC_PLATFORM_ARTIFACTS=14",
            "PUBLIC_RELEASE_FILES=17",
            "MACOS_RELEASE_ARTIFACT_VERIFIED=PASS",
        ):
            self.assertIn(marker, release)
        self.assertIn('test "$asset_count" -eq 17', retention)
        self.assertIn("EXPECTED_RELEASE_FILES = 17", digest)
        self.assertIn('f"Ghost-FTP-{version}-macOS-notarized.app.zip"', digest)


if __name__ == "__main__":
    unittest.main()
