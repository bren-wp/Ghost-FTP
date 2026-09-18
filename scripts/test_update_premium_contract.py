#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class UpdateAndPremiumContractTests(unittest.TestCase):
    def test_shared_update_flow_is_local_only_and_official_site_only(self) -> None:
        brand = read("internal/brand/brand.go")
        checker = read("internal/updatecheck/updatecheck.go")
        external = read("internal/external/open.go")

        for marker in (
            'WebsiteURL = "https://ghostftp.com/"',
            'UpdateURL  = "https://ghostftp.com/#download"',
            'PremiumURL = "https://ghostftp.com/premium/"',
        ):
            self.assertIn(marker, brand)
        for forbidden in ("https://github.com", "https://api.github.com"):
            self.assertNotIn(forbidden, brand)

        for marker in (
            "func Simulate(currentVersion string)",
            "Simulated:      true",
            "UpdateURL:      brand.UpdateURL",
        ):
            self.assertIn(marker, checker)
        for forbidden in ("net/http", "http.Client", "https://api.github.com", "https://github.com"):
            self.assertNotIn(forbidden, checker)

        for marker in (
            'officialHosts = []string{"ghostftp.com", "www.ghostftp.com"}',
            'parsed.Scheme != "https"',
            "parsed.User != nil",
            "OpenUpdatePage",
            "OpenPremiumPage",
            "OpenWebsite",
        ):
            self.assertIn(marker, external)
        self.assertNotIn("https://github.com", external)

    def test_windows_keeps_language_and_product_actions_in_settings(self) -> None:
        settings = read("internal/desktop/settings_windows.go")
        dialog = read("internal/platform/settings_dialog_windows.go")
        rail = read("internal/desktop/sidebar_windows.go")
        update = read("internal/desktop/update_windows.go")

        for marker in (
            "LanguageOptions:   languageOptions",
            'UpdateLabel:            "Update"',
            'DownloadLabel:          "Download latest"',
            'PremiumLabel:           "Premium"',
            'WebsiteLabel:           "Official website"',
            'case "update":',
            "a.checkForUpdates()",
            "a.openUpdateDownload()",
            "a.openPremiumDownload()",
            "a.openOfficialWebsite()",
        ):
            self.assertIn(marker, settings)
        for marker in (
            "settingsIDLanguage",
            "settingsIDUpdate",
            "settingsIDDownload",
            "settingsIDPremium",
            "settingsIDWebsite",
            "LanguageIndex",
        ):
            self.assertIn(marker, dialog)
        self.assertIn("showControls(false, a.languageCombo)", rail)
        self.assertNotIn('a.setSidebarButtonVisual(updates', rail)
        self.assertNotIn('a.setSidebarButtonVisual(premium', rail)

        self.assertIn("updatecheck.Simulate(a.version)", update)
        self.assertIn("external.OpenUpdatePage()", update)
        self.assertIn("external.OpenPremiumPage()", update)
        self.assertIn("external.OpenWebsite()", update)
        self.assertNotIn("GitHub", update)

    def test_linux_keeps_language_and_product_actions_in_settings(self) -> None:
        rail = read("internal/desktop/linux_master_rail.go")
        settings = read("internal/desktop/gui_linux_settings.go")
        update = read("internal/desktop/update_linux.go")
        gui = read("internal/desktop/gui_linux.go")

        self.assertNotIn("rail.language", rail)
        self.assertNotIn("rail.updates", rail)
        self.assertNotIn("rail.premium", rail)
        for marker in (
            "u.settingsRects.language",
            "u.settingsRects.update",
            "u.settingsRects.download",
            "u.settingsRects.premium",
            "u.settingsRects.website",
            '"Update"',
            '"Download latest"',
            '"Premium"',
            '"Official website"',
        ):
            self.assertIn(marker, settings)
        self.assertIn("linuxActionUpdateCheck", gui)
        self.assertIn("updatecheck.Simulate(u.version)", update)
        self.assertIn("external.OpenUpdatePage()", update)
        self.assertIn("external.OpenPremiumPage()", update)
        self.assertIn("external.OpenWebsite()", update)
        self.assertNotIn("GitHub", update)

    def test_android_update_is_local_simulation_and_official_site_only(self) -> None:
        activity = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        settings = activity[activity.index("private View buildSettingsSurface()"):activity.index("private View buildConnectionInfoSurface()")]

        for marker in (
            'private static final String WEBSITE_URL = "https://ghostftp.com/";',
            'private static final String UPDATE_URL = "https://ghostftp.com/#download";',
            'private static final String PREMIUM_URL = "https://ghostftp.com/premium/";',
            'primaryButton("Update")',
            'button("Download latest")',
            'button("Download Premium")',
            'button("Official website")',
            "simulateUpdate()",
            "openTrustedWebPage(UPDATE_URL)",
            "openTrustedWebPage(PREMIUM_URL)",
            "openTrustedWebPage(WEBSITE_URL)",
        ):
            self.assertIn(marker, activity)
        self.assertIn("Update simulation runs locally.", settings)
        for forbidden in (
            "api.github.com",
            "github.com/",
            "HttpURLConnection",
            "fetchLatestRelease",
            "JSONObject",
        ):
            self.assertNotIn(forbidden, activity)

    def test_macos_keeps_language_and_product_actions_in_settings(self) -> None:
        bridge = read("macos/Bridge/application.go")
        windows = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")

        for marker in (
            "//export GhostFTPOpenUpdatePage",
            "//export GhostFTPOpenPremiumPage",
            "//export GhostFTPOpenWebsite",
            "external.OpenUpdatePage",
            "external.OpenPremiumPage",
            "external.OpenWebsite",
        ):
            self.assertIn(marker, bridge)
        self.assertNotIn("GhostFTPOpenReleasePage", bridge)
        self.assertNotIn("updatecheck.New().Check", bridge)

        settings = windows[windows.index("final class SettingsWindowController"):windows.index("final class AboutWindowController")]
        for marker in (
            'NSButton(title: "Update"',
            'NSButton(title: "Download latest"',
            'NSButton(title: "Premium"',
            'NSButton(title: "Official website"',
            "GhostFTPOpenUpdatePage()",
            "GhostFTPOpenPremiumPage()",
            "GhostFTPOpenWebsite()",
            "DispatchQueue.main.asyncAfter",
        ):
            self.assertIn(marker, settings)
        about = windows[windows.index("final class AboutWindowController"):windows.index("final class DiagnosticsWindowController")]
        self.assertNotIn("Check for Updates", about)
        self.assertNotIn("Download Premium", about)

    def test_android_brand_mark_matches_reference_gold_ghost(self) -> None:
        for relative in (
            "android/app/src/main/res/drawable/ic_ghost_brand.xml",
            "android/app/src/main/res/drawable/ic_ghostftp.xml",
        ):
            icon = read(relative)
            for marker in ("#0B0F17", "#F6C445", "#10131A"):
                self.assertIn(marker, icon)
            for retired in ("#46D6C8", "#5A86F7", "opposing transfer arrows"):
                self.assertNotIn(retired, icon)

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
