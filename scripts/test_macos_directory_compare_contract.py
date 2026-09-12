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


class MacOSDirectoryCompareContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.bridge = read_bridge_package()
        cls.swift = (ROOT / "macos/Sources/GhostFTPApp/main.swift").read_text(encoding="utf-8")
        cls.parity = (ROOT / "macos/PARITY.md").read_text(encoding="utf-8")
        cls.api = (ROOT / "internal/api/directory_compare.go").read_text(encoding="utf-8")
        cls.shared = (ROOT / "internal/directorycompare/compare.go").read_text(encoding="utf-8")
        cls.windows = (ROOT / "internal/desktop/directory_compare_windows.go").read_text(encoding="utf-8")

    def test_bridge_uses_shared_typed_directory_compare_api(self):
        compare = function_body(self.bridge, "func GhostFTPCompareDirectories(")
        self.assertIn("engine.LocalList", compare)
        self.assertIn("engine.RemoteList", compare)
        self.assertIn("engine.CompareDirectoryItems", compare)
        self.assertIn("api.DirectoryComparisonOptions{}", compare)
        self.assertNotIn("os.ReadDir", compare)
        self.assertNotIn("filepath.Walk", compare)
        self.assertNotIn("session.List", compare)
        self.assertIn("directorycompare.Compare", self.api)

    def test_compare_is_snapshot_bound_and_cancellable(self):
        compare = function_body(self.bridge, "func GhostFTPCompareDirectories(")
        self.assertIn("requireDirectoryCompareSnapshot", compare)
        cancel = function_body(self.bridge, "func GhostFTPCancelDirectoryCompare(")
        self.assertIn("cancelDirectoryCompare", cancel)
        self.assertIn("GhostFTPCancelDirectoryCompare()", self.swift)
        disconnect = function_body(self.swift, "private func disconnectTapped(")
        self.assertIn("GhostFTPCancelDirectoryCompare()", disconnect)
        self.assertLess(disconnect.find("GhostFTPCancelDirectoryCompare()"), disconnect.find("engineQueue.async"))

    def test_shared_compare_is_read_only_and_fail_closed(self):
        self.assertIn("performs no filesystem/network I/O", self.api)
        self.assertIn("StatusConflict", self.shared)
        self.assertIn("StatusUnknown", self.shared)
        self.assertIn("DefaultTimestampTolerance", self.shared)
        self.assertIn("MaxTimestampTolerance", self.shared)
        self.assertIn("SynchronizedDirectoryName", self.api)

    def test_native_ui_has_compare_close_and_open_both_flow(self):
        for marker in (
            'NSButton(title: "Compare"',
            "directoryCompareTapped",
            "DirectoryCompareWindowController",
            'title = "Close"',
            'title = "Open Both"',
            "openComparedDirectoryBoth",
            "GhostFTPDirectoryCompareCount",
            "GhostFTPDirectoryCompareStatus",
            "GhostFTPDirectoryCompareCanOpenBoth",
        ):
            self.assertIn(marker, self.swift)

    def test_open_both_reuses_normal_navigation_paths(self):
        body = function_body(self.swift, "private func openComparedDirectoryBoth(")
        self.assertIn("GhostFTPDirectoryCompareCanOpenBoth", body)
        self.assertIn("refreshLocal", body)
        self.assertIn("refreshRemote", body)
        self.assertNotIn("GhostFTPLocalList", body)
        self.assertNotIn("GhostFTPRemoteList", body)

    def test_windows_and_macos_parity_are_aligned(self):
        self.assertIn("CompareDirectoryItems", self.windows)
        self.assertIn("SynchronizedDirectoryName", self.windows)
        self.assertIn("- [x] Directory Compare", self.parity)


if __name__ == "__main__":
    unittest.main()
