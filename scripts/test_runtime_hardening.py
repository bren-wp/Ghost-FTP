#!/usr/bin/env python3
"""Ghost FTP remote/transfer and release-runtime hardening regressions."""

from __future__ import annotations

import shutil
import subprocess
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

    def test_web_runtime_regressions(self) -> None:
        php = shutil.which("php")
        if php is None:
            self.skipTest("PHP CLI is not available")
        for relative in (
            "GhostFTP WEB/tests/json-store-bounds.php",
            "GhostFTP WEB/tests/ftp-listing.php",
            "GhostFTP WEB/tests/transfer-limiter.php",
        ):
            result = subprocess.run(
                [php, str(ROOT / relative)],
                cwd=ROOT,
                text=True,
                encoding="utf-8",
                errors="replace",
                stdout=subprocess.PIPE,
                stderr=subprocess.STDOUT,
                check=False,
            )
            self.assertEqual(result.returncode, 0, f"{relative}:\n{result.stdout}")
            self.assertIn("=PASS", result.stdout, relative)

    def test_web_downloads_are_bounded_during_transfer(self) -> None:
        bounded = (ROOT / "GhostFTP WEB/app/Remote/BoundedDownloadInterface.php").read_text(encoding="utf-8")
        limiter = (ROOT / "GhostFTP WEB/app/Remote/TransferLimiter.php").read_text(encoding="utf-8")
        ftp = (ROOT / "GhostFTP WEB/app/Remote/FtpClient.php").read_text(encoding="utf-8")
        sftp = (ROOT / "GhostFTP WEB/app/Remote/SftpClient.php").read_text(encoding="utf-8")
        download = (ROOT / "GhostFTP WEB/download.php").read_text(encoding="utf-8")
        preview = (ROOT / "GhostFTP WEB/preview.php").read_text(encoding="utf-8")

        self.assertIn("downloadBounded(string $remotePath, string $localFile, ?int $maxBytes = null): int", bounded)
        self.assertIn("public const UNKNOWN_SIZE_MAX_BYTES = 536870912;", limiter)
        self.assertIn("public static function effectiveLimit", limiter)
        self.assertIn("public static function limitForDestination", limiter)
        self.assertIn("stream_copy_to_stream($input, $output, self::probeLength($maxBytes))", limiter)
        self.assertIn("$copied > $maxBytes", limiter)

        for source in (ftp, sftp):
            self.assertIn("implements RemoteClientInterface, BoundedDownloadInterface", source)
            self.assertIn("private array $listedFileSizes = [];", source)
            self.assertIn("effectiveDownloadLimit", source)
            self.assertIn("unset($this->listedFileSizes[$remote]);", source)
            self.assertIn("TransferLimiter::effectiveLimit", source)
            self.assertIn("TransferLimiter::limitForDestination($localFile, $maxBytes)", source)
            self.assertIn("@ftruncate", source)

        self.assertIn("ftp_nb_fget", ftp)
        self.assertIn("ftp_nb_continue", ftp)
        self.assertIn("TransferLimiter::assertWithinLimit($fp, $maxBytes)", ftp)
        self.assertIn("TransferLimiter::copy($in, $out, $maxBytes)", sftp)

        self.assertIn("$client instanceof BoundedDownloadInterface", download)
        self.assertIn("$requestedLimit = $reportedSize > 0 ? $reportedSize : null;", download)
        self.assertIn("downloadBounded($path, $tmp, $requestedLimit)", download)
        self.assertIn("$client instanceof BoundedDownloadInterface", preview)
        self.assertIn("$maxPreviewBytes = 10485760;", preview)
        self.assertIn("downloadBounded($path, $tmp, $maxPreviewBytes)", preview)

    def test_release_workflow_requires_delayed_remote_readback(self) -> None:
        source = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
        immediate = "assert_release_asset_set immediate"
        delay = "sleep 5"
        main_guard = 'test "$main_sha" = "$GITHUB_SHA" || { echo "main moved before delayed release verification"'
        delayed = "assert_release_asset_set delayed"
        for marker in (
            "assert_release_asset_set()",
            "RELEASE_ASSET_READBACK=PASS ($phase)",
            "gh release view \"$RELEASE_TAG\" --repo \"$repo\" --json assets",
            "gh release download \"$RELEASE_TAG\" --repo \"$repo\" --pattern 'SHA256.txt'",
            "cmp release/SHA256.txt \"$readback_dir/SHA256.txt\"",
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
