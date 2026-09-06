#!/usr/bin/env python3
"""Ghost FTP desktop runtime and release hardening regressions."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class RuntimeHardeningTests(unittest.TestCase):
    def test_remote_session_cannot_escape_lifecycle_guard(self) -> None:
        manager = (ROOT / "internal/remote/manager.go").read_text(encoding="utf-8")
        engine = (ROOT / "internal/api/engine.go").read_text(encoding="utf-8")
        self.assertNotIn("func (m *Manager) Session()", manager)
        self.assertNotIn("remote.Session()", engine)
        self.assertIn("func (m *Manager) Operation(ctx context.Context)", manager)
        self.assertIn("ctx = nonNilContext(ctx)", manager)

    def test_transfer_reads_use_shared_lock(self) -> None:
        source = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        self.assertIn("mu             sync.RWMutex", source)
        for signature in (
            "func (m *Manager) List() []model.TransferJob",
            "func (m *Manager) Events(since int64) ([]Event, int64)",
            "func (m *Manager) ActiveCount() int",
            "func (m *Manager) jobSnapshot(id string) (model.TransferJob, bool)",
        ):
            start = source.index(signature)
            body = source[start : start + 260]
            self.assertIn("m.mu.RLock()", body, signature)
            self.assertIn("m.mu.RUnlock()", body, signature)

    def test_transfer_selection_validation_is_centralized(self) -> None:
        source = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        self.assertEqual(source.count("func selectedIDs("), 1)
        self.assertGreaterEqual(source.count("selectedIDs(ids)"), 2)
        self.assertIn("if ctx == nil", source[source.index("func (m *Manager) waitWorkers") :])

    def test_retired_web_runtime_is_absent(self) -> None:
        self.assertFalse((ROOT / "GhostFTP WEB").exists())
        for rel in (
            "scripts/package_web.py",
            "scripts/test_package_web.py",
            "scripts/audit_web.py",
        ):
            self.assertFalse((ROOT / rel).exists(), f"retired Web tooling exists: {rel}")

    def test_release_workflow_requires_delayed_remote_readback(self) -> None:
        source = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
        immediate = "assert_release_asset_set immediate"
        delay = "sleep 5"
        main_guard = "main moved before delayed release verification"
        delayed = "assert_release_asset_set delayed"
        for marker in (
            "assert_release_asset_set()",
            "RELEASE_ASSET_READBACK=PASS",
            "gh release view",
            "--json assets",
            "gh release download",
            "SHA256.txt",
            "remote_prerelease=",
            immediate,
            delay,
            main_guard,
            delayed,
        ):
            self.assertIn(marker, source)

        immediate_pos = source.index(immediate)
        delay_pos = source.index(delay, immediate_pos)
        main_guard_pos = source.index(main_guard, delay_pos)
        delayed_pos = source.index(delayed, main_guard_pos)
        self.assertLess(immediate_pos, delay_pos)
        self.assertLess(delay_pos, main_guard_pos)
        self.assertLess(main_guard_pos, delayed_pos)


if __name__ == "__main__":
    unittest.main()
