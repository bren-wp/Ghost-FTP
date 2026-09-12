from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


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


class MacOSRecursiveSearchContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.bridge = read_bridge_package()
        cls.swift = (ROOT / "macos/Sources/GhostFTPApp/main.swift").read_text(encoding="utf-8")
        cls.parity = (ROOT / "macos/PARITY.md").read_text(encoding="utf-8")
        cls.api = (ROOT / "internal/api/recursive_search.go").read_text(encoding="utf-8")
        cls.windows = (ROOT / "internal/desktop/recursive_search_windows.go").read_text(encoding="utf-8")

    def test_bridge_uses_shared_typed_recursive_search_api(self):
        local = function_body(self.bridge, "func GhostFTPSearchLocalRecursive(")
        remote = function_body(self.bridge, "func GhostFTPSearchRemoteRecursive(")
        self.assertIn("engine.SearchLocalRecursive", local)
        self.assertIn("engine.SearchRemoteRecursive", remote)
        for body in (local, remote):
            self.assertIn("api.SearchOptions{Query: query}", body)
        self.assertIn("filesearch.Local", self.api)
        self.assertIn("filesearch.Walk", self.api)
        self.assertIn("remote.Operation(ctx)", self.api)

    def test_search_is_bounded_batched_and_cancellable(self):
        for marker in (
            "DefaultMaxDepth",
            "DefaultMaxVisited",
            "DefaultMaxResults",
            "DefaultBatchSize",
            "DefaultTimeout",
        ):
            self.assertIn(marker, (ROOT / "internal/filesearch/types.go").read_text(encoding="utf-8"))
        cancel = function_body(self.bridge, "func GhostFTPCancelRecursiveSearch(")
        self.assertIn("cancelRecursiveSearch", cancel)
        self.assertIn("GhostFTPCancelRecursiveSearch()", self.swift)
        disconnect = function_body(self.swift, "private func disconnectTapped(")
        self.assertIn("GhostFTPCancelRecursiveSearch()", disconnect)

    def test_search_is_bound_to_visible_snapshot(self):
        local = function_body(self.bridge, "func GhostFTPSearchLocalRecursive(")
        remote = function_body(self.bridge, "func GhostFTPSearchRemoteRecursive(")
        self.assertIn("requireLocalSearchSnapshot", local)
        self.assertIn("requireRemoteSearchSnapshot", remote)
        self.assertIn("bridgeState.localPath", self.bridge)
        self.assertIn("bridgeState.remotePath", self.bridge)

    def test_native_ui_has_real_search_cancel_close_and_navigate_flow(self):
        for marker in (
            'NSButton(title: "Search"',
            "localRecursiveSearchTapped",
            "remoteRecursiveSearchTapped",
            "RecursiveSearchWindowController",
            'title = "Cancel"',
            'title = "Close"',
            'title = "Navigate"',
            "navigateRecursiveSearchResult",
            "GhostFTPSearchResultCount",
            "GhostFTPSearchResultParent",
        ):
            self.assertIn(marker, self.swift)

    def test_remote_search_cancel_precedes_queued_disconnect(self):
        disconnect = function_body(self.swift, "private func disconnectTapped(")
        cancel_pos = disconnect.find("GhostFTPCancelRecursiveSearch()")
        queue_pos = disconnect.find("engineQueue.async")
        self.assertGreaterEqual(cancel_pos, 0)
        self.assertGreater(queue_pos, cancel_pos)

    def test_windows_and_macos_parity_are_aligned(self):
        self.assertIn("SearchLocalRecursive", self.windows)
        self.assertIn("SearchRemoteRecursive", self.windows)
        self.assertIn("cancelRecursiveSearch", self.windows)
        self.assertIn("navigateRecursiveSearch", self.windows)
        self.assertIn("- [x] Local Recursive Search", self.parity)
        self.assertIn("- [x] Remote Recursive Search", self.parity)


if __name__ == "__main__":
    unittest.main()
