#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
JAVA = ROOT / "android/app/src/main/java/app/ghostftp/client"


class AndroidFileManagementContractTests(unittest.TestCase):
    def read(self, name: str) -> str:
        return (JAVA / name).read_text(encoding="utf-8")

    def test_remote_mutations_have_explicit_commands_and_safe_names(self) -> None:
        ftp = self.read("FtpSession.java")
        for marker in (
            "synchronized void makeDirectory(",
            'mutationCommand("MKD " + sanitizeArgument(path)',
            "synchronized void renameRemote(",
            'mutationCommand("RNFR " + sanitizeArgument(source)',
            'command("RNTO " + sanitizeArgument(target))',
            "synchronized void deleteRemote(",
            'String verb = directory ? "RMD " : "DELE ";',
            "private static String requireLeafName(",
            "name.indexOf('/') >= 0",
            "name.indexOf('\\\\') >= 0",
            '".".equals(name)',
            '"..".equals(name)',
        ):
            self.assertIn(marker, ftp)
        rename_start = ftp.index("synchronized void renameRemote(")
        rename_end = ftp.index("synchronized void deleteRemote(", rename_start)
        rename = ftp[rename_start:rename_end]
        self.assertIn("hardClose();", rename)
        self.assertIn("ambiguous rename state", rename)

    def test_local_file_management_stays_inside_saf(self) -> None:
        activity = self.read("MainActivity.java")
        for marker in (
            "private void createLocalFolder()",
            "private void renameLocalSelected()",
            "private void deleteLocalSelected()",
            "DocumentsContract.Document.MIME_TYPE_DIR",
            "DocumentsContract.createDocument",
            "DocumentsContract.renameDocument",
            "DocumentsContract.deleteDocument",
            "queryDocumentDisplayName",
            "ensureNoLocalNameConflict",
            "commitLocalMutation",
        ):
            self.assertIn(marker, activity)
        for forbidden in (
            "MANAGE_EXTERNAL_STORAGE",
            "Environment.getExternalStorage",
            "java.io.File(",
        ):
            self.assertNotIn(forbidden, activity)

    def test_remote_mutations_refresh_before_visible_commit(self) -> None:
        activity = self.read("MainActivity.java")
        for method, operation in (
            ("createRemoteFolder", "current.makeDirectory(base, safe);"),
            ("renameRemoteSelected", "current.renameRemote(source, safe);"),
            ("deleteRemoteSelected", "current.deleteRemote(target, entry.directory);"),
        ):
            start = activity.index(f"private void {method}()")
            next_method = activity.find("\n    private void ", start + 20)
            block = activity[start: next_method if next_method > start else len(activity)]
            self.assertIn(operation, block)
            self.assertIn("List<RemoteEntry> fresh = current.list(base);", block)
            self.assertLess(block.index(operation), block.index("List<RemoteEntry> fresh = current.list(base);"))
        self.assertIn("private void commitRemoteMutation(", activity)
        self.assertIn("session != current", activity)
        self.assertIn("!currentRemotePath.equals(base)", activity)

    def test_destructive_actions_require_confirmation_and_bookmarks_can_be_removed(self) -> None:
        activity = self.read("MainActivity.java")
        model = self.read("SiteProfile.java")
        self.assertIn('confirm("Delete local item?"', activity)
        self.assertIn('confirm("Delete server item?"', activity)
        self.assertIn('confirm("Delete saved site?"', activity)
        self.assertIn("private void removeLocalBookmark()", activity)
        self.assertIn("private void removeRemoteBookmark()", activity)
        self.assertIn("withoutLocalBookmark(index)", activity)
        self.assertIn("withoutRemoteBookmark(index)", activity)
        self.assertIn("SiteProfile withoutLocalBookmark(int index)", model)
        self.assertIn("SiteProfile withoutRemoteBookmark(int index)", model)

    def test_directory_selection_does_not_accidentally_transfer_directory_as_file(self) -> None:
        activity = self.read("MainActivity.java")
        upload_start = activity.index("private void uploadSelected()")
        download_start = activity.index("private void downloadSelected()", upload_start)
        upload = activity[upload_start:download_start]
        next_method = activity.index("private TransferProgress transferProgress", download_start)
        download = activity[download_start:next_method]
        self.assertIn("if (entry.directory)", upload)
        self.assertIn("Select a local file, not a directory, to upload.", upload)
        self.assertIn("if (entry.directory)", download)
        self.assertIn("Select a server file, not a directory, to download.", download)


if __name__ == "__main__":
    unittest.main()
