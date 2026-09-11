#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class QueuePriorityContractTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_core_reorders_only_queued_jobs_and_preserves_nonqueued_slots(self) -> None:
        core = self.read("internal/transfer/queue_order.go")
        for marker in (
            "func (m *Manager) MoveQueuedTop",
            "func (m *Manager) MoveQueuedUp",
            "func (m *Manager) MoveQueuedDown",
            "func (m *Manager) MoveQueuedBottom",
            'm.jobs[index].Status != "queued"',
            'm.jobs[i].Status != "queued"',
            "queuedSlots = append(queuedSlots, i)",
            "m.jobs[queuedSlots[pos]] = m.jobs[queuedSlots[pos-1]]",
            "m.jobs[queuedSlots[pos]] = m.jobs[queuedSlots[pos+1]]",
            'Type:   "state"',
            "Jobs:   append([]model.TransferJob(nil), m.jobs...)",
            "Paused: m.paused",
        ):
            self.assertIn(marker, core)
        self.assertNotIn("m.pump()", core, "reordering must not start transfers as a side effect")
        self.assertNotIn("jobConnections[", core, "reordering must not rewrite connection identity bindings")

    def test_engine_exposes_four_bounded_queue_reorder_actions(self) -> None:
        api = self.read("internal/api/queue_order.go")
        for method, manager in (
            ("MoveTransferTop", "MoveQueuedTop"),
            ("MoveTransferUp", "MoveQueuedUp"),
            ("MoveTransferDown", "MoveQueuedDown"),
            ("MoveTransferBottom", "MoveQueuedBottom"),
        ):
            self.assertIn(method, api)
            self.assertIn(f"e.transfers.{manager}(id)", api)

    def test_shared_ui_policy_is_single_selection_queued_only_and_four_way(self) -> None:
        policy = self.read("internal/desktop/queue_priority.go")
        for marker in (
            "len(selected) != 1",
            'jobs[index].Status != "queued"',
            'jobs[i].Status == "queued"',
            "MoveTop = true",
            "MoveUp = true",
            "MoveDown = true",
            "MoveBottom = true",
            "queuePriorityTop",
            "queuePriorityBottom",
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
            "idMoveQueueTop",
            "idMoveQueueUp",
            "idMoveQueueDown",
            "idMoveQueueBottom",
            'wstr("BUTTON")',
            "wsChild|wsVisible|wsTabStop|bsOwnerDraw",
            "a.engine.MoveTransferTop(id)",
            "a.engine.MoveTransferUp(id)",
            "a.engine.MoveTransferDown(id)",
            "a.engine.MoveTransferBottom(id)",
            "queuePriorityWords(a.languageCode())",
        ):
            self.assertIn(marker, windows)
        for marker in (
            "case idMoveQueueTop:",
            "case idMoveQueueUp:",
            "case idMoveQueueDown:",
            "case idMoveQueueBottom:",
        ):
            self.assertIn(marker, commands)
        self.assertIn("deriveQueuePriorityState", actions)
        self.assertIn("a.updateQueuePriorityControls(priorityState)", actions)
        self.assertIn("a.layoutQueuePriorityControls()", layout)
        self.assertIn("selected := a.selectedTransferIDSet()", transfers)
        self.assertIn("a.restoreTransferSelection(selected)", transfers)

    def test_linux_priority_controls_are_rendered_click_wired_and_identity_safe(self) -> None:
        linux = self.read("internal/desktop/queue_priority_linux.go")
        gui = self.read("internal/desktop/gui_linux.go")

        for marker in (
            "deriveQueuePriorityState",
            "queuePriorityWords(u.language)",
            "u.engine.MoveTransferTop(id)",
            "u.engine.MoveTransferUp(id)",
            "u.engine.MoveTransferDown(id)",
            "u.engine.MoveTransferBottom(id)",
            "u.transferJobs = u.engine.Transfers()",
            "u.restoreQueuePrioritySelection(id)",
            "func (u *linuxDesktop) renderQueuePriorityControls() error",
            "func (u *linuxDesktop) handleQueuePriorityMouse(x, y int) bool",
        ):
            self.assertIn(marker, linux)
        self.assertIn("u.renderQueuePriorityControls()", gui)
        self.assertIn("u.handleQueuePriorityMouse(x, y)", gui)

    def test_tree_transfer_dependencies_are_prepared_before_queue_commit(self) -> None:
        tree = self.read("internal/api/tree_transfer.go")
        upload = tree[tree.index("func (s *Engine) addUploadTree"):tree.index("func buildDownloadPlan")]
        download = tree[tree.index("func (s *Engine) addDownloadTree"):tree.index("type localDirectoryPreparationHooks")]
        self.assertLess(upload.index("ensureRemoteDirectoryCached"), upload.index("reservation.Commit()"))
        self.assertLess(download.index("prepareLocalDirectories"), download.index("reservation.Commit()"))

    def test_docs_bind_queue_priority_to_current_release_and_safety_contract(self) -> None:
        version = self.read("VERSION").strip()
        roadmap = self.read("docs/ROADMAP.md")
        detail = self.read("docs/QUEUE-PRIORITY.md")

        self.assertIn(f"Ghost FTP **{version}**", roadmap)
        self.assertIn(f"Status: implemented in Ghost FTP {version}", roadmap)
        self.assertIn(f"Ghost FTP **{version}** includes queue priority/reordering", detail)
        self.assertIn("MoveTransferTop(id)", detail)
        self.assertIn("MoveTransferBottom(id)", detail)
        self.assertIn("`jobConnections` is not rewritten", detail)
        self.assertIn("reservation.Commit()", detail)
        self.assertIn(f"Root `VERSION` is **{version}**", detail)
        self.assertIn("exact-head tests/builds", detail)
        self.assertNotIn("post-0.0.3 source line", detail)
        self.assertNotIn("Root `VERSION` remains **0.0.3**", detail)

    def test_tests_cover_scheduler_and_ui_invariants(self) -> None:
        manager_tests = self.read("internal/transfer/queue_order_test.go")
        ui_tests = self.read("internal/desktop/queue_priority_test.go")
        linux_tests = self.read("internal/desktop/queue_priority_linux_test.go")
        for marker in (
            "TestMoveQueuedUpAndDownChangesOnlySchedulerOrder",
            "TestMoveQueuedTopAndBottomPreserveQueuedRelativeOrder",
            "TestMoveQueuedSkipsRunningAndTerminalSlots",
            "TestMoveQueuedRejectsMissingAndNonQueuedJobs",
            "TestMoveQueuedAtEdgeIsIdempotentAndDoesNotEmitNoise",
            "TestMoveQueuedEmitsFullStateSnapshot",
            "TestMoveQueuedRejectsClosedManager",
            "reorder changed connection binding",
        ):
            self.assertIn(marker, manager_tests)
        self.assertIn("TestQueuePriorityStateRequiresOneQueuedSelection", ui_tests)
        self.assertIn("TestQueuePriorityWordsCoverEverySupportedLanguage", ui_tests)
        self.assertIn("TestLinuxQueuePriorityRectsFollowClearQueueWithoutOverlap", linux_tests)
        self.assertIn("TestLinuxQueuePriorityStateUsesSharedQueuedOnlyPolicy", linux_tests)
        self.assertIn("TestLinuxQueuePrioritySelectionRestoresByTransferID", linux_tests)


if __name__ == "__main__":
    unittest.main()
