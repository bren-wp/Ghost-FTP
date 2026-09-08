#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsFileColumnContractTests(unittest.TestCase):
    def test_sidebar_uses_canonical_file_column_policy(self) -> None:
        sidebar = read("internal/desktop/sidebar_windows.go")
        self.assertIn("a.fitFileColumnsToWorkspace()", sidebar)
        self.assertNotIn("parts := []int{35, 16, 14, 22, 13}", sidebar)

    def test_actual_list_client_width_drives_file_columns(self) -> None:
        source = read("internal/desktop/file_columns_windows.go")
        self.assertIn("getClientRect.Call(spec.list", source)
        self.assertIn("fileColumnWidths(logicalWidth, spec.remote)", source)
        self.assertIn("lvmSetColumnWidth", source)

    def test_remote_compact_permissions_width_is_not_old_thirteen_percent_policy(self) -> None:
        source = read("internal/desktop/file_column_widths.go")
        self.assertIn("typeW, sizeW, modifiedW, permissionsW = 48, 48, 74, 84", source)
        self.assertIn("return []int{nameW, typeW, sizeW, modifiedW, permissionsW}", source)


if __name__ == "__main__":
    unittest.main()
