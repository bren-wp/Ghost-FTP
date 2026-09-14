from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]
PROMPTS = ROOT / "docs" / "prompts"


class EngineeringPromptCurrentContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.engineering = (PROMPTS / "GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md").read_text(
            encoding="utf-8"
        )
        self.website = (PROMPTS / "GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md").read_text(
            encoding="utf-8"
        )

    def test_engineering_prompt_tracks_current_platform_and_release_boundaries(self) -> None:
        for marker in (
            "Windows — public production surface",
            "Linux — public production surface",
            "Android — public production surface",
            "Browser helper — public production surface",
            "macOS — active development/source surface only",
            "13 platform artifacts plus 3 metadata files = 16 public release files",
            "`minSdk 26`, `targetSdk 35`",
            "Android SFTP remains hidden/unsupported",
            "zero browser permissions and zero host permissions",
            "GHOSTFTP_ANDROID_CERT_SHA256",
            "Developer ID Application",
            "Apple notarization",
        ):
            self.assertIn(marker, self.engineering)

        self.assertNotIn("maintained application platforms are **Windows and Linux**", self.engineering)
        self.assertNotIn("18 platform artifacts", self.engineering)
        self.assertNotIn("21 public release files", self.engineering)

    def test_website_prompt_tracks_current_public_surfaces(self) -> None:
        for marker in (
            "Windows:** one universal Setup and one universal Portable",
            "Linux:** six universal distro bundles",
            "Android:** one canonical production-signed APK",
            "Browser helper:** deterministic ZIPs for **Chrome, Edge, Firefox and Opera**",
            "macOS:** active development/source surface only",
            "Android API 26+ (`minSdk 26`)",
            "13 platform artifacts plus 3 metadata files = 16 public release files",
            "Android SFTP is hidden/unsupported",
            "zero browser permissions and zero host permissions",
            "no browser-to-desktop handoff",
            "without separate explicit opt-in consent",
        ):
            self.assertIn(marker, self.website)

        self.assertNotIn("maintained Windows/Linux Ghost FTP application", self.website)
        self.assertNotIn("Maintained desktop platforms: **Windows and Linux**", self.website)
        self.assertNotIn("Windows/Linux availability", self.website)


if __name__ == "__main__":
    unittest.main()
