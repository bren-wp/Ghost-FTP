import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class ProductSurfaceContractTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_android_runtime_copy_hides_implementation_language(self) -> None:
        source = self.read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        forbidden = (
            "Storage Access Framework",
            "scoped storage",
            "Fresh MLSD",
            "runtime owner",
            "canonical dark brand palette",
            "Repository build",
            "development package",
            "production-signature evidence",
            "site JSON/preferences",
            "SAF capability URI",
            "persistent SAF permission",
            "staged local document",
            "staged data",
            "final-name commit",
            "Storage provider",
            "fake history",
            "visible commit",
        )
        for phrase in forbidden:
            self.assertNotIn(phrase, source, f"Android public surface leaked implementation wording: {phrase}")
        self.assertNotIn('putString("password"', source)
        self.assertNotIn('putString("passphrase"', source)
        self.assertNotIn('infoLine("Package", BuildConfig.APPLICATION_ID)', source)
        self.assertIn('preferences.getString("appearance", GhostTheme.APPEARANCE_DARK)', source)
        self.assertIn("GhostTheme.apply(this, appearanceMode);", source)
        self.assertIn("GhostTheme.applySystemBars(this);", source)
        self.assertIn('surfaceHeading("Files", "Browse local files and your connected server from one workspace.")', source)
        self.assertIn('infoLine("Data collection", "No telemetry, analytics or ads")', source)

    def test_secondary_light_themes_remain_off_white_and_neutral(self) -> None:
        desktop = self.read("internal/uipalette/palette.go")
        android = self.read("android/app/src/main/java/app/ghostftp/client/GhostTheme.java")
        browser = self.read("extensions/shared/popup.css")

        self.assertRegex(desktop, r"Window:\s+RGB\{0xEE, 0xF1, 0xF5\}")
        self.assertRegex(desktop, r"Panel:\s+RGB\{0xF6, 0xF8, 0xFB\}")
        self.assertIn("WINDOW = Color.rgb(0xEE, 0xF1, 0xF5);", android)
        self.assertIn("PANEL = Color.rgb(0xF6, 0xF8, 0xFB);", android)
        self.assertIn("ACCENT = Color.rgb(0xA6, 0x65, 0x00);", android)
        self.assertNotIn("WINDOW = Color.rgb(0xFF, 0xFF, 0xFF);", android)
        self.assertNotIn("PANEL = Color.rgb(0xFF, 0xFF, 0xFF);", android)
        self.assertIn("--bg: #eef1f5;", browser)
        self.assertIn("--panel: #f6f8fb;", browser)
        self.assertNotIn("--bg: #ffffff;", browser.lower())
        self.assertNotIn("--panel: #ffffff;", browser.lower())

    def test_browser_helper_remains_local_and_permissionless(self) -> None:
        popup = self.read("extensions/shared/popup.html")
        self.assertIn("Local only · No telemetry · No server connection", popup)
        self.assertNotIn("debug", popup.lower())
        self.assertNotIn("developer", popup.lower())
        for browser in ("chrome", "edge", "firefox", "opera"):
            manifest = json.loads(self.read(f"extensions/{browser}/manifest.json"))
            self.assertEqual([], manifest.get("permissions", []), f"{browser} requested browser permissions")
            self.assertEqual([], manifest.get("host_permissions", []), f"{browser} requested host permissions")

    def test_retired_web_surfaces_are_absent(self) -> None:
        retired = (
            "web",
            ".github/workflows/web.yml",
            "docs/WEB.md",
            "scripts/check_web_contract.py",
            "scripts/test_web_contract.py",
            "docs/prompts/GHOST-FTP-WEB-APP-PROMPT.md",
            "docs/prompts/GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md",
        )
        for relative in retired:
            self.assertFalse((ROOT / relative).exists(), f"retired web surface is still tracked: {relative}")

    def test_desktop_navigation_uses_product_language(self) -> None:
        labels = self.read("internal/desktop/navigation_labels.go")
        self.assertIn('"Connection info"', labels)
        self.assertNotIn('"Diagnostics"', labels)


if __name__ == "__main__":
    unittest.main()
