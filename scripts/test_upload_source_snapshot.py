#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class UploadSourceSnapshotTests(unittest.TestCase):
    def test_both_adapters_use_snapshot_path_and_verify_before_remote_commit(self) -> None:
        for path, signature in (
            ("internal/remote/curl_ftp.go", "func (c *CurlFTP) Upload"),
            ("internal/remote/sftp.go", "func (s *SFTP) Upload"),
        ):
            text = (ROOT / path).read_text(encoding="utf-8")
            start = text.index(signature)
            block = text[start:]
            prepare = block.index("prepareUploadSource(local)")
            verify = block.index("source.Verify()")
            revalidate = block.index("revalidateRemoteCommit(")
            commit = block.index("commitRemoteTemp(")
            self.assertLess(prepare, verify, path)
            self.assertLess(verify, revalidate, path)
            self.assertLess(revalidate, commit, path)
            self.assertIn("defer source.Close()", block[prepare:verify], path)
            self.assertIn("source.Path()", block[prepare:verify], path)

    def test_child_upload_lines_do_not_use_original_local_path(self) -> None:
        curl = (ROOT / "internal/remote/curl_ftp.go").read_text(encoding="utf-8")
        curl_block = curl[curl.index("func (c *CurlFTP) Upload"):curl.index("func (c *CurlFTP) Download")]
        self.assertIn('"upload-file = " + cfgQuote(source.Path())', curl_block)
        self.assertNotIn('"upload-file = " + cfgQuote(local)', curl_block)

        sftp = (ROOT / "internal/remote/sftp.go").read_text(encoding="utf-8")
        sftp_block = sftp[sftp.index("func (s *SFTP) Upload"):sftp.index("func (s *SFTP) Download")]
        self.assertIn('sftpQuote(source.Path())', sftp_block)
        self.assertNotIn('"put "+sftpQuote(local)', sftp_block)

    def test_snapshot_helper_uses_open_handle_copy_digests_and_cleanup(self) -> None:
        text = (ROOT / "internal/remote/upload_source_snapshot.go").read_text(encoding="utf-8")
        for marker in (
            "os.Open(local)",
            "os.SameFile",
            "io.MultiWriter(out, copyHash)",
            "source.Seek(0, io.SeekStart)",
            "sha256.New()",
            "bytes.Equal(copyHash.Sum(nil), verifyHash.Sum(nil))",
            "bytes.Equal(h.Sum(nil), s.digest[:])",
            "if closeErr := s.Close(); closeErr != nil",
            "nije moguće ukloniti lokalni upload snapshot",
            "security.RemoveTreeNoFollow(s.dir)",
        ):
            self.assertIn(marker, text)
        self.assertNotIn("os.Link(local, snapshotPath)", text)


if __name__ == "__main__":
    unittest.main()
