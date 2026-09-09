#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class QueuePriorityContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_core_reorders_only_queued_jobs(self) -> None:
        core = self.read("internal/transfer/queue_order.go")
        for marker in (
            "func (m *Manager) MoveQueuedUp",
            "func (m *Manager) MoveQueuedDown",
            'm.jobs[index].Status != "queued"',
            'm.jobs[i].Status == "queued"',
            "m.jobs[index], m.jobs[neighbor] = m.jobs[neighbor], m.jobs[index]",
            'Type:   "state"',
            "Jobs:   append([]model.TransferJob(nil), m.jobs...)",
            "Paused: m.paused",
        ):
            self.assertIn(marker, core)
        self.assertNotIn("m.pump()", core, "reordering must not start transfers as a side effect")

    def test_engine_exposes_bounded_queue_reordering_only(self) -> None:
        api = self.read("internal/api/queue_order.go")
        self.assertIn("MoveTransferUp", api)
        self.assertIn("e.transfers.MoveQueuedUp(id)", api)
        self.assertIn("MoveTransferDown", api)
        self.assertIn("e.transfers.MoveQueuedDown(id)", api)

    def test_shared_ui_policy_is_single_selection_and_queued_only(self) -> None:
        policy = self.read("internal/desktop/queue_priority.go")
        for marker in (
            "len(selected) != 1",
            'jobs[index].Status != "queued"',
            'jobs[i].Status == "queued"',
            "MoveUp = true",
            "MoveDown = true",
            "i18n.Normalize",
        ):
            self.assertIn(marker, policy)

    def test_windows_priority_controls_are_real_wired_controls(self) -> None:
        windows = self.read("internal/desktop/queue_priority_windows.go")
        commands = self.read("internal/desktop/commands_windows.go")
        actions = self.read("internal/desktop/action_state_windows.go")
        layout = self.read("internal/desktop/workspace_layout_windows.go")
        transfers = self.read("internal/desktop/transfers_windows.go")

        for marker in (
            "idMoveQueueUp",
            "idMoveQueueDown",
            'wstr("BUTTON")',
            "wsChild|wsVisible|wsTabStop|bsOwnerDraw",
            "a.engine.MoveTransferUp(id)",
            "a.engine.MoveTransferDown(id)",
            "queuePriorityWords(a.languageCode())",
        ):
            self.assertIn(marker, windows)
        self.assertIn("case idMoveQueueUp:", commands)
        self.assertIn("case idMoveQueueDown:", commands)
        self.assertIn("deriveQueuePriorityState", actions)
        self.assertIn("a.updateQueuePriorityControls(priorityState)", actions)
        self.assertIn("a.layoutQueuePriorityControls()", layout)
        self.assertIn("selected := a.selectedTransferIDSet()", transfers)
        self.assertIn("a.restoreTransferSelection(selected)", transfers)

    def test_tests_cover_scheduler_and_ui_invariants(self) -> None:
        manager_tests = self.read("internal/transfer/queue_order_test.go")
        ui_tests = self.read("internal/desktop/queue_priority_test.go")
        for marker in (
            "TestMoveQueuedUpAndDownChangesOnlySchedulerOrder",
            "TestMoveQueuedSkipsRunningAndTerminalSlots",
            "TestMoveQueuedRejectsMissingAndNonQueuedJobs",
            "TestMoveQueuedAtEdgeIsIdempotentAndDoesNotEmitNoise",
            "TestMoveQueuedEmitsFullStateSnapshot",
            "TestMoveQueuedRejectsClosedManager",
        ):
            self.assertIn(marker, manager_tests)
        self.assertIn("TestQueuePriorityStateRequiresOneQueuedSelection", ui_tests)
        self.assertIn("TestQueuePriorityWordsCoverEverySupportedLanguage", ui_tests)


if __name__ == "__main__":
    unittest.main()
