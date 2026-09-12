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


class MacOSRemotePermissionsContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.bridge = read_bridge_package()
        cls.swift = (ROOT / "macos/Sources/GhostFTPApp/main.swift").read_text(encoding="utf-8")
        cls.parity = (ROOT / "macos/PARITY.md").read_text(encoding="utf-8")
        cls.windows = (ROOT / "internal/desktop/files_actions_windows.go").read_text(encoding="utf-8")

    def test_bridge_exposes_snapshot_bound_remote_chmod(self):
        body = function_body(self.bridge, "func GhostFTPRemoteChmod(")
        self.assertIn("requireRemoteSnapshot", body)
        self.assertIn("requireVisibleRemotePermissionItem", body)
        self.assertIn("engine.RemoteChmod", body)
        self.assertIn("context.WithTimeout", body)

        guard = function_body(self.bridge, "func requireVisibleRemotePermissionItem(")
        self.assertIn("snapshotItem(bridgeState.remoteItems", guard)
        self.assertIn("ValidateRemoteName", guard)
        self.assertIn("IsSymlink", guard)

    def test_app_exposes_real_permissions_action_with_windows_batch_bound(self):
        for marker in (
            'NSButton(title: "Permissions"',
            "remotePermissionsTapped",
            'messageText = "Remote permissions"',
            'informativeText = "Permissions (644 / 755):"',
            'NSTextField(string: "644")',
            "selected.count > 1000",
            "item.isSymlink",
            "GhostFTPRemoteChmod",
        ):
            self.assertIn(marker, self.swift)

    def test_permissions_mode_is_prevalidated_but_engine_remains_authoritative(self):
        action = function_body(self.swift, "private func remotePermissionsTapped(")
        self.assertIn("isValidPermissionMode", action)
        validator = function_body(self.swift, "private func isValidPermissionMode(")
        self.assertIn("value.count == 3 || value.count == 4", validator)
        self.assertIn("$0 >= \"0\" && $0 <= \"7\"", validator)

        bridge = function_body(self.bridge, "func GhostFTPRemoteChmod(")
        self.assertIn("engine.RemoteChmod", bridge)

    def test_permissions_batch_reports_success_failure_and_skipped_links(self):
        action = function_body(self.swift, "private func remotePermissionsTapped(")
        for marker in (
            "var changed = 0",
            "var failed = 0",
            "var skipped = 0",
            '"Changed: \\(changed)"',
            '"Failed: \\(failed)"',
            '"Skipped links: \\(skipped)"',
        ):
            self.assertIn(marker, action)
        self.assertIn("runRemoteMutation", action)

    def test_permissions_reuses_remote_mutation_generation_refresh_guard(self):
        runner = function_body(self.swift, "private func runRemoteMutation(")
        self.assertIn("let generation = remoteNavigationGeneration", runner)
        self.assertIn("result.changed && generation == self.remoteNavigationGeneration", runner)
        self.assertIn("GhostFTPIsConnected() == 1", runner)
        self.assertIn("self.refreshRemote(base)", runner)

    def test_windows_contract_and_macos_parity_are_aligned(self):
        windows = function_body(self.windows, "func (a *app) remoteChmodAction(")
        self.assertIn("len(indices) > 1000", windows)
        self.assertIn("item.IsSymlink", windows)
        self.assertIn('"644"', windows)
        self.assertIn("engine.RemoteChmod", windows)
        self.assertIn("- [x] Remote Permissions", self.parity)


if __name__ == "__main__":
    unittest.main()
