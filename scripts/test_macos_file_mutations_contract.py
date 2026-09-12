from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read_bridge_package() -> str:
    bridge_dir = ROOT / "macos/Bridge"
    sources = sorted(bridge_dir.glob("*.go"))
    if not sources:
        raise AssertionError("missing macOS bridge Go sources")
    return "\n".join(path.read_text(encoding="utf-8") for path in sources)


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
        self.assertIn("local folder changed; refresh and try again", self.bridge)

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
        self.assertIn("remote folder changed; refresh and try again", self.bridge)

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

    def test_mutations_refresh_only_the_unchanged_navigation_generation(self):
        self.assertIn("localMutationGeneration", self.swift)
        self.assertIn("remoteMutationGeneration", self.swift)
        self.assertIn("generation == self.localNavigationGeneration", self.swift)
        self.assertIn("generation == self.remoteNavigationGeneration", self.swift)

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
