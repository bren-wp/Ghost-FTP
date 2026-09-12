from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def read_bridge_package() -> str:
    sources = sorted((ROOT / "macos/Bridge").glob("*.go"))
    if not sources:
        raise AssertionError("missing macOS bridge Go sources")
    return "\n".join(path.read_text(encoding="utf-8") for path in sources)


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
        cls.bridge = read_bridge_package()
        cls.bridge_main = read("macos/Bridge/main.go")
        cls.swift = read("macos/Sources/GhostFTPApp/main.swift")
        cls.parity = read("macos/PARITY.md")
        cls.shared = read("internal/itemlist/filter.go")
        cls.windows = read("internal/desktop/file_filter_windows.go")

    def test_bridge_uses_shared_non_destructive_filter_semantics(self):
        for marker in (
            '"github.com/bren-wp/Ghost-FTP/internal/itemlist"',
            "localVisibleItems",
            "remoteVisibleItems",
            "localFilterQuery",
            "remoteFilterQuery",
            "itemlist.Filter(bridgeState.localItems, bridgeState.localFilterQuery)",
            "itemlist.Filter(bridgeState.remoteItems, bridgeState.remoteFilterQuery)",
            "GhostFTPLocalFilter",
            "GhostFTPRemoteFilter",
            "GhostFTPLocalAllItemCount",
            "GhostFTPRemoteAllItemCount",
        ):
            self.assertIn(marker, self.bridge)

        self.assertIn("unicode.SimpleFold", self.shared)
        self.assertIn("strings.Fields", self.shared)
        self.assertIn("return append(out, items...)", self.shared)

    def test_filter_exports_never_trigger_hidden_io(self):
        for action in ("GhostFTPLocalFilter", "GhostFTPRemoteFilter"):
            body = function_body(self.bridge, f"func {action}(")
            self.assertIn("FilterQuery", body)
            self.assertNotIn("LocalList", body)
            self.assertNotIn("RemoteList", body)
            self.assertNotIn("context.WithTimeout", body)

    def test_directory_refresh_reapplies_existing_filter_to_new_snapshot(self):
        local_list = function_body(self.bridge_main, "func GhostFTPLocalList(")
        remote_list = function_body(self.bridge_main, "func GhostFTPRemoteList(")
        self.assertIn("refreshLocalFilterLocked()", local_list)
        self.assertIn("refreshRemoteFilterLocked()", remote_list)
        self.assertIn("bridgeState.localItems", local_list)
        self.assertIn("bridgeState.remoteItems", remote_list)

    def test_existing_item_accessors_expose_visible_not_authoritative_rows(self):
        for signature in (
            "func GhostFTPLocalItemCount(",
            "func GhostFTPLocalItemName(",
            "func GhostFTPLocalItemSize(",
            "func GhostFTPLocalItemIsDirectory(",
            "func GhostFTPLocalItemIsSymlink(",
            "func GhostFTPLocalItemModifiedUnix(",
        ):
            self.assertIn("localVisibleItems", function_body(self.bridge_main, signature))
        for signature in (
            "func GhostFTPRemoteItemCount(",
            "func GhostFTPRemoteItemName(",
            "func GhostFTPRemoteItemSize(",
            "func GhostFTPRemoteItemIsDirectory(",
            "func GhostFTPRemoteItemIsSymlink(",
            "func GhostFTPRemoteItemModifiedUnix(",
            "func GhostFTPRemoteItemPermissions(",
        ):
            self.assertIn("remoteVisibleItems", function_body(self.bridge_main, signature))

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
            "GhostFTPLocalAllItemCount",
            "GhostFTPRemoteAllItemCount",
        ):
            self.assertIn(marker, self.swift)

        apply_filter = function_body(self.swift, "private func applyCurrentFolderFilter(")
        self.assertNotIn("refreshLocal(", apply_filter)
        self.assertNotIn("refreshRemote(", apply_filter)
        self.assertNotIn("GhostFTPLocalList", apply_filter)
        self.assertNotIn("GhostFTPRemoteList", apply_filter)

    def test_filter_preserves_visible_selection_by_name_and_remote_enablement(self):
        apply_filter = function_body(self.swift, "private func applyCurrentFolderFilter(")
        self.assertIn("selectedItemNames", apply_filter)
        self.assertIn("restoreSelection", apply_filter)
        controls = function_body(self.swift, "private func updateWorkspaceControls(")
        self.assertIn("localFilterButton.isEnabled", controls)
        self.assertIn("remoteFilterButton.isEnabled = connected", controls)

    def test_windows_and_macos_parity_are_aligned(self):
        self.assertIn("itemlist.Filter(state.localAll, state.localQuery)", self.windows)
        self.assertIn("itemlist.Filter(state.remoteAll, state.remoteQuery)", self.windows)
        self.assertIn("- [x] Local Filter", self.parity)
        self.assertIn("- [x] Remote Filter", self.parity)
        self.assertNotIn("Recursive Search", function_body(self.swift, "private func applyCurrentFolderFilter("))


if __name__ == "__main__":
    unittest.main()
