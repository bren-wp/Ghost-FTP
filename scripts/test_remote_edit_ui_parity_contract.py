import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class RemoteEditUIParityContract(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_windows_uses_built_in_editor_and_shared_engine(self):
        workflow = self.read("internal/desktop/remote_edit_windows.go")
        dialog = self.read("internal/platform/text_editor_windows.go")
        commands = self.read("internal/desktop/commands_windows.go")
        ui = self.read("internal/desktop/ui_windows.go")

        self.assertIn("RemoteEditOpen", workflow)
        self.assertIn("RemoteEditSave", workflow)
        self.assertIn("ErrRemoteEditConflict", workflow)
        self.assertIn("TextEditorDialog", workflow)
        self.assertIn("idRemoteEdit", commands)
        self.assertIn("remoteEditAction()", commands)
        self.assertIn("storeRemoteEditButton", ui)
        self.assertIn("remoteEditButton(a)", ui)
        self.assertIn("esMultiline", dialog)
        self.assertIn("case 'S':", dialog)
        self.assertIn("case 'R':", dialog)
        self.assertNotIn("exec.Command", workflow + dialog)
        self.assertNotIn("os.StartProcess", workflow + dialog)

    def test_linux_uses_built_in_overlay_and_shared_engine(self):
        workflow = self.read("internal/desktop/remote_edit_linux.go")
        ui = self.read("internal/desktop/gui_linux.go")

        self.assertIn("RemoteEditOpen", workflow)
        self.assertIn("RemoteEditSave", workflow)
        self.assertIn("ErrRemoteEditConflict", workflow)
        self.assertIn("renderRemoteEditorOverlay", workflow)
        self.assertIn("handleRemoteEditKey", ui)
        self.assertIn("handleRemoteEditorMouse", ui)
        self.assertIn("renderRemoteEditButton", ui)
        self.assertIn("handleRemoteEditResult", ui)
        self.assertNotIn("exec.Command", workflow)
        self.assertNotIn("os.StartProcess", workflow)

    def test_both_frontends_share_safety_contract(self):
        common = self.read("internal/desktop/remote_edit_buffer.go")
        windows = self.read("internal/desktop/remote_edit_windows.go")
        linux = self.read("internal/desktop/remote_edit_linux.go")

        self.assertIn("errRemoteEditMixedNewlines", common)
        self.assertIn("remoteEditNewlineCRLF", common)
        self.assertIn("remoteEditNewlineCR", common)
        self.assertIn("api.MaxRemoteEditBytes", windows)
        self.assertIn("api.MaxRemoteEditBytes", linux)
        self.assertIn("newRemoteEditBuffer", windows)
        self.assertIn("newRemoteEditBuffer", linux)
        self.assertIn("REMOTE_EDIT_UI_PARITY=WINDOWS_LINUX_BUILTIN", common)


if __name__ == "__main__":
    unittest.main()
