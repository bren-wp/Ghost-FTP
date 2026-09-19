#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]

def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")

class CrossPlatformUIUXPolishTests(unittest.TestCase):
    def test_retired_macos_ui_is_absent(self) -> None:
        self.assertFalse((ROOT / "macos").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-app.yml").exists())
        self.assertFalse((ROOT / ".github/workflows/macos-production.yml").exists())

    def test_windows_title_is_not_product_name_twice(self) -> None:
        source = read("internal/desktop/windows.go")
        self.assertIn('wstr(brand.ProductName+" "+version)', source)
        self.assertNotIn('wstr(brand.ProductName+" "+version+" — "+brand.Company)', source)

    def test_windows_bookmarks_have_a_localized_empty_state(self) -> None:
        source = read("internal/desktop/bookmark_manager_windows.go")
        load_start = source.index("func (state *bookmarkManagerState) loadItems")
        load_end = source.index("func (state *bookmarkManagerState) readSelection", load_start)
        load = source[load_start:load_end]
        self.assertIn("if len(items) == 0 {", load)
        self.assertIn("bookmarkWordsForLanguage(state.parent.languageCode())", load)
        self.assertIn("wstr(words.Empty)", load)

    def test_linux_workspace_matches_master_dual_pane_and_queue_hierarchy(self) -> None:
        source = read("internal/desktop/gui_linux.go")
        workspace_start = source.index("func (u *linuxDesktop) renderWorkspace() error")
        queue_start = source.index("func linuxTransferFileName", workspace_start)
        render_start = source.index("func (u *linuxDesktop) render() error", queue_start)
        workspace = source[workspace_start:queue_start]
        queue = source[queue_start:render_start]
        self.assertIn('localTitle := u.tr("section.local")', workspace)
        self.assertIn('remoteTitle := u.tr("section.remote")', workspace)
        self.assertIn("u.renderItemRows(u.fileFilterListRect(false)", workspace)
        self.assertNotIn("u.layout.localNew", workspace)
        self.assertNotIn("u.layout.remoteChmod", workspace)
        for marker in (
            '{fileX, "File"}',
            'u.tr("column.direction")',
            '{progressX, "Progress"}',
            'u.tr("column.status")',
            '{speedX, "Speed"}',
            '{etaX, "ETA"}',
        ):
            self.assertIn(marker, queue)

    def test_linux_empty_transfer_queue_has_clear_state(self) -> None:
        source = read("internal/desktop/gui_linux.go")
        start = source.index("func (u *linuxDesktop) renderQueue() error")
        end = source.index("func (u *linuxDesktop) render() error", start)
        queue = source[start:end]
        self.assertIn("if len(u.transferJobs) == 0 {", queue)
        self.assertIn('u.tr("transfer.summary", 0, 0, 0)', queue)

    def test_linux_master_rail_and_more_menu_match_reference_navigation(self) -> None:
        rail = read("internal/desktop/linux_master_rail.go")
        info = read("internal/desktop/linux_info_overlay.go")
        self.assertIn('"One client.", "Five platforms.", "Zero friction."', rail)
        self.assertIn('"v"+u.version+" (Linux)"', rail)
        self.assertNotIn("u.drawButton(rail.bookmarks", rail)
        self.assertIn("linuxInfoOverlayMore", rail)
        for marker in (
            "u.openFileFilterPrompt(false)",
            "u.openRecursiveSearchPrompt(true)",
            "u.openSelectedRemoteChmod()",
            "u.openSelectedRemoteEditor()",
            "u.startDirectoryComparisonLinux()",
        ):
            self.assertIn(marker, info)

    def test_primary_english_and_croatian_copy_uses_final_product_terminology(self) -> None:
        catalog = read("internal/i18n/catalogs.go")
        for marker in (
            '"profile.quick": "Quick Connect"',
            '"section.local": "Local Files"',
            '"section.remote": "Remote Files"',
            '"section.transfers": "Transfer Queue"',
            '"settings.title": "Ghost FTP — Settings"',
            '"profile.quick": "Brzo povezivanje"',
            '"section.local": "Lokalne datoteke"',
            '"section.remote": "Udaljene datoteke"',
            '"section.transfers": "Red prijenosa"',
            '"settings.title": "Ghost FTP — Postavke"',
        ):
            self.assertIn(marker, catalog)

    def test_windows_connection_manager_uses_master_navigation_name(self) -> None:
        navigation = read("internal/desktop/navigation_windows.go")
        manager = read("internal/desktop/site_manager_windows.go")
        capture = read("scripts/capture_windows_screenshots.ps1")
        runtime = read("scripts/test_windows_modal_keyboard_runtime.ps1")
        self.assertIn("words[5] = labels.Connections", navigation)
        self.assertIn('TitleContains "Connections"', capture)
        self.assertIn("-Title 'Connections'", runtime)
        self.assertNotIn("Site Manager control initialization failed", manager)

    def test_windows_connections_security_heading_is_protocol_neutral(self) -> None:
        source = read("internal/desktop/site_manager_windows.go")
        self.assertIn("func cleanConnectionSecurityTitle", source)
        self.assertIn('strings.ReplaceAll(value, "SFTP", "")', source)
        self.assertIn('return "Security"', source)

    def test_restore_defaults_is_real_on_active_native_settings_surfaces(self) -> None:
        windows = read("internal/desktop/settings_windows.go")
        windows_platform = read("internal/platform/settings_dialog_windows.go")
        linux = read("internal/desktop/gui_linux_settings.go")
        android = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")

        self.assertIn("defaults := config.DefaultSettings()", windows)
        self.assertIn("settingsIDReset", windows_platform)
        self.assertIn("linuxDefaultSettingsDraft", linux)
        self.assertIn("u.settingsRects.reset", linux)
        self.assertIn('Button restoreDefaults = button("Restore app defaults")', android)
        self.assertIn("private void restoreDefaultPreferences()", android)

    def test_android_master_bottom_navigation_uses_reference_names(self) -> None:
        activity = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        self.assertIn('bottomNavButton("Files"', activity)
        self.assertIn('bottomNavButton("Sites"', activity)
        self.assertIn('bottomNavButton("Bookmarks"', activity)
        self.assertIn('bottomNavButton("Transfers"', activity)
        self.assertIn('bottomNavButton("Settings"', activity)
        self.assertNotIn('bottomNavButton("Connections"', activity)
        self.assertNotIn('bottomNavButton("Transfer Queue"', activity)

    def test_android_quick_connect_persistence_is_opt_in_and_delete_safety_is_configurable(self) -> None:
        activity = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        self.assertIn("private boolean rememberEndpoint = false;", activity)
        self.assertIn('prefs.getBoolean("rememberEndpoint", false)', activity)
        self.assertIn('prefs.getBoolean("confirmDelete", true)', activity)
        self.assertIn('.putBoolean("confirmDelete", confirmDelete)', activity)

if __name__ == "__main__":
    unittest.main()
