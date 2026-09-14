from pathlib import Path
import re
import unittest


ROOT = Path(__file__).resolve().parents[1]
VERSION_HEADER_RE = re.compile(
    r"^(?:Version|Verzija)\s+\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?\s*$",
    re.MULTILINE,
)


class ProprietaryLicenseContractTests(unittest.TestCase):
    def test_license_is_version_independent_and_proprietary(self) -> None:
        license_text = (ROOT / "LICENSE").read_text(encoding="utf-8")

        self.assertTrue(
            license_text.startswith("GHOST FTP COMMERCIAL PROPRIETARY SOFTWARE LICENSE\n")
        )
        for marker in (
            "Copyright (c) 2026 ",
            "All rights reserved.",
            "proprietary, source-available software",
            "It is not open-source software",
            "not licensed under an OSI-approved open-source license",
            "LIMITED LICENSE TO USE OFFICIAL BUILDS",
            "PUBLIC GITHUB REPOSITORY",
            "PROHIBITED USES WITHOUT PRIOR WRITTEN PERMISSION",
            "SECURITY RESEARCH, INTEROPERABILITY, AND NON-WAIVABLE RIGHTS",
            "OFFICIAL BUILDS, DISTRIBUTION, AND PUBLISHER IDENTITY",
            "THIRD-PARTY COMPONENTS",
            "DISCLAIMER OF WARRANTIES",
            "LIMITATION OF LIABILITY",
            "COMMERCIAL, OEM, REDISTRIBUTION, AND OTHER SPECIAL RIGHTS",
            "Licensing contact:",
            "Official company website:",
            "Ghost FTP website: https://ghostftp.com",
        ):
            self.assertIn(marker, license_text)

        self.assertIsNone(
            VERSION_HEADER_RE.search(license_text),
            "The controlling proprietary license must remain version-independent.",
        )

    def test_readme_points_to_controlling_proprietary_license(self) -> None:
        readme = (ROOT / "README.md").read_text(encoding="utf-8")

        self.assertIn("## Commercial proprietary license", readme)
        self.assertIn("Ghost FTP is **not open-source software**", readme)
        self.assertIn("[`LICENSE`](LICENSE)", readme)


if __name__ == "__main__":
    unittest.main()
