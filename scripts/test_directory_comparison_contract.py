import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


class DirectoryComparisonContractTests(unittest.TestCase):
    def test_shared_comparator_is_conservative_and_read_only(self):
        source = read("internal/directorycompare/compare.go")
        for marker in (
            'StatusSame        Status = "same"',
            'StatusLocalOnly   Status = "local_only"',
            'StatusRemoteOnly  Status = "remote_only"',
            'StatusNewerLocal  Status = "newer_local"',
            'StatusNewerRemote Status = "newer_remote"',
            'StatusConflict    Status = "conflict"',
            'StatusUnknown     Status = "unknown"',
            "DefaultTimestampTolerance = 2 * time.Second",
            "MaxTimestampTolerance     = 5 * time.Minute",
            "case len(locals) > 1 || len(remotes) > 1:",
            "if local.IsSymlink || remote.IsSymlink",
            "if local.IsDirectory != remote.IsDirectory",
            "localTimeKnown := !local.Modified.IsZero()",
            "remoteTimeKnown := !remote.Modified.IsZero()",
        ):
            self.assertIn(marker, source)
        for forbidden in (".Upload(", ".Download(", ".Delete(", ".Rename(", ".Chmod("):
            self.assertNotIn(forbidden, source)

    def test_sync_resolver_requires_exact_paired_ordinary_directory(self):
        source = read("internal/directorycompare/sync.go")
        for marker in (
            "if entry.Name != name",
            "entry.Status != StatusSame || !entry.HasLocal || !entry.HasRemote",
            "!entry.Local.IsDirectory || !entry.Remote.IsDirectory",
            "entry.Local.IsSymlink || entry.Remote.IsSymlink",
        ):
            self.assertIn(marker, source)

    def test_engine_facade_keeps_comparison_pure(self):
        source = read("internal/api/directory_compare.go")
        self.assertIn("func (e *Engine) CompareDirectoryItems", source)
        self.assertIn("return directorycompare.Compare(local, remote, opts)", source)
        self.assertIn("func (e *Engine) SynchronizedDirectoryName", source)
        self.assertIn("return directorycompare.SynchronizedDirectoryName(entries, name)", source)

    def test_windows_uses_dedicated_read_only_surface_and_safe_sync_navigation(self):
        controller = read("internal/desktop/directory_compare_windows.go")
        commands = read("internal/desktop/commands_windows.go")
        action_state = read("internal/desktop/action_state_windows.go")
        layout = read("internal/desktop/workspace_layout_windows.go")

        for marker in (
            'wstr("BUTTON")',
            "bsOwnerDraw",
            'wstr("SysListView32")',
            "state.localList = a.createDirectoryComparisonList",
            "state.remoteList = a.createDirectoryComparisonList",
            "security.SafeLocalChild(state.localBase, name)",
            "security.ValidateRemoteName(name)",
            "security.ValidateRemotePath(remoteTarget)",
            "a.engine.SynchronizedDirectoryName(state.entries, entry.Name)",
        ):
            self.assertIn(marker, controller)
        self.assertIn("case idDirectoryCompare:", commands)
        self.assertIn("case idDirectoryCompareOpenBoth:", commands)
        self.assertIn("comparisonActive := a.directoryComparisonActive()", action_state)
        self.assertIn("!comparisonActive", action_state)
        self.assertIn("a.ensureDirectoryComparisonControls()", layout)
        self.assertIn("a.layoutDirectoryComparisonControls()", layout)

        nav = controller.index("func (a *app) openComparedDirectoryBoth()")
        local_list = controller.index("a.engine.LocalList(ctx, localTarget)", nav)
        remote_list = controller.index("a.engine.RemoteList(ctx, remoteTarget)", nav)
        local_commit = controller.index("a.localCurrent = localResolved", nav)
        remote_commit = controller.index("a.remoteCurrent = remoteTarget", nav)
        self.assertLess(local_list, remote_list)
        self.assertLess(remote_list, local_commit)
        self.assertLess(remote_list, remote_commit)

    def test_linux_comparison_is_modal_and_commits_navigation_after_fresh_lists(self):
        controller = read("internal/desktop/directory_compare_linux.go")
        integration = read("internal/desktop/file_filter_linux.go")
        for marker in (
            "func (u *linuxDesktop) renderDirectoryComparisonButtonLinux()",
            "func (u *linuxDesktop) handleDirectoryComparisonMouseLinux",
            "u.busy = true",
            "return true",
            "security.SafeLocalChild(localBase, name)",
            "security.ValidateRemoteName(name)",
            "security.ValidateRemotePath(remoteTarget)",
            "u.engine.SynchronizedDirectoryName(entries, entry.Name)",
        ):
            self.assertIn(marker, controller)
        for marker in (
            "u.reconcileDirectoryComparisonLinux()",
            "u.renderDirectoryComparisonControlsLinux()",
            "u.handleDirectoryComparisonMouseLinux(x, y)",
        ):
            self.assertIn(marker, integration)

        nav = controller.index("func (u *linuxDesktop) openComparedDirectoryBothLinux()")
        local_list = controller.index("u.engine.LocalList(ctx, localTarget)", nav)
        remote_list = controller.index("u.engine.RemoteList(ctx, remoteTarget)", nav)
        pending_commit = controller.index("state.navReady = true", nav)
        self.assertLess(local_list, remote_list)
        self.assertLess(remote_list, pending_commit)

    def test_localization_covers_all_supported_languages(self):
        words = read("internal/desktop/directory_compare_words.go")
        tests = read("internal/desktop/directory_compare_words_test.go")
        self.assertIn("directoryCompareTranslations", words)
        self.assertIn("TestDirectoryCompareWordsCoverEverySupportedLanguage", tests)
        self.assertIn("len(directoryCompareTranslations) != len(languages)", tests)


if __name__ == "__main__":
    unittest.main()
