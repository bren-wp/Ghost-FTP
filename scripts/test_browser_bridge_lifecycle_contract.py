#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[1]
BACKGROUND = ROOT / "extensions" / "shared" / "background.js"
POPUP = ROOT / "extensions" / "shared" / "popup.js"


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def constant_ms(source: str, name: str) -> int:
    match = re.search(rf"const\s+{re.escape(name)}\s*=\s*(\d+)\s*;", source)
    if match is None:
        raise AssertionError(f"missing {name} constant")
    return int(match.group(1))


class BrowserBridgeLifecycleContractTests(unittest.TestCase):
    def test_background_times_out_unanswered_native_requests_before_popup(self) -> None:
        background = read(BACKGROUND)
        popup = read(POPUP)

        background_timeout = constant_ms(background, "REQUEST_TIMEOUT_MS")
        popup_timeout = constant_ms(popup, "REQUEST_TIMEOUT_MS")

        self.assertGreater(background_timeout, 0)
        self.assertLess(background_timeout, popup_timeout)
        for marker in (
            "function finishPending(id)",
            "clearTimeout(entry.timer)",
            "setTimeout(() => timeoutPending(request.id), REQUEST_TIMEOUT_MS)",
            "code: 'operation_timeout'",
            "closeNativeIfIdle();",
        ):
            self.assertIn(marker, background)

    def test_background_rejects_duplicate_request_ids(self) -> None:
        background = read(BACKGROUND)
        self.assertIn("pending.has(request.id)", background)
        self.assertIn("code: 'duplicate_request'", background)

    def test_transfer_event_polling_is_single_flight(self) -> None:
        background = read(BACKGROUND)
        for marker in (
            "let transferPollPending = false;",
            "pending.size >= MAX_PENDING || transferPollPending",
            "transferPollPending = true;",
            "transferPollPending = false;",
        ):
            self.assertIn(marker, background)
        self.assertNotIn("setInterval(", background)


if __name__ == "__main__":
    unittest.main()
