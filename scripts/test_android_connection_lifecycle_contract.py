#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
ACTIVITY = ROOT / "android/app/src/main/java/app/ghostftp/client/MainActivity.java"
SESSION = ROOT / "android/app/src/main/java/app/ghostftp/client/FtpSession.java"


def read(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def method_body(source: str, signature: str) -> str:
    start = source.index(signature)
    opening = source.index("{", start)
    depth = 0
    for index in range(opening, len(source)):
        char = source[index]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return source[opening + 1 : index]
    raise AssertionError(f"unterminated method: {signature}")


class AndroidConnectionLifecycleContractTests(unittest.TestCase):
    def test_activity_owns_pending_connection_until_ui_commit(self) -> None:
        source = read(ACTIVITY)
        self.assertIn("private volatile FtpSession connectingSession;", source)
        self.assertIn("private volatile boolean lifecycleDestroyed;", source)
        self.assertIn("private boolean connectionAttemptCurrent(FtpSession candidate)", source)
        helper = method_body(source, "private boolean connectionAttemptCurrent(FtpSession candidate)")
        self.assertIn("!lifecycleDestroyed", helper)
        self.assertIn("connectingSession == candidate", helper)

    def test_destroy_aborts_pending_connection_before_executor_shutdown(self) -> None:
        source = read(ACTIVITY)
        body = method_body(source, "protected void onDestroy()")
        for marker in (
            "lifecycleDestroyed = true;",
            "FtpSession pending = connectingSession;",
            "connectingSession = null;",
            "pending.abort();",
            "io.shutdownNow();",
        ):
            self.assertIn(marker, body)
        self.assertLess(body.index("lifecycleDestroyed = true;"), body.index("pending.abort();"))
        self.assertLess(body.index("pending.abort();"), body.index("io.shutdownNow();"))

    def test_connect_registers_attempt_before_background_work_and_rejects_stale_commit(self) -> None:
        source = read(ACTIVITY)
        body = method_body(source, "private void connect()")
        for marker in (
            "final FtpSession next;",
            "next = new FtpSession(",
            "connectingSession = next;",
            "io.execute(() -> {",
            "if (!connectionAttemptCurrent(next))",
            "next.abort();",
            "connectingSession = null;",
            "session = next;",
        ):
            self.assertIn(marker, body)
        self.assertLess(body.index("connectingSession = next;"), body.index("io.execute(() -> {"))
        self.assertLess(body.index("if (!connectionAttemptCurrent(next))"), body.index("session = next;"))

    def test_connect_failure_does_not_post_to_destroyed_activity(self) -> None:
        source = read(ACTIVITY)
        body = method_body(source, "private void connect()")
        self.assertIn("if (!connectionAttemptCurrent(next)) return;", body)
        post_error = method_body(source, "private void postError(")
        self.assertIn("if (lifecycleDestroyed) return;", post_error)

    def test_session_abort_is_nonblocking_and_cancels_connect_races(self) -> None:
        source = read(SESSION)
        self.assertIn("private volatile boolean aborted;", source)
        abort = method_body(source, "void abort()")
        self.assertIn("aborted = true;", abort)
        self.assertIn("connected = false;", abort)
        self.assertIn("closeQuietly(activeDataSocket);", abort)
        self.assertIn("closeQuietly(controlSocket);", abort)
        self.assertNotIn("synchronized void abort()", source)

        connect = method_body(source, "synchronized void connect(")
        self.assertGreaterEqual(connect.count("requireNotAborted();"), 2)
        self.assertIn("controlSocket = plain;", connect)
        self.assertLess(connect.index("controlSocket = plain;"), connect.index("requireNotAborted();", connect.index("controlSocket = plain;")))

        guard = method_body(source, "private void requireNotAborted()")
        self.assertIn("if (aborted)", guard)
        self.assertIn("hardClose();", guard)
        self.assertIn('throw new IOException("Connection cancelled.");', guard)


if __name__ == "__main__":
    unittest.main()
