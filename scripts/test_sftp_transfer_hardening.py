#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class SFTPTransferHardeningTests(unittest.TestCase):
    def test_rsa_key_type_is_normalized_before_session_config(self):
        manager = (ROOT / "internal/remote/manager.go").read_text(encoding="utf-8")
        policy = (ROOT / "internal/remote/sftp_hostkey_policy.go").read_text(encoding="utf-8")
        self.assertIn("hostKeyConstraintForScannedKey(keyAlgorithm)", manager)
        self.assertIn('scannedKeyType == "ssh-rsa"', policy)
        self.assertNotIn('return "ssh-rsa"', policy)

    def test_unix_curl_path_does_not_prefer_windows_name(self):
        tools = (ROOT / "internal/remote/tools.go").read_text(encoding="utf-8")
        windows_block, unix_block = tools.split('if runtime.GOOS == "windows"', 1)[1].split('if p, err := exec.LookPath("curl")', 1)
        self.assertIn("windowsCurlCandidates(systemDir, runtime.GOARCH)", windows_block)
        self.assertIn('filepath.Join(systemDir, "curl.exe")', tools)
        self.assertIn('"Sysnative", "curl.exe"', tools)
        self.assertNotIn('exec.LookPath("curl.exe")', unix_block)

    def test_engine_validates_file_target_before_queue(self):
        engine = (ROOT / "internal/api/engine.go").read_text(encoding="utf-8")
        add_one = engine.index("func (e *Engine) AddTransfer(")
        add_batch = engine.index("func (e *Engine) AddTransfers(")
        self.assertLess(engine.index("security.ValidateRemoteFilePath(remotePath)", add_one), engine.index("e.transfers.AddBatchOne", add_one))
        self.assertLess(engine.index("security.ValidateRemoteFilePath(r.RemotePath)", add_batch), engine.index("e.transfers.AddBatch(batch)", add_batch))

    def test_remote_revalidation_rejects_unsafe_staging(self):
        source = (ROOT / "internal/remote/remote_commit_revalidation.go").read_text(encoding="utf-8")
        self.assertIn("remoteEntry(items, tempName)", source)
        self.assertIn("staged.IsDirectory || staged.IsSymlink", source)


if __name__ == "__main__":
    unittest.main()
