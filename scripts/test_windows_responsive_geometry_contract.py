#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class WindowsResponsiveGeometryContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_dpi_clamp_resolves_the_destination_monitor(self) -> None:
        geometry = self.read("internal/desktop/window_geometry_windows.go")
        clamp = geometry.split("func (a *app) clampSuggestedWindowRectToWorkArea", 1)[1]
        self.assertIn('NewProc("MonitorFromRect")', geometry)
        self.assertIn("monitorFromRectResponsive.Call", clamp)
        self.assertNotIn("monitorFromWindowResponsive.Call(a.hwnd", clamp)

    def test_minimum_and_startup_geometry_are_work_area_aware(self) -> None:
        windows = self.read("internal/desktop/windows.go")
        geometry = self.read("internal/desktop/window_geometry_windows.go")
        pure = self.read("internal/desktop/window_geometry.go")
        self.assertIn("a.responsiveWindowBounds()", windows)
        self.assertIn("a.responsiveMinTrackSize()", windows)
        self.assertIn("monitorWorkAreaLogical()", geometry)
        self.assertIn("responsiveWindowBoundsForWorkArea", pure)
        self.assertIn("responsiveMinimumTrackSize", pure)


if __name__ == "__main__":
    unittest.main()
