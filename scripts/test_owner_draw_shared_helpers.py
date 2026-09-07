#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class OwnerDrawSharedHelperTests(unittest.TestCase):
    def test_site_manager_owner_draw_helpers_are_shared_without_retired_menu(self) -> None:
        helper = (ROOT / "internal/desktop/owner_draw_windows.go").read_text(encoding="utf-8")
        navigation = (ROOT / "internal/desktop/site_manager_navigation_windows.go").read_text(encoding="utf-8")
        site = (ROOT / "internal/desktop/site_manager_windows.go").read_text(encoding="utf-8")

        for marker in (
            "const wmMeasureItem = 0x002C",
            'fillRectW = user32.NewProc("FillRect")',
            "type measureItemStruct struct",
            "func measureItemFromLParam",
            "func measureItemToLParam",
            "rtlMoveMemory.Call",
        ):
            self.assertIn(marker, helper)

        for marker in ("measureItemFromLParam", "measureItemToLParam", "fillRectW.Call"):
            self.assertIn(marker, navigation)
        self.assertIn("case wmMeasureItem:", site)

        self.assertFalse((ROOT / "internal/desktop/menu_windows.go").exists())
        self.assertFalse((ROOT / "internal/desktop/menu_draw_windows.go").exists())
        for retired in ("measureMenuItem", "drawMenuItem", "applyDarkMenuBackground"):
            self.assertNotIn(retired, helper)


if __name__ == "__main__":
    unittest.main()
