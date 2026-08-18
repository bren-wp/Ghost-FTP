#!/usr/bin/env python3

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class StateDurabilitySourceTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_store_syncs_temp_before_replace_and_directory_after_replace(self) -> None:
        store = self.read("internal/config/store.go")
        for marker in (
            "func writeSyncedTemp(",
            "f.Sync()",
            "func replaceSyncedGeneration(",
            "replaceFile(tmp, dst)",
            "syncStateDirectory(dir)",
            'replaceSyncedGeneration(s.dir, prevTmp, path+".previous")',
            "replaceSyncedGeneration(s.dir, tmp, path)",
        ):
            self.assertIn(marker, store)

    def test_unix_replace_has_directory_sync(self) -> None:
        other = self.read("internal/config/replace_other.go")
        for marker in (
            "func syncStateDirectory(dir string) error",
            "os.Open(dir)",
            "st.IsDir()",
            "f.Sync()",
        ):
            self.assertIn(marker, other)

    def test_windows_replace_keeps_write_through(self) -> None:
        windows = self.read("internal/config/replace_windows.go")
        self.assertIn("moveFileWriteThrough", windows)
        self.assertIn("moveFileReplaceExisting|moveFileWriteThrough", windows)
        self.assertIn("func syncStateDirectory(string) error { return nil }", windows)


if __name__ == "__main__":
    unittest.main()
