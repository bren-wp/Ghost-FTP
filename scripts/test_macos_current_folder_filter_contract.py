from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def function_body(source: str, signature: str) -> str:
    start = source.find(signature)
    if start < 0:
        raise AssertionError(f"missing function signature: {signature}")
    open_brace = source.find("{", start)
    if open_brace < 0:
        raise AssertionError(f"missing function body: {signature}")
    depth = 0
    for index in range(open_brace, len(source)):
        char = source[index]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return source[open_brace : index + 1]
    raise AssertionError(f"unterminated function body: {signature}")


class MacOSCurrentFolderFilterContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.filters = read("macos/Bridge/filters.go")
        cls.bridge_main = read("macos/Bridge/main.go")
        cls.mutations = read("macos/Bridge/mutations.go")
        cls.swift = read("macos/Sources/GhostFTPApp/main.swift")
        cls.parity = read("macos/PARITY.md")
        cls.shared = read("internal/itemlist/filter.go")
        cls.windows = read("internal/desktop/file_filter_windows.go")

    def test_bridge_filters_authoritative_snapshots_with_shared_semantics(self):
        for marker in (
            '"github.com/bren-wp/Ghost-FTP/internal/itemlist"',
            "macFileFilterState.localVisible",
            "macFileFilterState.remoteVisible",
            "macFileFilterState.localQuery",
            "macFileFilterState.remoteQuery",
            "itemlist.Filter(bridgeState.localItems, macFileFilterState.localQuery)",
            "itemlist.Filter(bridgeState.remoteItems, macFileFilterState.remoteQuery)",
            "GhostFTPLocalFilter",
            "GhostFTPRemoteFilter",
            "GhostFTPLocalFilteredItemCount",
            "GhostFTPRemoteFilteredItemCount",
        ):
            self.assertIn(marker, self.filters)

        self.assertIn("unicode.SimpleFold", self.shared)
        self.assertIn("strings.Fields", self.shared)
        self.assertIn("return append(out, items...)", self.shared)

    def test_filter_exports_never_trigger_hidden_io(self):
        for action in ("GhostFTPLocalFilter", "GhostFTPRemoteFilter"):
            body = function_body(self.filters, f"func {action}(")
            self.assertNotIn("LocalList", body)
            self.assertNotIn("RemoteList", body)
            self.assertNotIn("context.WithTimeout", body)
        remote = function_body(self.filters, "func GhostFTPRemoteFilter(")
        self.assertIn("ActiveConnection", remote)

    def test_authoritative_snapshots_remain_separate_from_filtered_rows(self):
        local_list = function_body(self.bridge_main, "func GhostFTPLocalList(")
        remote_list = function_body(self.bridge_main, "func GhostFTPRemoteList(")
        self.assertIn("bridgeState.localItems", local_list)
        self.assertIn("bridgeState.remoteItems", remote_list)
        self.assertNotIn("localVisible", local_list)
        self.assertNotIn("remoteVisible", remote_list)

        for action in ("GhostFTPLocalRename", "GhostFTPLocalDelete"):
            body = function_body(self.mutations, f"func {action}(")
            self.assertIn("requireVisibleLocalItem", body)
        for action in ("GhostFTPRemoteRename", "GhostFTPRemoteDelete"):
            body = function_body(self.mutations, f"func {action}(")
            self.assertIn("requireVisibleRemoteItem", body)

    def test_directory_refresh_reapplies_filter_without_extra_listing(self):
        local = function_body(self.swift, "private func refreshLocal(")
        remote = function_body(self.swift, "private func refreshRemote(")
        self.assertEqual(local.count("GhostFTPLocalList("), 1)
        self.assertEqual(remote.count("GhostFTPRemoteList("), 1)
        self.assertIn("GhostFTPLocalFilter", local)
        self.assertIn("readLocalFilteredSnapshot", local)
        self.assertIn("GhostFTPRemoteFilter", remote)
        self.assertIn("readRemoteFilteredSnapshot", remote)
        self.assertIn("GhostFTPLocalItemCount", local)
        self.assertIn("GhostFTPRemoteItemCount", remote)

    def test_app_exposes_real_local_and_remote_filter_actions(self):
        for marker in (
            'NSButton(title: "Filter"',
            "localFilterButton",
            "remoteFilterButton",
            "localFilterQuery",
            "remoteFilterQuery",
            "localFilterTapped",
            "remoteFilterTapped",
            "promptCurrentFolderFilter",
            "applyCurrentFolderFilter",
            "GhostFTPLocalFilter",
            "GhostFTPRemoteFilter",
            "GhostFTPLocalFilteredItemCount",
            "GhostFTPRemoteFilteredItemCount",
        ):
            self.assertIn(marker, self.swift)

        apply_filter = function_body(self.swift, "private func applyCurrentFolderFilter(")
        self.assertNotIn("refreshLocal(", apply_filter)
        self.assertNotIn("refreshRemote(", apply_filter)
        self.assertNotIn("GhostFTPLocalList", apply_filter)
        self.assertNotIn("GhostFTPRemoteList", apply_filter)
        self.assertIn("engineQueue.async", apply_filter)

    def test_filter_preserves_selection_and_blocks_row_actions_while_replacing_view(self):
        apply_filter = function_body(self.swift, "private func applyCurrentFolderFilter(")
        self.assertIn("selectedItemNames", apply_filter)
        self.assertIn("restoreSelection", apply_filter)
        self.assertIn("localFilterBusy", apply_filter)
        self.assertIn("remoteFilterBusy", apply_filter)

        controls = function_body(self.swift, "private func updateWorkspaceControls(")
        self.assertIn("!localFilterBusy", controls)
        self.assertIn("!remoteFilterBusy", controls)
        self.assertIn("localFilterButton.isEnabled = localReady", controls)
        self.assertIn("remoteFilterButton.isEnabled = remoteReady", controls)
        self.assertIn("remoteTable.isEnabled = remoteReady", controls)

    def test_disconnect_clears_remote_visible_rows_and_source_count(self):
        disconnect = function_body(self.swift, "private func disconnectTapped(")
        self.assertIn("remoteNavigationGeneration += 1", disconnect)
        self.assertIn("remoteFilterBusy = false", disconnect)
        self.assertIn("remoteSourceCount = 0", disconnect)
        self.assertIn("remoteItems.removeAll", disconnect)

    def test_windows_and_macos_parity_are_aligned(self):
        self.assertIn("itemlist.Filter(state.localAll, state.localQuery)", self.windows)
        self.assertIn("itemlist.Filter(state.remoteAll, state.remoteQuery)", self.windows)
        self.assertIn("- [x] Local Filter", self.parity)
        self.assertIn("- [x] Remote Filter", self.parity)
        self.assertNotIn("Recursive Search", function_body(self.swift, "private func applyCurrentFolderFilter("))


if __name__ == "__main__":
    unittest.main()
