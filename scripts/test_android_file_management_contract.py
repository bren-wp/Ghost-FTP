#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
ACTIVITY = ROOT / "android/app/src/main/java/app/ghostftp/client/MainActivity.java"
SESSION = ROOT / "android/app/src/main/java/app/ghostftp/client/FtpSession.java"


class AndroidFileManagementContractTests(unittest.TestCase):
    def test_local_scoped_file_management_is_real(self) -> None:
        text = ACTIVITY.read_text(encoding="utf-8")
        for marker in (
            'localCreateDirectory = button("New folder")',
            'localRename = button("Rename")',
            'localDelete = dangerButton("Delete")',
            "DocumentsContract.createDocument(getContentResolver(), parent, DocumentsContract.Document.MIME_TYPE_DIR, child)",
            "DocumentsContract.renameDocument(getContentResolver(), document, child)",
            "DocumentsContract.deleteDocument(getContentResolver(), document)",
            "setOnItemLongClickListener",
            "ensureNoLocalNameConflict(",
            "queryDocumentDisplayName(",
        ):
            self.assertIn(marker, text)
        for forbidden in ("MANAGE_EXTERNAL_STORAGE", "java.io.File(" ):
            self.assertNotIn(forbidden, text)

    def test_remote_file_management_routes_through_validated_session(self) -> None:
        activity = ACTIVITY.read_text(encoding="utf-8")
        session = SESSION.read_text(encoding="utf-8")
        for marker in (
            'remoteCreateDirectory = button("New folder")',
            'remoteRename = button("Rename")',
            'remoteDelete = dangerButton("Delete")',
            'remoteChmod = button("Permissions")',
            "current.createDirectory(",
            "current.rename(",
            "current.delete(",
            "current.chmod(",
            "runRemoteMutation(",
            "List<RemoteEntry> fresh = current.list(directory);",
        ):
            self.assertIn(marker, activity)
        for marker in (
            "synchronized void createDirectory(",
            "synchronized void rename(",
            "synchronized void delete(",
            "synchronized void chmod(",
            "requireMutableRemotePath(",
            "normalizeChmodMode(",
            '".".equals(segment)',
            '"..".equals(segment)',
        ):
            self.assertIn(marker, session)

    def test_destructive_actions_are_explicit_and_names_are_single_segment(self) -> None:
        text = ACTIVITY.read_text(encoding="utf-8")
        self.assertGreaterEqual(text.count("confirmDestructive("), 3)
        for marker in (
            'if (".".equals(name) || "..".equals(name))',
            "name.indexOf('/') >= 0",
            "name.indexOf('\\\\') >= 0",
            "name.indexOf('\\0') >= 0",
            "name.indexOf('\\r') >= 0",
            "name.indexOf('\\n') >= 0",
        ):
            self.assertIn(marker, text)


if __name__ == "__main__":
    unittest.main()
