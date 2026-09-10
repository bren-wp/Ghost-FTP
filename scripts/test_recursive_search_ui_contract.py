#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def source(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


class RecursiveSearchUIContractTests(unittest.TestCase):
    def test_shared_walker_is_bounded_read_only_and_symlink_safe(self) -> None:
        types = source("internal/filesearch/types.go")
        walk = source("internal/filesearch/walk.go")
        local = source("internal/filesearch/local.go")
        api = source("internal/api/recursive_search.go")

        for marker in (
            "HardMaxDepth",
            "HardMaxVisited",
            "HardMaxResults",
            "HardMaxBatchSize",
            "HardMaxTimeout",
        ):
            self.assertIn(marker, types)
        self.assertIn("context.WithTimeout", walk)
        self.assertIn("item.IsDirectory && !item.IsSymlink", walk)
        self.assertIn("itemlist.MatchesName", walk)
        self.assertIn("os.OpenRoot", local)
        self.assertIn("security.SafeLocalChild", local)
        self.assertIn("e.remote.Operation(ctx)", api)
        self.assertIn("security.ValidateRemoteName", api)
        for forbidden in ("Remove(", "Rename(", "Delete(", "Upload(", "Download("):
            self.assertNotIn(forbidden, walk)

    def test_linux_search_is_modal_incremental_and_fresh_navigated(self) -> None:
        search = source("internal/desktop/recursive_search_linux.go")
        filters = source("internal/desktop/file_filter_linux.go")
        prompts = source("internal/desktop/gui_linux_actions.go")

        self.assertIn("u.busy = true", search)
        self.assertIn("u.resultCh <- linuxUIResult{action: linuxActionNone}", search)
        self.assertIn("state.results = append(state.results, batch...)", search)
        self.assertIn("u.engine.SearchLocalRecursive", search)
        self.assertIn("u.engine.SearchRemoteRecursive", search)
        self.assertIn("u.refreshLocal(result.Parent)", search)
        self.assertIn("u.refreshRemote(result.Parent)", search)
        self.assertIn("filter.localQuery = \"\"", search)
        self.assertIn("filter.remoteQuery = \"\"", search)
        self.assertIn("u.reconcileRecursiveSearchState()", filters)
        self.assertIn("u.handleRecursiveSearchMouse(x, y)", filters)
        self.assertIn("linuxPromptLocalRecursiveSearch", prompts)
        self.assertIn("linuxPromptRemoteRecursiveSearch", prompts)
        self.assertIn("u.startRecursiveSearch(false, value)", prompts)
        self.assertIn("u.startRecursiveSearch(true, value)", prompts)

    def test_windows_results_use_dedicated_list_and_fresh_navigation(self) -> None:
        search = source("internal/desktop/recursive_search_windows.go")
        commands = source("internal/desktop/commands_windows.go")
        state = source("internal/desktop/action_state_windows.go")
        layout = source("internal/desktop/workspace_layout_windows.go")

        self.assertIn('wstr("SysListView32")', search)
        self.assertIn("idLocalRecursiveSearchList", search)
        self.assertIn("idRemoteRecursiveSearchList", search)
        self.assertIn("showControls(false, filterButton, normalList)", search)
        self.assertIn("a.engine.SearchLocalRecursive", search)
        self.assertIn("a.engine.SearchRemoteRecursive", search)
        self.assertIn("items, err := a.engine.RemoteList(ctx, parent)", search)
        self.assertIn("resolved, items, err := a.engine.LocalList(ctx, parent)", search)
        self.assertIn("restoreItemSelection(a.remoteList, a.remoteItems, selected)", search)
        self.assertIn("restoreItemSelection(a.localList, a.localItems, selected)", search)
        self.assertIn("case idLocalRecursiveSearch:", commands)
        self.assertIn("case idRemoteRecursiveSearch:", commands)
        self.assertIn("case idLocalRecursiveSearchNavigate:", commands)
        self.assertIn("case idRemoteRecursiveSearchNavigate:", commands)
        self.assertIn("!localSearch", state)
        self.assertIn("!remoteSearch", state)
        self.assertIn("a.layoutRecursiveSearchControls()", layout)

    def test_all_desktop_locales_have_search_disclosure_and_actions(self) -> None:
        words = source("internal/desktop/recursive_search_words.go")
        tests = source("internal/desktop/recursive_search_words_test.go")
        self.assertIn("LocalDisclosure", words)
        self.assertIn("RemoteDisclosure", words)
        self.assertIn("20 seconds", words)
        self.assertIn("1,000 results", words)
        self.assertIn("TestRecursiveSearchWordsCoverEverySupportedLanguage", tests)
        self.assertIn("len(recursiveSearchTranslations) != len(languages)", tests)

    def test_search_results_do_not_replace_authoritative_windows_filter_source(self) -> None:
        search = source("internal/desktop/recursive_search_windows.go")
        self.assertNotIn("localAll =", search)
        self.assertNotIn("remoteAll =", search)
        self.assertNotIn("a.localItems =", search)
        self.assertNotIn("a.remoteItems =", search)


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("RECURSIVE_SEARCH_UI_CONTRACT=PASS")
