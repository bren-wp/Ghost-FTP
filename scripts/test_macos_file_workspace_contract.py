from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class MacOSFileWorkspaceContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.bridge = (ROOT / "macos/Bridge/main.go").read_text(encoding="utf-8")
        cls.swift = (ROOT / "macos/Sources/GhostFTPApp/main.swift").read_text(encoding="utf-8")
        cls.parity = (ROOT / "macos/PARITY.md").read_text(encoding="utf-8")

    def test_bridge_exposes_typed_local_and_remote_snapshots(self):
        for marker in (
            "GhostFTPLocalList",
            "GhostFTPLocalPath",
            "GhostFTPLocalItemCount",
            "GhostFTPLocalItemName",
            "GhostFTPLocalItemSize",
            "GhostFTPLocalItemIsDirectory",
            "GhostFTPLocalItemModifiedUnix",
            "GhostFTPRemoteList",
            "GhostFTPRemotePath",
            "GhostFTPRemoteItemCount",
            "GhostFTPRemoteItemName",
            "GhostFTPRemoteItemSize",
            "GhostFTPRemoteItemIsDirectory",
            "GhostFTPRemoteItemModifiedUnix",
            "GhostFTPRemoteItemPermissions",
        ):
            self.assertIn(marker, self.bridge)
        self.assertIn("engine.LocalList", self.bridge)
        self.assertIn("engine.RemoteList", self.bridge)

    def test_bridge_exposes_real_transfer_entry_points(self):
        self.assertIn("GhostFTPUpload", self.bridge)
        self.assertIn("GhostFTPDownload", self.bridge)
        self.assertIn("engine.AddTransfer", self.bridge)
        self.assertIn("engine.AddTreeTransfer", self.bridge)

    def test_app_uses_native_two_pane_workspace(self):
        for marker in (
            "localTable",
            "remoteTable",
            "NSTableView",
            'makeFilePane(title: "Local"',
            'makeFilePane(title: "Remote"',
            'title: "Refresh"',
            'title: "Choose Folder…"',
            'title: "Up"',
            'title: "Upload"',
            'title: "Download"',
            "localDoubleClicked",
            "remoteDoubleClicked",
            "refreshLocal",
            "refreshRemote",
        ):
            self.assertIn(marker, self.swift)

    def test_workspace_actions_are_marked_complete_only_when_wired(self):
        for item in (
            "Connect",
            "Disconnect",
            "Local Refresh",
            "Local Choose Folder",
            "Local Up",
            "Remote Refresh",
            "Remote Up",
            "Upload",
            "Download",
        ):
            self.assertIn(f"- [x] {item}", self.parity)


if __name__ == "__main__":
    unittest.main()
