#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class RemoteCommitRevalidationTests(unittest.TestCase):
    def test_both_upload_adapters_revalidate_before_commit(self) -> None:
        for path, signature in (
            ("internal/remote/curl_ftp.go", "func (c *CurlFTP) Upload"),
            ("internal/remote/sftp.go", "func (s *SFTP) Upload"),
        ):
            text = (ROOT / path).read_text(encoding="utf-8")
            start = text.index(signature)
            block = text[start:]
            revalidate = block.index("revalidateRemoteCommit(")
            commit = block.index("commitRemoteTemp(")
            self.assertLess(revalidate, commit, path)
            self.assertIn("options.SkipExisting", block[revalidate:commit], path)

    def test_revalidation_helper_cleans_temporary_upload(self) -> None:
        text = (ROOT / "internal/remote/remote_commit_revalidation.go").read_text(encoding="utf-8")
        for marker in (
            "cleanupFailure(revalidationErr, dir, tempName, delete)",
            "remoteEntry(items, base)",
            "existing.IsDirectory || existing.IsSymlink",
            "cleanupFailure(ErrSkipped, dir, tempName, delete)",
        ):
            self.assertIn(marker, text)

        cleanup = (ROOT / "internal/remote/remote_cleanup_hardening.go").read_text(encoding="utf-8")
        for marker in (
            "cleanupContext()",
            "delete(cleanupCtx, dir, name, false)",
            "remoteCleanupConfirmsMissing(err)",
            "remoteResidualArtifactError",
        ):
            self.assertIn(marker, cleanup)


if __name__ == "__main__":
    unittest.main()
