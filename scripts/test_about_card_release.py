import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class AboutCardReleaseRegressionTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_about_is_the_only_runtime_author_identity_surface(self):
        source = self.read("internal/desktop/settings_windows.go")
        identity = self.read("internal/desktop/about_identity_windows.go")
        brand = self.read("internal/brand/brand.go")

        self.assertIn(
            'strings.ReplaceAll(a.tr("about.body", brand.Website, aboutSupport), "GhostFTP", brand.ProductName)',
            source,
        )
        self.assertIn('aboutPublisher + " · " + aboutAuthorWebsite', source)
        self.assertIn('aboutPublisher     = "BRENDIGO LTD"', identity)
        self.assertIn('aboutAuthorWebsite = "brendigo.com"', identity)
        self.assertIn('aboutSupport       = "brendigo.com/kontakt"', identity)
        self.assertNotIn("BRENDIGO", brand.upper())
        self.assertNotIn("brendigo.com", brand.lower())
        self.assertIn('Website = "ghostftp.com"', brand)
        self.assertIn("Support = Website", brand)

    def test_about_card_reserves_multiline_heading_space(self):
        source = self.read("internal/platform/info_card_windows.go")
        self.assertIn("windowWidth  = 760", source)
        self.assertIn("windowHeight = 460", source)
        self.assertIn('makeControl("STATIC", heading, 0, 40, 26, 680, 92', source)
        self.assertNotIn('makeControl("STATIC", heading, 0, 38, 28, 584, 42', source)


if __name__ == "__main__":
    unittest.main()
