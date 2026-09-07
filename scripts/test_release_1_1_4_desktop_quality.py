#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class Release114DesktopQualityTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_language_combo_closes_and_persists_off_ui_thread(self) -> None:
        text = self.read("internal/desktop/localization_windows.go")
        change = text.split("func (a *app) changeLanguageFromUI()", 1)[1].split(
            "func retrySummary", 1
        )[0]

        self.assertIn("cbShowDropDown  = 0x014F", text)
        self.assertIn(
            "sendMessageW.Call(a.languageCombo, cbShowDropDown, 0, 0)", change
        )
        self.assertIn("enableWindow.Call(a.languageCombo, 0)", change)
        self.assertIn("enableWindow.Call(a.languageCombo, 1)", change)
        self.assertIn("a.goSafe(func() {", change)
        self.assertIn("a.dispatch(func() {", change)
        self.assertLess(change.index("a.goSafe(func() {"), change.index("a.engine.SetSettings(next)"))
        self.assertEqual(change.count("a.engine.SetSettings(next)"), 1)
        self.assertIn("displayedLanguage := a.languageCode()", change)
        self.assertIn("if i18n.Normalize(saved.Language) != displayedLanguage", change)

    def test_idle_transfer_poll_has_no_redraw_work(self) -> None:
        text = self.read("internal/desktop/transfers_windows.go")
        refresh = text.split("func (a *app) refreshTransfers()", 1)[1].split(
            "func (a *app) pauseTransfers()", 1
        )[0]

        self.assertIn("events, seq := a.engine.TransferEvents(a.transferSeq)", refresh)
        self.assertIn("if len(events) == 0 {\n\t\treturn\n\t}", refresh)
        self.assertLess(refresh.index("if len(events) == 0"), refresh.index("selected := a.selectedTransferIDSet()"))
        self.assertLess(refresh.index("if len(events) == 0"), refresh.index("a.updateTransferSummary()"))

    def test_windows_settings_share_backend_limits_and_defaults(self) -> None:
        text = self.read("internal/desktop/settings_windows.go")
        self.assertIn('"github.com/bren-wp/Ghost-FTP/internal/config"', text)
        self.assertIn("func normalizeSettingsForPrompt", text)
        for marker in (
            "config.DefaultSettings()",
            "config.MinParallelism",
            "config.MaxParallelism",
            "config.MinAutoRetryCount",
            "config.MaxAutoRetryCount",
            "config.MinRetryDelaySeconds",
            "config.MaxRetryDelaySeconds",
            "config.MinConnectionTimeoutSeconds",
            "config.MaxConnectionTimeoutSeconds",
        ):
            self.assertIn(marker, text)

        for stale in (
            'a.promptNumber("settings.parallel", settings.Parallelism, 1, 8)',
            'a.promptNumber("settings.timeout", settings.ConnectionTimeoutSeconds, 5, 60)',
            'a.promptNumber("settings.retries", settings.AutoRetryCount, 0, 3)',
            'a.promptNumber("settings.retry_delay", settings.RetryDelaySeconds, 1, 30)',
        ):
            self.assertNotIn(stale, text)

    def test_windows_icons_remain_local_and_registration_is_deduplicated(self) -> None:
        text = self.read("internal/desktop/icons_windows.go")
        self.assertIn('name = "Segoe Fluent Icons"', text)
        self.assertIn('name := "Segoe MDL2 Assets"', text)
        self.assertIn("func (a *app) registerButtonVisual", text)
        self.assertIn("return a.registerButtonVisual(hwnd, icon, label, variant, false)", text)
        self.assertIn("return a.registerButtonVisual(hwnd, icon, label, variant, true)", text)
        self.assertEqual(text.count("make(map[uintptr]buttonVisual)"), 1)
        self.assertNotIn("http://", text)
        self.assertNotIn("https://", text)


if __name__ == "__main__":
    unittest.main()
