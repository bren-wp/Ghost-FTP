import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class AboutCardReleaseRegressionTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_about_uses_public_product_name(self):
        source = self.read("internal/desktop/settings_windows.go")
        self.assertIn(
            'strings.ReplaceAll(a.tr("about.body", brand.Website, brand.Support), "GhostFTP", brand.ProductName)',
            source,
        )

    def test_about_card_reserves_multiline_heading_space(self):
        source = self.read("internal/platform/info_card_windows.go")
        self.assertIn("windowWidth  = 760", source)
        self.assertIn("windowHeight = 460", source)
        self.assertIn('makeControl("STATIC", heading, 0, 40, 26, 680, 92', source)
        self.assertNotIn('makeControl("STATIC", heading, 0, 38, 28, 584, 42', source)


if __name__ == "__main__":
    unittest.main()
