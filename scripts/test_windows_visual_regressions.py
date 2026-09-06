import pathlib
import unittest


ROOT = pathlib.Path(__file__).resolve().parents[1]


class WindowsVisualRegressionTests(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_workspace_refinement_does_not_erase_entire_parent(self):
        source = self.read("internal/desktop/workspace_layout_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", source)
        self.assertIn("a.stabilizeWorkspaceChrome()", source)

    def test_workspace_uses_real_pe_brand_icon(self):
        source = self.read("internal/desktop/chrome_windows.go")
        self.assertIn('loadImageW.Call(hinst, 1, imageIcon', source)
        self.assertIn('createWindowExW.Call(', source)
        self.assertIn('wstr("STATIC")', source)
        self.assertIn('stmSetImage', source)
        self.assertIn('a.move(a.brandTitle, 54, 10, 106, 35)', source)

    def test_disconnected_remote_list_keeps_dark_enabled_surface(self):
        source = self.read("internal/desktop/chrome_windows.go")
        self.assertIn("setControlEnabled(a.remoteList, true)", source)
        self.assertIn("styleWorkspaceList(list)", source)

    def test_startup_settings_skip_redundant_language_rebuild(self):
        source = self.read("internal/desktop/settings_windows.go")
        self.assertIn("previousLanguage := a.languageCode()", source)
        self.assertIn("if a.languageCode() != previousLanguage", source)

    def test_x86_ftp_transport_has_secure_sysnative_fallback(self):
        source = self.read("internal/remote/tools.go")
        self.assertIn('arch == "386"', source)
        self.assertIn('"Sysnative", "curl.exe"', source)
        self.assertNotIn('exec.LookPath("curl.exe")', source)


if __name__ == "__main__":
    unittest.main()
