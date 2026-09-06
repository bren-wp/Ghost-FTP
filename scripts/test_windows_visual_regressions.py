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
        connection = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertNotIn("invalidateRect.Call(a.hwnd, 0, 1)", connection)

    def test_workspace_uses_real_pe_brand_icon_and_full_title_gutter(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        self.assertIn('loadImageW.Call(hinst, 1, imageIcon', chrome)
        self.assertIn('stmSetImage', chrome)
        self.assertNotIn('a.move(a.brandTitle, 54, 10, 106, 35)', chrome)
        self.assertIn('a.move(a.brandTitle, 54, headerY, 126, 35)', ui)
        self.assertIn('subtitleX := 188', ui)

    def test_native_dark_chrome_and_site_manager_use_premium_controls(self):
        dark = self.read("internal/desktop/dark_mode_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")
        site = self.read("internal/desktop/site_manager_windows.go")
        self.assertIn("enableImmersiveDarkMode(hwnd)", ui)
        self.assertIn("SetPreferredAppMode", dark)
        self.assertIn('case "COMBOBOX", "EDIT":', ui)
        self.assertIn("case wmDrawItem:", site)
        self.assertIn("bsOwnerDraw", site)
        self.assertIn("IconSm:     icon", site)

    def test_resize_is_batched_without_background_erase(self):
        source = self.read("internal/desktop/layout_batch_windows.go")
        windows = self.read("internal/desktop/windows.go")
        self.assertIn("wmSetRedraw", source)
        self.assertIn("rdwAllChildren", source)
        self.assertNotIn("rdwErase", source)
        self.assertIn("a.reflowWorkspace(w, h)", windows)

    def test_full_wordmark_dark_headers_and_ownerdrawn_menu_are_guarded(self):
        chrome = self.read("internal/desktop/chrome_windows.go")
        header = self.read("internal/desktop/header_draw_windows.go")
        menu = self.read("internal/desktop/menu_draw_windows.go")
        wnd = self.read("internal/desktop/windows.go")
        self.assertIn("titleWidth, subtitleX = 54, 168, 230", chrome)
        self.assertIn("installWorkspaceHeaderDraw(a, list)", chrome)
        self.assertIn("SetWindowSubclass", header)
        self.assertIn("workspaceListSubclass", header)
        self.assertIn("nmCustomDraw", header)
        self.assertIn("fillRectHeader.Call", header)
        self.assertIn("setTextColor.Call(d.HDC, textColor())", header)
        self.assertIn("cdrfSkipDefault", header)
        self.assertIn("mfOwnerDraw", self.read("internal/desktop/menu_windows.go"))
        self.assertIn("applyDarkMenuBackground", menu)
        self.assertIn("a.measureMenuItem(lParam)", wnd)
        self.assertIn("a.drawMenuItem(&d)", wnd)

    def test_disconnected_remote_list_keeps_dark_enabled_surface(self):
        source = self.read("internal/desktop/chrome_windows.go")
        self.assertIn("setControlEnabled(a.remoteList, true)", source)
        self.assertIn("styleWorkspaceList(list)", source)

    def test_startup_settings_skip_redundant_language_rebuild(self):
        source = self.read("internal/desktop/settings_windows.go")
        self.assertIn("previousLanguage := a.languageCode()", source)
        self.assertIn("if a.languageCode() != previousLanguage", source)

    def test_connect_timeout_and_cancel_are_real_ui_behaviors(self):
        source = self.read("internal/desktop/connection_profiles_windows.go")
        self.assertIn("connectionTimeoutDuration(a.settings)", source)
        self.assertIn("a.connectionBusy && !a.connected", source)
        self.assertIn("a.cancelConnectionAttempt()", source)

    def test_x86_ftp_and_sftp_have_secure_sysnative_fallbacks(self):
        ftp = self.read("internal/remote/tools.go")
        sftp = self.read("internal/remote/sftp.go")
        self.assertIn('arch == "386"', ftp)
        self.assertIn('"Sysnative", "curl.exe"', ftp)
        self.assertNotIn('exec.LookPath("curl.exe")', ftp)
        self.assertIn("windowsOpenSSHCandidates", sftp)
        self.assertIn('"Sysnative", "OpenSSH", name', sftp)


if __name__ == "__main__":
    unittest.main()
