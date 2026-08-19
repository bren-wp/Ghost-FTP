#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class ProcessLifecycleHardeningTests(unittest.TestCase):
    def read(self, rel: str) -> str:
        return (ROOT / rel).read_text(encoding="utf-8")

    def test_unix_commands_use_process_groups(self) -> None:
        text = self.read("internal/remote/process_other.go")
        for marker in (
            "Setpgid: true",
            "cmd.Cancel = func() error",
            "syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)",
            "originalCancel := cmd.Cancel",
        ):
            self.assertIn(marker, text)

    def test_windows_cancel_kills_descendant_tree(self) -> None:
        text = self.read("internal/remote/process_windows.go")
        for marker in (
            'NewProc("CreateToolhelp32Snapshot")',
            'NewProc("Process32FirstW")',
            'NewProc("Process32NextW")',
            'NewProc("TerminateProcess")',
            "cmd.Cancel = func() error",
            "terminateDescendantsFromSnapshot(rootPID, parents)",
            "terminateProcessDescendants(rootPID)",
        ):
            self.assertIn(marker, text)

    def test_every_remote_command_context_is_configured(self) -> None:
        for rel in (
            "internal/remote/curl_ftp.go",
            "internal/remote/curl_capability.go",
            "internal/remote/sftp.go",
        ):
            text = self.read(rel)
            command_count = text.count("exec.CommandContext(")
            configured_count = text.count("configureToolCommand(cmd)")
            self.assertGreater(command_count, 0, rel)
            self.assertEqual(command_count, configured_count, rel)

    def test_functional_regression_uses_real_descendant(self) -> None:
        text = self.read("internal/remote/process_lifecycle_test.go")
        for marker in (
            'processHelperEnv = "BYFTP_PROCESS_HELPER"',
            'child := exec.Command(os.Args[0], "-test.run=TestProcessLifecycleHelper")',
            "configureToolCommand(cmd)",
            "cancel()",
            "descendant survived cancellation",
        ):
            self.assertIn(marker, text)


if __name__ == "__main__":
    unittest.main()
