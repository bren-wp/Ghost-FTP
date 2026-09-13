#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        raise AssertionError(f"missing required macOS application parity file: {relative}")
    return path.read_text(encoding="utf-8")


class MacOSApplicationParityContractTests(unittest.TestCase):
    def test_typed_application_bridge_uses_shared_engine_only(self) -> None:
        bridge = read("macos/Bridge/application.go")
        for marker in (
            "GhostFTPRefreshBookmarks",
            "GhostFTPBookmarkCount",
            "GhostFTPSaveLocalBookmark",
            "GhostFTPSaveRemoteBookmark",
            "GhostFTPRemoveBookmark",
            "GhostFTPNavigateBookmark",
            "bridgeState.engine.Bookmarks()",
            "bridgeState.engine.SaveLocalBookmark",
            "bridgeState.engine.SaveRemoteBookmark",
            "bridgeState.engine.RemoveBookmark",
            "bridgeState.engine.NavigateBookmark",
            "GhostFTPRefreshSettings",
            "GhostFTPSaveSettings",
            "bridgeState.engine.Settings()",
            "bridgeState.engine.SetSettings",
            "GhostFTPLanguageCount",
            "i18n.Languages()",
            "GhostFTPDiagnosticsConnected",
            "bridgeState.engine.ActiveConnection()",
            "GhostFTPAboutPublisher",
            "macAboutPublisher",
        ):
            self.assertIn(marker, bridge)
        for forbidden in ("json.Marshal", "json.Unmarshal", "PasswordBlob", "PassphraseBlob"):
            self.assertNotIn(forbidden, bridge)
        for definition in (
            "macAboutPublisher     =",
            "macAboutWebsite       =",
            "macAboutAuthorWebsite =",
            "macAboutSupport       =",
        ):
            self.assertNotIn(definition, bridge)

        about_identity = read("macos/Bridge/about_identity.go")
        for marker in ("macAboutPublisher", "macAboutWebsite", "macAboutAuthorWebsite", "macAboutSupport"):
            self.assertIn(marker, about_identity)

    def test_bookmarks_window_has_real_navigation_and_mutation_actions(self) -> None:
        source = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")
        for marker in (
            "BookmarksWindowController",
            'NSButton(title: "Open"',
            'NSButton(title: "Add Local"',
            'NSButton(title: "Add Remote"',
            'NSButton(title: "Delete"',
            "GhostFTPRefreshBookmarks",
            "GhostFTPSaveLocalBookmark",
            "GhostFTPSaveRemoteBookmark",
            "GhostFTPRemoveBookmark",
            "GhostFTPNavigateBookmark",
            "NSAlert()",
        ):
            self.assertIn(marker, source)

    def test_settings_window_exposes_shared_settings_and_languages(self) -> None:
        source = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")
        for marker in (
            "SettingsWindowController",
            'window.title = "Settings"',
            "GhostFTPRefreshSettings",
            "GhostFTPSaveSettings",
            "GhostFTPLanguageCount",
            "Language",
            "Appearance",
            "Parallel transfers",
            "Upload limit",
            "Download limit",
            "Connection timeout",
            "Automatic retries",
            "Retry delay",
            "Conflict policy",
            "Confirm delete",
            "Dark",
            "Classic Light",
        ):
            self.assertIn(marker, source)

        preparer = read("macos/prepare_site_manager_sources.py")
        self.assertIn("GhostFTPSettingsConfirmDelete() == 0", preparer)

    def test_about_and_diagnostics_are_privacy_safe(self) -> None:
        source = read("macos/Sources/GhostFTPApp/ApplicationWindows.swift")
        for marker in (
            "AboutWindowController",
            "DiagnosticsWindowController",
            "GhostFTPAboutPublisher",
            "GhostFTPAboutVersion",
            "GhostFTPDiagnosticsConnected",
            "GhostFTPDiagnosticsProtocol",
            "GhostFTPDiagnosticsRemotePath",
            "No telemetry or tracking.",
            "Saved profiles stay on this computer.",
        ):
            self.assertIn(marker, source)
        for forbidden in ("Password", "Passphrase", "PasswordBlob", "PassphraseBlob", "log dump", "credential"):
            self.assertNotIn(forbidden, source)

    def test_generated_main_has_visible_entry_points_and_callbacks(self) -> None:
        preparer = read("macos/prepare_site_manager_sources.py")
        for marker in (
            "BookmarksWindowController()",
            "SettingsWindowController()",
            "AboutWindowController()",
            "DiagnosticsWindowController()",
            'NSButton(title: "Bookmarks"',
            'NSButton(title: "Settings"',
            'NSButton(title: "About"',
            'NSButton(title: "Diagnostics"',
            "bookmarkNavigationApplied",
            "settingsAppearanceChanged",
        ):
            self.assertIn(marker, preparer)

    def test_build_compiles_application_windows_and_checks_final_surface(self) -> None:
        build = read("macos/BUILD.sh")
        for marker in (
            "ApplicationWindows.swift",
            'grep -F \'NSButton(title: "Bookmarks"\'',
            'grep -F \'NSButton(title: "Settings"\'',
            'grep -F \'NSButton(title: "About"\'',
            'grep -F \'NSButton(title: "Diagnostics"\'',
            "main.productVersion=${VERSION}",
        ):
            self.assertIn(marker, build)

    def test_action_inventory_is_fully_complete(self) -> None:
        parity = read("macos/PARITY.md")
        global_contract = read("scripts/test_macos_windows_parity_contract.py")
        for action in ("Bookmarks", "Settings", "About", "Diagnostics"):
            self.assertIn(f"- [x] {action}", parity)
            self.assertIn(f'"{action}"', global_contract.split("IMPLEMENTED_MACOS_ACTIONS =", 1)[1])
        self.assertNotIn("- [ ]", parity.split("## Implemented application boundary", 1)[0])


if __name__ == "__main__":
    unittest.main()
