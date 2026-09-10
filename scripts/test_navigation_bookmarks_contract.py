#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class NavigationBookmarksContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_bookmark_model_remains_non_secret_navigation_metadata(self) -> None:
        model = self.read("internal/model/bookmark.go")
        for marker in (
            "type Bookmark struct",
            'Path     string `json:"path"`',
            'Protocol string `json:"protocol,omitempty"`',
            'Host     string `json:"host,omitempty"`',
            'Port     int    `json:"port,omitempty"`',
            'Username string `json:"username,omitempty"`',
            "credentials and trust material never",
        ):
            self.assertIn(marker, model)
        for forbidden in ("Password", "Passphrase", "PrivateKey", "Fingerprint", "ProfileID"):
            self.assertNotIn(forbidden, model)

    def test_persisted_bookmarks_validate_paths_identity_and_corrupt_state(self) -> None:
        store = self.read("internal/config/bookmarks.go")
        tests = self.read("internal/config/bookmarks_test.go")
        for marker in (
            "maxBookmarks  = 256",
            "filepath.IsAbs(b.Path)",
            "security.ValidateRemotePath(b.Path)",
            "security.ValidateConnection(b.Protocol, b.Host, b.Username, b.Port)",
            "profilebinding.AccountMatches(",
            'bookmarksFile + ".previous"',
            "spremljene bookmarke nije moguće pročitati",
        ):
            self.assertIn(marker, store)
        for marker in (
            "TestRemoteBookmarksAreAccountBoundAndPersistNoSecretSchema",
            "TestBookmarksCorruptExistingStateFailsClosed",
            '[]string{"password", "passphrase", "privatekey", "fingerprint", "profileid"}',
            "remote bookmark crossed account boundary",
        ):
            self.assertIn(marker, tests)

    def test_engine_remote_bookmark_uses_real_connection_and_revalidates_after_list(self) -> None:
        api = self.read("internal/api/bookmarks.go")
        for marker in (
            "cfg, connected := e.remote.Config()",
            "Protocol: cfg.Protocol, Host: cfg.Host, Port: cfg.Port, Username: cfg.Username",
            "config.RemoteBookmarkMatchesAccount(bookmark, cfg)",
            "identityBefore, err := e.remote.ConnectionIdentity()",
            "items, err := e.RemoteList(ctx, bookmark.Path)",
            "identityAfter, err := e.remote.ConnectionIdentity()",
            "if identityBefore != identityAfter",
            "config.RemoteBookmarkMatchesAccount(bookmark, cfgAfter)",
        ):
            self.assertIn(marker, api)
        self.assertNotIn("SaveProfile", api, "quick-connect bookmark save must not create a hidden profile")

    def test_profile_start_identity_changes_fail_closed(self) -> None:
        binding = self.read("internal/profilebinding/binding.go")
        config_tests = self.read("internal/config/profile_start_directory_binding_test.go")
        linux = self.read("internal/desktop/profile_start_linux.go")
        linux_tests = self.read("internal/desktop/profile_start_linux_test.go")
        for marker in (
            "func AccountMatches(",
            "EndpointMatches(protocolA, hostA, portA, protocolB, hostB, portB) && usernameA == usernameB",
        ):
            self.assertIn(marker, binding)
        for marker in (
            "TestProfileRemoteStartDoesNotSilentlyCrossAccountIdentity",
            "TestProfileExplicitNewRemoteStartSurvivesIdentityChange",
            "TestProfileInheritedSFTPRemoteStartResetsToDot",
        ):
            self.assertIn(marker, config_tests)
        for marker in (
            "profilebinding.AccountMatches(",
            "u.localCurrent = previousLocal",
            "u.refreshLocal(profile.LocalPath)",
            "currentAccountKey != state.accountKey",
            "u.remoteCurrent == state.inheritedRemote",
            "u.remoteCurrent = linuxProtocolRemoteDefault(u.protocol)",
            'state.inheritedRemote = ""',
            "protocol/host/port/username",
        ):
            self.assertIn(marker, linux)
        for marker in (
            "TestLinuxProfileRemoteStartDoesNotOverwriteManualEditOnRepaint",
            "TestLinuxProfileRemoteStartResetsInheritedPathOnAccountChange",
            "TestLinuxProfileExplicitRemoteStartSurvivesAccountChange",
        ):
            self.assertIn(marker, linux_tests)

    def test_windows_bookmark_manager_is_real_and_generation_bound(self) -> None:
        manager = self.read("internal/desktop/bookmark_manager_windows.go")
        commands = self.read("internal/desktop/commands_windows.go")
        for marker in (
            "bookmarkIDOpen",
            "bookmarkIDAddLocal",
            "bookmarkIDAddRemote",
            "bookmarkIDDelete",
            "bookmarkIDClose",
            "a.engine.NavigateBookmark(ctx, bookmark.ID)",
            "a.connectionGeneration",
            "generation != a.connectionGeneration",
            "state.parent.engine.SaveLocalBookmark",
            "state.parent.engine.SaveRemoteBookmark",
            "state.parent.engine.RemoveBookmark",
            "state.parent.engine.ActiveConnection()",
            "config.RemoteBookmarkMatchesAccount",
        ):
            self.assertIn(marker, manager)
        self.assertIn("case idBookmarks:", commands)
        self.assertIn("a.openBookmarkManager()", commands)

    def test_linux_bookmark_manager_is_rendered_and_fully_dispatched(self) -> None:
        bookmark = self.read("internal/desktop/bookmark_linux.go")
        actions = self.read("internal/desktop/gui_linux_actions.go")
        filters = self.read("internal/desktop/file_filter_linux.go")
        prompt_test = self.read("internal/desktop/bookmark_prompt_linux_test.go")
        viewport_test = self.read("internal/desktop/bookmark_viewport_linux_test.go")
        for marker in (
            "func (u *linuxDesktop) renderBookmarksHeaderButton() error",
            "u.enforceLinuxProfileStartDirectories()",
            "func (u *linuxDesktop) renderBookmarkManagerOverlay() error",
            "func (u *linuxDesktop) handleBookmarkManagerMouse(x, y int) bool",
            "func (u *linuxDesktop) handleBookmarkManagerKey(sym uint32) bool",
            "firstVisible int",
            "visibleRows  int",
            "func (state *linuxBookmarkUIState) ensureSelectionVisible()",
            "func (state *linuxBookmarkUIState) scrollViewport(delta int)",
            "state.scrollUp.contains(x, y)",
            "state.scrollDown.contains(x, y)",
            "func (u *linuxDesktop) bookmarkRowAt(x, y int) int",
            "u.engine.SaveRemoteBookmark",
            "u.engine.SaveLocalBookmark",
            "u.engine.NavigateBookmark(ctx, bookmark.ID)",
            "u.engine.RemoveBookmark(item.ID)",
        ):
            self.assertIn(marker, bookmark)
        for marker in (
            "linuxPromptBookmarkManager",
            "linuxPromptBookmarkLocalName",
            "linuxPromptBookmarkRemoteName",
            "return u.handleBookmarkManagerKey(sym)",
            "return u.renderBookmarkManagerOverlay()",
            "return u.handleBookmarkManagerMouse(x, y)",
            "u.saveLinuxBookmark(false, value)",
            "u.saveLinuxBookmark(true, value)",
            "func (u *linuxDesktop) bookmarkNamePrompt() bool",
            "func (u *linuxDesktop) cancelPrompt()",
            "returnToBookmarks := u.bookmarkNamePrompt()",
            'u.openLinuxBookmarks("")',
            "u.cancelPrompt()",
        ):
            self.assertIn(marker, actions)
        self.assertIn("TestLinuxBookmarkNamePromptClassification", prompt_test)
        self.assertIn("TestLinuxBookmarkViewportKeepsKeyboardSelectionVisible", viewport_test)
        self.assertIn("TestLinuxBookmarkViewportMouseScrollReachesLaterRows", viewport_test)
        self.assertIn("u.renderBookmarksHeaderButton()", filters)
        self.assertIn("u.handleBookmarksHeaderMouse(x, y)", filters)

    def test_docs_keep_source_feature_separate_from_published_003(self) -> None:
        roadmap = self.read("docs/ROADMAP.md")
        detail = self.read("docs/NAVIGATION-BOOKMARKS.md")
        changelog = self.read("CHANGELOG.md")
        self.assertIn("navigation bookmarks and profile start directories", roadmap.lower())
        self.assertIn("implemented in the post-0.0.3 source line", roadmap)
        self.assertIn("targeted for the next public release", roadmap)
        self.assertIn("post-0.0.3 source line", detail)
        self.assertIn("not retroactively part of the already published Ghost FTP 0.0.3 release", detail)
        self.assertIn("AccountMatches", detail)
        self.assertIn("NavigateBookmark", detail)
        self.assertIn("Root `VERSION` remains **0.0.3**", detail)
        self.assertIn("Navigation bookmarks and profile start directories", changelog)


if __name__ == "__main__":
    unittest.main()
