import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class MacOSRemoteEditContract(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_shared_engine_keeps_conflict_and_read_back_guards(self):
        source = self.read("internal/api/remote_edit.go")
        self.assertIn("func (e *Engine) RemoteEditOpen", source)
        self.assertIn("func (e *Engine) RemoteEditSave", source)
        self.assertIn("ErrRemoteEditConflict", source)
        self.assertIn("verified.Revision != writtenRevision", source)
        self.assertIn("remote edit was uploaded but read-back content did not match", source)
        self.assertIn("MaxRemoteEditBytes", source)

    def test_macos_bridge_is_only_a_typed_adapter_to_shared_remote_edit(self):
        bridge = self.read("macos/Bridge/remote_edit.go")
        self.assertIn("//export GhostFTPRemoteEditOpen", bridge)
        self.assertIn("//export GhostFTPRemoteEditSave", bridge)
        self.assertIn("bridgeState.engine.RemoteEditOpen", bridge)
        self.assertIn("bridgeState.engine.RemoteEditSave", bridge)
        self.assertIn("expectedRevision", bridge)
        self.assertIn("context.WithTimeout", bridge)
        self.assertNotIn("Upload(", bridge)
        self.assertNotIn("Download(", bridge)
        self.assertNotIn("exec.Command", bridge)
        self.assertNotIn("os.StartProcess", bridge)

    def test_appkit_editor_binds_to_visible_file_and_revision(self):
        source = self.read("macos/Sources/GhostFTPApp/main.swift")
        self.assertIn('NSButton(title: "Edit"', source)
        self.assertIn("remoteEditTapped", source)
        self.assertIn("GhostFTPRemoteEditOpen", source)
        self.assertIn("GhostFTPRemoteEditSave", source)
        self.assertIn("remoteEditGeneration", source)
        self.assertIn("expectedRevision", source)
        self.assertIn("NSTextView", source)
        self.assertIn("const MaxRemoteEditBytes int64 = 4 << 20", self.read("internal/api/remote_edit.go"))
        self.assertNotIn("NSWorkspace.shared.open", source)

    def test_save_revalidates_navigation_before_refresh(self):
        source = self.read("macos/Sources/GhostFTPApp/main.swift")
        start = source.index("private func saveRemoteEdit")
        end = source.index("private func", start + len("private func saveRemoteEdit"))
        body = source[start:end]
        self.assertIn("expectedRevision", body)
        self.assertIn("remoteEditGeneration", body)
        self.assertIn("remoteNavigationGeneration", body)
        self.assertIn("remoteCurrent == base", body)
        self.assertIn("refreshRemote(base)", body)

    def test_parity_marks_remote_edit_complete_only_with_implementation(self):
        parity = self.read("macos/PARITY.md")
        self.assertIn("- [x] Remote Edit", parity)


if __name__ == "__main__":
    unittest.main()
