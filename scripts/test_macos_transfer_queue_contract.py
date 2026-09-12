from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        raise AssertionError(f"missing required transfer queue file: {relative}")
    return path.read_text(encoding="utf-8")


def read_bridge_package() -> str:
    files = sorted((ROOT / "macos/Bridge").glob("*.go"))
    return "\n".join(path.read_text(encoding="utf-8") for path in files)


class MacOSTransferQueueContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.bridge = read_bridge_package()
        cls.swift = read("macos/Sources/GhostFTPApp/main.swift")
        cls.api = read("internal/api/transfer_snapshot.go") + "\n" + read("internal/api/queue_order.go") + "\n" + read("internal/api/engine.go")
        cls.manager = read("internal/transfer/manager.go")
        cls.windows = read("internal/desktop/transfers_windows.go") + "\n" + read("internal/desktop/queue_priority_windows.go")
        cls.parity = read("macos/PARITY.md")

    def test_shared_engine_remains_queue_authority(self):
        for marker in (
            "TransferQueueSnapshot",
            "PauseTransfers",
            "ResumeTransfers",
            "CancelTransfers",
            "RetryTransfers",
            "ClearFinishedTransfers",
            "MoveTransferTop",
            "MoveTransferUp",
            "MoveTransferDown",
            "MoveTransferBottom",
        ):
            self.assertIn(marker, self.api)
        self.assertIn("paused", self.manager)
        self.assertIn("m.jobs", self.manager)

    def test_bridge_exposes_snapshot_not_second_scheduler(self):
        for marker in (
            "GhostFTPRefreshTransferQueue",
            "engine.TransferQueueSnapshot",
            "GhostFTPTransferQueuePaused",
            "GhostFTPTransferCount",
            "GhostFTPTransferID",
            "GhostFTPTransferStatus",
            "GhostFTPTransferProgress",
            "GhostFTPTransferBytesPerSecond",
            "GhostFTPTransferETASeconds",
        ):
            self.assertIn(marker, self.bridge)
        for forbidden in ("go func(", "session.Upload", "session.Download", "exec.Command"):
            self.assertNotIn(forbidden, read("macos/Bridge/transfer_queue.go"))

    def test_bridge_actions_use_shared_batch_and_order_apis(self):
        queue_bridge = read("macos/Bridge/transfer_queue.go")
        for marker in (
            "engine.PauseTransfers()",
            "engine.ResumeTransfers()",
            "engine.CancelTransfers(ids)",
            "engine.RetryTransfers(ids)",
            "engine.ClearFinishedTransfers()",
            "engine.MoveTransferTop(id)",
            "engine.MoveTransferUp(id)",
            "engine.MoveTransferDown(id)",
            "engine.MoveTransferBottom(id)",
        ):
            self.assertIn(marker, queue_bridge)
        self.assertIn("maxQueueSelection", queue_bridge)

    def test_native_queue_surface_has_all_windows_actions(self):
        for marker in (
            "transferQueueTable",
            'NSButton(title: "Pause"',
            'NSButton(title: "Resume"',
            'NSButton(title: "Cancel"',
            'NSButton(title: "Retry"',
            'NSButton(title: "Clear Finished"',
            'NSButton(title: "Top"',
            'NSButton(title: "Up"',
            'NSButton(title: "Down"',
            'NSButton(title: "Bottom"',
            "refreshTransferQueue",
            "restoreTransferQueueSelection",
        ):
            self.assertIn(marker, self.swift)

    def test_action_enablement_uses_authoritative_statuses_and_paused_state(self):
        for marker in (
            'status == "queued" || status == "running"',
            'status == "failed" || status == "cancelled"',
            'status == "queued"',
            "transferQueuePaused",
            "GhostFTPTransferQueuePaused()",
        ):
            self.assertIn(marker, self.swift)
        self.assertIn("selectedTransferIDs", self.swift)

    def test_reordering_requires_one_queued_job_and_restores_id_selection(self):
        for marker in (
            "selectedQueueReorderCandidate",
            "queuedBefore",
            "queuedAfter",
            "selectedTransferIDs",
            "restoreTransferQueueSelection",
        ):
            self.assertIn(marker, self.swift)
        self.assertIn("MoveTransferTop", self.windows)
        self.assertIn("MoveTransferBottom", self.windows)

    def test_queue_refresh_is_bounded_and_uses_shared_metrics(self):
        for marker in (
            "Timer.scheduledTimer",
            "timeInterval: 1.0",
            "bytesTransferred",
            "bytesTotal",
            "bytesPerSecond",
            "etaSeconds",
        ):
            self.assertIn(marker, self.swift)

    def test_parity_marks_all_transfer_queue_actions_complete(self):
        for action in (
            "Pause Queue",
            "Resume Queue",
            "Cancel Transfer",
            "Retry Transfer",
            "Clear Finished",
            "Move Top",
            "Move Up",
            "Move Down",
            "Move Bottom",
        ):
            self.assertIn(f"- [x] {action}", self.parity)


if __name__ == "__main__":
    unittest.main()
