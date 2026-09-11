#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
JAVA = ROOT / "android/app/src/main/java/app/ghostftp/client"


class AndroidDarkUiContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_android_palette_matches_canonical_desktop_dark_theme(self) -> None:
        colors = self.read("android/app/src/main/res/values/colors.xml")
        expected = {
            "ghost_window": "#0B0F17",
            "ghost_panel": "#121824",
            "ghost_list": "#161D2A",
            "ghost_border": "#2C3648",
            "ghost_text": "#F2F5FA",
            "ghost_muted": "#97A3B8",
            "ghost_accent": "#5B7CFA",
            "ghost_accent_strong": "#7A98FF",
            "ghost_success": "#4AD79B",
            "ghost_warn": "#F2BA55",
            "ghost_danger": "#FF6878",
            "ghost_selection": "#202F50",
        }
        for name, value in expected.items():
            self.assertIn(f'<color name="{name}">{value}</color>', colors)

        desktop = self.read("internal/uipalette/palette.go")
        for rgb in (
            "RGB{0x0B, 0x0F, 0x17}",
            "RGB{0x12, 0x18, 0x24}",
            "RGB{0x16, 0x1D, 0x2A}",
            "RGB{0x2C, 0x36, 0x48}",
            "RGB{0xF2, 0xF5, 0xFA}",
            "RGB{0x97, 0xA3, 0xB8}",
            "RGB{0x5B, 0x7C, 0xFA}",
            "RGB{0x7A, 0x98, 0xFF}",
            "RGB{0x4A, 0xD7, 0x9B}",
            "RGB{0xF2, 0xBA, 0x55}",
            "RGB{0xFF, 0x68, 0x78}",
            "RGB{0x20, 0x2F, 0x50}",
        ):
            self.assertIn(rgb, desktop)

    def test_theme_resources_use_canonical_palette(self) -> None:
        styles = self.read("android/app/src/main/res/values/styles.xml")
        for marker in (
            "@color/ghost_window",
            "@color/ghost_accent",
            "@color/ghost_text",
            "@color/ghost_muted",
            "android:windowLightStatusBar\">false",
            "android:windowLightNavigationBar\">false",
        ):
            self.assertIn(marker, styles)

    def test_runtime_workspace_uses_dark_theme_and_responsive_panes(self) -> None:
        activity = (JAVA / "MainActivity.java").read_text(encoding="utf-8")
        theme = (JAVA / "GhostTheme.java").read_text(encoding="utf-8")
        for marker in (
            "GhostTheme.WINDOW",
            "GhostTheme.PANEL",
            "GhostTheme.LIST",
            "GhostTheme.styleSpinner",
            "GhostTheme.styleList",
            "GhostTheme.spinnerAdapter",
            "GhostTheme.listAdapter",
            "screenWidthDp >= 700",
            'connectionState.setText("TRANSFER ACTIVE")',
            'connectionState.setText(secure ? "FTPS CONNECTED" : "FTP CONNECTED")',
            'card("LOCAL"',
            'card("SERVER"',
            'card("TRANSFERS"',
        ):
            self.assertIn(marker, activity)
        self.assertNotIn("Color.rgb(16, 19, 23)", activity)
        self.assertNotIn("Color.rgb(94, 214, 200)", activity)
        self.assertIn("RippleDrawable", theme)
        self.assertIn("statusColor", theme)

    def test_lists_support_navigation_and_long_press_management_selection(self) -> None:
        activity = (JAVA / "MainActivity.java").read_text(encoding="utf-8")
        self.assertIn("localList.setOnItemClickListener", activity)
        self.assertIn("localList.setOnItemLongClickListener", activity)
        self.assertIn("remoteList.setOnItemClickListener", activity)
        self.assertIn("remoteList.setOnItemLongClickListener", activity)
        self.assertIn("Selected local item for management", activity)
        self.assertIn("Selected server item for management", activity)


if __name__ == "__main__":
    unittest.main()
