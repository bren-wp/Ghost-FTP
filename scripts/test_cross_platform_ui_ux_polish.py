#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class CrossPlatformUIUXPolishTests(unittest.TestCase):
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
        self.assertIn("siteLBAddString", load)
        self.assertIn("wstr(words.Empty)", load)

    def test_linux_empty_transfer_queue_is_not_a_blank_box(self) -> None:
        source = read("internal/desktop/gui_linux.go")
        start = source.index("func (u *linuxDesktop) renderQueue() error")
        end = source.index("func (u *linuxDesktop) render() error", start)
        queue = source[start:end]
        self.assertIn("if len(u.transferJobs) == 0 {", queue)
        self.assertIn('u.tr("transfer.summary", 0, 0, 0)', queue)

    def test_linux_workspace_titles_stay_out_of_master_rail(self) -> None:
        source = read("internal/desktop/gui_linux.go")
        workspace_start = source.index("func (u *linuxDesktop) renderWorkspace() error")
        queue_start = source.index("func (u *linuxDesktop) renderQueue() error", workspace_start)
        render_start = source.index("func (u *linuxDesktop) render() error", queue_start)
        workspace = source[workspace_start:queue_start]
        queue = source[queue_start:render_start]
        self.assertIn("left: u.layout.localPath.left - 6", workspace)
        self.assertIn("u.x.text(u.layout.localPath.left+8, leftPanel.top+20", workspace)
        self.assertIn("u.x.text(u.layout.queue.left+8, u.layout.pause.top+19", queue)
        self.assertNotIn("u.x.text(premiumOuterGap, leftPanel.top+20", workspace)
        self.assertNotIn("u.x.text(premiumOuterGap, u.layout.pause.top+19", queue)

    def test_linux_quick_connect_label_avoids_known_parenthetical_clipping(self) -> None:
        source = read("internal/desktop/gui_linux.go")
        start = source.index("func (u *linuxDesktop) renderQuickConnect() error")
        end = source.index("func (u *linuxDesktop) renderItemRows", start)
        quick = source[start:end]
        self.assertIn('strings.Index(profileLabel, " (")', quick)
        self.assertIn("strings.TrimSpace(profileLabel[:i])", quick)

    def test_macos_master_rail_and_empty_queue_have_clear_state(self) -> None:
        preparer = read("macos/prepare_site_manager_sources.py")
        main = read("macos/Sources/GhostFTPApp/main.swift")
        self.assertIn(
            "button.contentTintColor = active ? Palette.accentStrong : Palette.muted",
            preparer,
        )
        update_start = main.index("private func updateEmbeddedTransferQueue()")
        update_end = main.index("@objc private func embeddedPauseResumeTapped()", update_start)
        update = main[update_start:update_end]
        self.assertIn('embeddedTransferSummary.stringValue = "No transfers yet."', update)
        self.assertIn("let hasPausableTransfers = transferQueueEntries.contains", update)
        self.assertIn('entry.status == "queued" || entry.status == "running"', update)
        self.assertIn(
            "embeddedPauseResumeButton.isEnabled = engineReady && !transferQueueBusy && hasPausableTransfers",
            update,
        )

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
        self.assertNotIn('"profile.quick": "Quick connect (no profile)"', catalog)
        self.assertNotIn('"profile.quick": "Brzi spoj (bez profila)"', catalog)

    def test_windows_connection_manager_uses_master_navigation_name(self) -> None:
        navigation = read("internal/desktop/navigation_windows.go")
        manager = read("internal/desktop/site_manager_windows.go")
        capture = read("scripts/capture_windows_screenshots.ps1")
        runtime = read("scripts/test_windows_modal_keyboard_runtime.ps1")
        self.assertIn("words[5] = labels.Connections", navigation)
        self.assertNotIn("Site Manager control initialization failed", manager)
        self.assertIn('TitleContains "Connections"', capture)
        self.assertIn("-Title 'Connections'", runtime)

    def test_windows_connections_security_heading_is_protocol_neutral(self) -> None:
        source = read("internal/desktop/site_manager_windows.go")
        self.assertIn("func cleanConnectionSecurityTitle", source)
        self.assertIn('strings.ReplaceAll(value, "SFTP", "")', source)
        self.assertIn('return "Security"', source)
        self.assertIn('cleanConnectionSecurityTitle(parent.tr("sftp.security"))', source)
        self.assertNotIn('cleanSFTPSecurityTitle(parent.tr("sftp.security"))', source)

    def test_macos_connection_surfaces_use_public_master_names(self) -> None:
        manager = read("macos/Sources/GhostFTPApp/SiteManager.swift")
        windows = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")
        readme = read("macos/README.md")
        parity = read("macos/PARITY.md")

        self.assertIn('window.title = "Connections"', manager)
        self.assertIn('nameColumn.title = "Saved profiles"', manager)
        self.assertNotIn('window.title = "Site Manager"', manager)

        diagnostics = windows[windows.index("final class DiagnosticsWindowController"):]
        self.assertIn('window.title = "Connection info"', diagnostics)
        self.assertIn('applicationLabel("Connection info"', diagnostics)
        self.assertNotIn('window.title = "Diagnostics"', diagnostics)
        self.assertNotIn("GhostFTPDiagnosticsRemotePath", diagnostics)
        self.assertNotIn("Remote folder:", diagnostics)
        self.assertIn("About and Connection info surfaces", readme)
        self.assertIn("Connection info is intentionally compact and privacy-safe", parity)

    def test_restore_defaults_is_real_on_every_native_settings_surface(self) -> None:
        windows = read("internal/desktop/settings_windows.go")
        windows_platform = read("internal/platform/settings_dialog_windows.go")
        linux = read("internal/desktop/gui_linux_settings.go")
        macos = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")
        android = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")

        self.assertIn("defaults := config.DefaultSettings()", windows)
        self.assertIn("ResetLabel:             settingsResetLabel(language)", windows)
        self.assertIn("settingsIDReset", windows_platform)
        self.assertIn("state.config.DefaultNumbers", windows_platform)

        self.assertIn("linuxDefaultSettingsDraft", linux)
        self.assertIn("settingsResetLabel(u.settingsDraft.Language)", linux)
        self.assertIn("u.settingsRects.reset", linux)

        settings = macos[macos.index("final class SettingsWindowController"):macos.index("final class AboutWindowController")]
        self.assertIn('NSButton(title: "Restore Defaults"', settings)
        self.assertIn("restoreDefaultsTapped", settings)
        self.assertIn('parallelismField.stringValue = "2"', settings)
        self.assertIn("confirmDeleteButton.state = .on", settings)

        self.assertIn('Button restoreDefaults = button("Restore app defaults")', android)
        self.assertIn("private void restoreDefaultPreferences()", android)
        self.assertIn("confirmDelete = true;", android)

    def test_linux_settings_header_does_not_mislabel_the_whole_panel_as_delete_safety(self) -> None:
        source = read("internal/desktop/gui_linux_settings.go")
        start = source.index("func (u *linuxDesktop) renderSettingsOverlay() error")
        overlay = source[start:]
        self.assertNotIn('u.draftTr("settings.confirm_delete_body")', overlay)
        self.assertIn("row := top + 60", overlay)
        self.assertIn("conflictPolicyText(u.settingsDraft.Language).Title", overlay)

    def test_android_quick_connect_persistence_is_opt_in_and_delete_safety_is_configurable(self) -> None:
        activity = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        self.assertIn("private boolean rememberEndpoint = false;", activity)
        self.assertIn('prefs.getBoolean("rememberEndpoint", false)', activity)
        self.assertIn('prefs.getBoolean("confirmDelete", true)', activity)
        self.assertIn('.putBoolean("confirmDelete", confirmDelete)', activity)
        self.assertIn("if (!confirmDelete) {", activity)

    def test_android_hides_only_idle_phone_status_chrome(self) -> None:
        activity = read("android/app/src/main/java/app/ghostftp/client/MainActivity.java")
        build_start = activity.index("private LinearLayout buildMainColumn()")
        build_end = activity.index("private LinearLayout buildNavigationPanel()", build_start)
        build = activity[build_start:build_end]
        self.assertIn(
            "status.setVisibility(tabletLayout ? View.VISIBLE : View.GONE);",
            build,
        )
        set_start = activity.index("private void setStatus(String value)")
        set_end = activity.index("private static String safeMessage", set_start)
        set_status = activity[set_start:set_end]
        self.assertIn('boolean idle = safe.isEmpty() || "Ready.".equals(safe);', set_status)
        self.assertIn(
            "status.setVisibility(!tabletLayout && idle ? View.GONE : View.VISIBLE);",
            set_status,
        )
        self.assertIn("updateTransferSurface();", set_status)


if __name__ == "__main__":
    unittest.main()
