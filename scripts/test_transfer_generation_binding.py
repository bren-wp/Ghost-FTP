#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class TransferGenerationBindingTests(unittest.TestCase):
    def test_reserve_captures_generation_before_identity_and_rechecks_after(self) -> None:
        text = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        start = text.index("func (m *Manager) ReserveBatch")
        end = text.index("func (b *BatchReservation) Cancel", start)
        block = text[start:end]
        capture = block.index("generation := m.generation")
        identity = block.index("m.remote.ConnectionIdentity()")
        recheck = block.index("m.generation != generation")
        reserve = block.index("m.reserved += len(jobs)")
        self.assertLess(capture, identity)
        self.assertLess(identity, recheck)
        self.assertLess(recheck, reserve)

    def test_retry_captures_generation_before_identity_and_rechecks_before_mutation(self) -> None:
        text = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        start = text.index("func (m *Manager) RetryBatch")
        end = text.index("func (m *Manager) ClearFinished", start)
        block = text[start:end]
        capture = block.index("generation := m.generation")
        identity = block.index("m.remote.ConnectionIdentity()")
        recheck = block.index("m.generation != generation")
        mutation = block.index('m.jobs[i].Status = "queued"')
        self.assertLess(capture, identity)
        self.assertLess(identity, recheck)
        self.assertLess(recheck, mutation)

    def test_identity_lookup_is_not_performed_while_transfer_mutex_is_held(self) -> None:
        text = (ROOT / "internal/transfer/manager.go").read_text(encoding="utf-8")
        for signature, end_marker in (
            ("func (m *Manager) ReserveBatch", "func (b *BatchReservation) Cancel"),
            ("func (m *Manager) RetryBatch", "func (m *Manager) ClearFinished"),
        ):
            start = text.index(signature)
            end = text.index(end_marker, start)
            block = text[start:end]
            identity = block.index("m.remote.ConnectionIdentity()")
            unlock_before = block.rfind("m.mu.Unlock()", 0, identity)
            lock_after = block.find("m.mu.Lock()", identity)
            self.assertNotEqual(unlock_before, -1, signature)
            self.assertNotEqual(lock_after, -1, signature)
            self.assertLess(unlock_before, identity, signature)
            self.assertGreater(lock_after, identity, signature)


if __name__ == "__main__":
    unittest.main()
