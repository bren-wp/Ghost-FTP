from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read_bridge_package() -> str:
    bridge_dir = ROOT / "macos/Bridge"
    sources = sorted(bridge_dir.glob("*.go"))
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


class MacOSFileMutationsContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.bridge = read_bridge_package()
        cls.swift = (ROOT / "macos/Sources/GhostFTPApp/main.swift").read_text(encoding="utf-8")
        cls.parity = (ROOT / "macos/PARITY.md").read_text(encoding="utf-8")

    def test_bridge_exposes_real_local_mutations(self):
        for marker in (
            "GhostFTPLocalMkdir",
            "GhostFTPLocalRename",
            "GhostFTPLocalDelete",
            "engine.LocalMkdir",
            "engine.LocalRename",
            "engine.LocalDelete",
        ):
            self.assertIn(marker, self.bridge)

        snapshot_guard = function_body(self.bridge, "func requireLocalSnapshot(")
        self.assertIn("bridgeState.localPath", snapshot_guard)
        self.assertIn("local folder changed; refresh and try again", snapshot_guard)
        for action in ("GhostFTPLocalMkdir", "GhostFTPLocalRename", "GhostFTPLocalDelete"):
            body = function_body(self.bridge, f"func {action}(")
            self.assertIn("requireLocalSnapshot", body)
        for action in ("GhostFTPLocalRename", "GhostFTPLocalDelete"):
            body = function_body(self.bridge, f"func {action}(")
            self.assertIn("requireVisibleLocalItem", body)

    def test_bridge_exposes_real_remote_mutations(self):
        for marker in (
            "GhostFTPRemoteMkdir",
            "GhostFTPRemoteRename",
            "GhostFTPRemoteDelete",
            "engine.RemoteMkdir",
            "engine.RemoteRename",
            "engine.RemoteDelete",
        ):
            self.assertIn(marker, self.bridge)

        snapshot_guard = function_body(self.bridge, "func requireRemoteSnapshot(")
        self.assertIn("bridgeState.remotePath", snapshot_guard)
        self.assertIn("remote folder changed; refresh and try again", snapshot_guard)
        for action in ("GhostFTPRemoteMkdir", "GhostFTPRemoteRename", "GhostFTPRemoteDelete"):
            body = function_body(self.bridge, f"func {action}(")
            self.assertIn("requireRemoteSnapshot", body)
        for action in ("GhostFTPRemoteRename", "GhostFTPRemoteDelete"):
            body = function_body(self.bridge, f"func {action}(")
            self.assertIn("requireVisibleRemoteItem", body)

    def test_app_exposes_native_mutation_actions_and_delete_confirmation(self):
        for marker in (
            'title: "New Folder"',
            'title: "Rename"',
            'title: "Delete"',
            "localNewFolderTapped",
            "localRenameTapped",
            "localDeleteTapped",
            "remoteNewFolderTapped",
            "remoteRenameTapped",
            "remoteDeleteTapped",
            "confirmDelete",
            'messageText = "Delete selected item?"',
        ):
            self.assertIn(marker, self.swift)

    def test_mutation_completion_guards_refresh_by_navigation_generation(self):
        local = function_body(self.swift, "private func runLocalMutation(")
        self.assertIn("let generation = localNavigationGeneration", local)
        self.assertIn("result.changed && generation == self.localNavigationGeneration", local)
        self.assertIn("self.refreshLocal(base)", local)

        remote = function_body(self.swift, "private func runRemoteMutation(")
        self.assertIn("let generation = remoteNavigationGeneration", remote)
        self.assertIn("result.changed && generation == self.remoteNavigationGeneration", remote)
        self.assertIn("GhostFTPIsConnected() == 1", remote)
        self.assertIn("self.refreshRemote(base)", remote)

    def test_parity_marks_only_newly_wired_mutations_complete(self):
        for item in (
            "Local New Folder",
            "Local Rename",
            "Local Delete",
            "Remote New Folder",
            "Remote Rename",
            "Remote Delete",
        ):
            self.assertIn(f"- [x] {item}", self.parity)
        self.assertIn("- [ ] Remote Permissions", self.parity)


if __name__ == "__main__":
    unittest.main()
