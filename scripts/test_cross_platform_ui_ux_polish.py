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

    def test_windows_connection_manager_uses_master_navigation_name(self) -> None:
        navigation = read("internal/desktop/navigation_windows.go")
        manager = read("internal/desktop/site_manager_windows.go")
        capture = read("scripts/capture_windows_screenshots.ps1")
        runtime = read("scripts/test_windows_modal_keyboard_runtime.ps1")
        self.assertIn("words[5] = labels.Connections", navigation)
        self.assertNotIn("Site Manager control initialization failed", manager)
        self.assertIn('TitleContains "Connections"', capture)
        self.assertIn("-Title 'Connections'", runtime)

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
