#!/usr/bin/env python3
"""Regresije za ByFTP remote/transfer runtime hardening."""

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


if __name__ == "__main__":
    unittest.main()
