import pathlib
import unittest

ROOT = pathlib.Path(__file__).resolve().parents[1]


class RemoteEditMetadataRefreshContract(unittest.TestCase):
    def read(self, relative: str) -> str:
        return (ROOT / relative).read_text(encoding="utf-8")

    def test_windows_refreshes_listing_only_after_verified_save(self):
        source = self.read("internal/desktop/remote_edit_windows.go")
        save_start = source.index("func (a *app) saveRemoteTextEditor")
        reload_start = source.index("func (a *app) reloadRemoteTextEditor")
        save_body = source[save_start:reload_start]

        saved_status = save_body.index("a.setStatus(words.Saved)")
        refresh = save_body.index("a.refreshRemote(a.remoteCurrent)")
        reopen = save_body.index("a.showRemoteTextEditor(saved, next, next.Text, generation)")
        conflict_return = save_body.index("return", save_body.index("if err != nil"))

        self.assertLess(conflict_return, saved_status)
        self.assertLess(saved_status, refresh)
        self.assertLess(refresh, reopen)
        self.assertEqual(save_body.count("a.refreshRemote(a.remoteCurrent)"), 1)

    def test_linux_uses_quiet_serialized_metadata_refresh(self):
        source = self.read("internal/desktop/remote_edit_linux.go")

        self.assertIn("linuxActionRemoteEditMetadataRefresh", source)
        self.assertIn("func (u *linuxDesktop) refreshRemoteMetadataAfterEdit()", source)
        self.assertIn("u.engine.RemoteList(ctx, target)", source)
        self.assertIn("if pending == linuxRemoteEditPendingSave", source)
        self.assertIn("u.refreshRemoteMetadataAfterEdit()", source)
        self.assertIn("if result.action == linuxActionRemoteEditMetadataRefresh", source)
        self.assertIn("selectedName = item.Name", source)
        self.assertIn("u.remoteItems[index].Name == selectedName", source)

        refresh_start = source.index("func (u *linuxDesktop) refreshRemoteMetadataAfterEdit()")
        save_start = source.index("func (u *linuxDesktop) remoteEditSave()")
        refresh_body = source[refresh_start:save_start]
        self.assertNotIn("setStatus", refresh_body)

    def test_metadata_refresh_does_not_add_a_second_transfer_path(self):
        windows = self.read("internal/desktop/remote_edit_windows.go")
        linux = self.read("internal/desktop/remote_edit_linux.go")

        self.assertNotIn("exec.Command", windows + linux)
        self.assertNotIn("os.StartProcess", windows + linux)
        self.assertEqual(windows.count("RemoteEditSave"), 1)
        self.assertEqual(linux.count("RemoteEditSave"), 1)


if __name__ == "__main__":
    unittest.main()
