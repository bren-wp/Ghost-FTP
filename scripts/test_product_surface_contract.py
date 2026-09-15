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
            "Repository build",
            "development package",
            "production-signature evidence",
            "site JSON/preferences",
            "SAF capability URI",
            "persistent SAF permission",
            "staged local document",
            "final-name commit",
            "fake history",
            "visible commit",
        )
        for phrase in forbidden:
            self.assertNotIn(phrase, source, f"Android public surface leaked implementation wording: {phrase}")
        self.assertNotIn('infoLine("Package", BuildConfig.APPLICATION_ID)', source)
        self.assertIn("GhostTheme.apply(this);", source)
        self.assertIn("GhostTheme.applySystemBars(this);", source)
        self.assertIn('surfaceHeading("Files", "Browse local files and your connected server from one workspace.")', source)
        self.assertIn('infoLine("Data collection", "No telemetry, analytics or ads")', source)

    def test_soft_light_theme_is_consistent_on_active_surfaces(self) -> None:
        desktop = self.read("internal/uipalette/palette.go")
        android = self.read("android/app/src/main/java/app/ghostftp/client/GhostTheme.java")
        browser = self.read("extensions/shared/popup.css")

        self.assertIn("Window: RGB{0xEE, 0xF1, 0xF5}", desktop)
        self.assertIn("Panel:  RGB{0xF6, 0xF8, 0xFB}", desktop)
        self.assertIn("WINDOW = Color.rgb(0xEE, 0xF1, 0xF5);", android)
        self.assertIn("PANEL = Color.rgb(0xF6, 0xF8, 0xFB);", android)
        self.assertIn("--bg: #eef1f5;", browser)
        self.assertIn("--panel: #f6f8fb;", browser)
        self.assertNotIn("--bg: #ffffff;", browser.lower())
        self.assertNotIn("--panel: #ffffff;", browser.lower())

    def test_browser_surface_has_no_dev_or_fake_copy(self) -> None:
        popup = self.read("extensions/shared/popup.html")
        source = (popup + self.read("extensions/shared/popup.js") + self.read("extensions/shared/core.js")).lower()
        for forbidden in (
            "debug mode",
            "developer build",
            "demo credentials",
            "fake connection",
            "simulated connection",
            "dummy data",
            "test server",
        ):
            self.assertNotIn(forbidden, source)
        for browser in ("chrome", "edge", "firefox", "opera"):
            manifest = json.loads(self.read(f"extensions/{browser}/manifest.json"))
            self.assertNotIn("<all_urls>", manifest.get("host_permissions", []))

    def test_retired_web_project_is_absent(self) -> None:
        self.assertFalse((ROOT / "web").exists())
        self.assertFalse((ROOT / "docs" / "WEB.md").exists())
        self.assertFalse((ROOT / "scripts" / "check_web_contract.py").exists())
        self.assertFalse((ROOT / "scripts" / "test_web_contract.py").exists())
        self.assertFalse((ROOT / ".github" / "workflows" / "web.yml").exists())

    def test_desktop_navigation_uses_product_language(self) -> None:
        labels = self.read("internal/desktop/navigation_labels.go")
        self.assertIn('"Connection info"', labels)
        self.assertNotIn('"Diagnostics"', labels)


if __name__ == "__main__":
    unittest.main()
