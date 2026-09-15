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
        self.assertNotIn('infoLine("Package", BuildConfig.APPLICATION_ID)', source)
        self.assertIn("GhostTheme.apply(this);", source)
        self.assertIn("GhostTheme.applySystemBars(this);", source)
        self.assertIn('surfaceHeading("Files", "Browse local files and your connected server from one workspace.")', source)
        self.assertIn('infoLine("Data collection", "No telemetry, analytics or ads")', source)

    def test_dirty_off_white_light_theme_is_consistent(self) -> None:
        desktop = self.read("internal/uipalette/palette.go")
        android = self.read("android/app/src/main/java/app/ghostftp/client/GhostTheme.java")
        browser = self.read("extensions/shared/popup.css")
        site = self.read("web/assets/css/site.css")
        web_app = self.read("web/ftp/assets/app.css")

        self.assertIn("Window: RGB{0xEE, 0xF1, 0xF5}", desktop)
        self.assertIn("Panel:  RGB{0xF6, 0xF8, 0xFB}", desktop)
        self.assertIn("WINDOW = Color.rgb(0xEE, 0xF1, 0xF5);", android)
        self.assertIn("PANEL = Color.rgb(0xF6, 0xF8, 0xFB);", android)
        for css in (browser, site, web_app):
            self.assertIn("--bg: #eef1f5;", css)
            self.assertIn("--panel: #f6f8fb;", css)
            self.assertNotIn("--bg: #ffffff;", css.lower())
            self.assertNotIn("--panel: #ffffff;", css.lower())

    def test_browser_helper_remains_local_and_permissionless(self) -> None:
        popup = self.read("extensions/shared/popup.html")
        self.assertIn("Local only · No telemetry · No server connection", popup)
        self.assertNotIn("debug", popup.lower())
        self.assertNotIn("developer", popup.lower())
        for browser in ("chrome", "edge", "firefox", "opera"):
            manifest = json.loads(self.read(f"extensions/{browser}/manifest.json"))
            self.assertEqual([], manifest.get("permissions", []), f"{browser} requested browser permissions")
            self.assertEqual([], manifest.get("host_permissions", []), f"{browser} requested host permissions")

    def test_web_public_pages_do_not_expose_build_scaffolding(self) -> None:
        public_files = (
            "web/index.html",
            "web/download.html",
            "web/ftp/index.php",
            "web/ftp/assets/app.js",
        )
        forbidden = (
            "TODO",
            "FIXME",
            "development package",
            "repository build",
            "source commit",
            "stack trace",
        )
        for relative in public_files:
            source = self.read(relative)
            for phrase in forbidden:
                self.assertNotIn(phrase.lower(), source.lower(), f"{relative} leaked development wording: {phrase}")

    def test_desktop_navigation_uses_product_language(self) -> None:
        labels = self.read("internal/desktop/navigation_labels.go")
        self.assertIn('"Connection info"', labels)
        self.assertNotIn('"Diagnostics"', labels)


if __name__ == "__main__":
    unittest.main()
