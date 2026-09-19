from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[1]
PROMPTS = ROOT / "docs" / "prompts"


class EngineeringPromptCurrentContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.engineering = (PROMPTS / "GHOST-FTP-ENGINEERING-AUDIT-PROMPT.md").read_text(
            encoding="utf-8"
        )

    def test_engineering_prompt_tracks_current_platform_and_release_boundaries(self) -> None:
        for marker in (
            "Windows — public production surface",
            "Linux — public production surface",
            "Android — public production surface",
            "Browser helper — public production surface",
            "macOS — retired",
            "Do not reintroduce them as part of ordinary parity work.",
            "13 platform artifacts plus 3 metadata files = 16 public release files",
            "`minSdk 26`, `targetSdk 35`",
            "Android SFTP remains hidden/unsupported",
            "zero browser permissions and zero host permissions",
            "GHOSTFTP_ANDROID_CERT_SHA256",
        ):
            self.assertIn(marker, self.engineering)

        self.assertNotIn("maintained application platforms are **Windows and Linux**", self.engineering)
        self.assertNotIn("18 platform artifacts", self.engineering)
        self.assertNotIn("21 public release files", self.engineering)
        self.assertNotIn("Developer ID Application", self.engineering)
        self.assertNotIn("Apple notarization", self.engineering)
        self.assertNotIn("0.0.6", self.engineering)

    def test_retired_web_prompts_are_absent(self) -> None:
        self.assertFalse((PROMPTS / "GHOST-FTP-WEB-APP-PROMPT.md").exists())
        self.assertFalse((PROMPTS / "GHOSTFTP-COM-DARK-THEME-REDESIGN-PROMPT.md").exists())


if __name__ == "__main__":
    unittest.main()
