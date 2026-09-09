#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WindowsVerifiedCleanupContractTests(unittest.TestCase):
    def test_verified_handle_blocks_replacement_and_deletes_same_object(self):
        delete_impl = (ROOT / "internal/platform/delete_windows.go").read_text(encoding="utf-8")

        self.assertIn("deleteFileAccess", delete_impl)
        self.assertIn("fileShareRead", delete_impl)
        self.assertIn("fileFlagOpenReparsePoint", delete_impl)
        self.assertIn('NewProc("SetFileInformationByHandle")', delete_impl)
        self.assertIn("fileDispositionInfo", delete_impl)
        self.assertIn("openVerifiedRegularFile(path, true)", delete_impl)
        self.assertIn("deleteVerifiedOpenFile(f)", delete_impl)
        self.assertIn("errVerifiedOwnershipDigestMismatch", delete_impl)

    def test_shortcut_digest_and_delete_use_verified_handle_helpers(self):
        shortcut = (ROOT / "internal/platform/shortcut_windows.go").read_text(encoding="utf-8")

        self.assertIn("return verifiedRegularFileSHA256(path)", shortcut)
        self.assertIn("removeVerifiedRegularFileMatchingSHA256(path, expectedDigest)", shortcut)
        self.assertIn("errors.Is(err, errVerifiedOwnershipDigestMismatch)", shortcut)
        self.assertNotIn("os.Remove(path)", shortcut)
        self.assertNotIn("confirmed, err := stableShortcutDigest(path)", shortcut)


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_VERIFIED_CLEANUP_HANDLE=PINNED")
