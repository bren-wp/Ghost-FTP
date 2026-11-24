import base64
import importlib.util
from pathlib import Path
import unittest
import uuid
from unittest.mock import patch
from types import SimpleNamespace

SPEC = importlib.util.spec_from_file_location(
    "reset_suite", Path(__file__).with_name("verify-hard-reset.py"))
suite = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(suite)


class SafetyTests(unittest.TestCase):
    def test_only_confirmed_disposable_domain(self):
        identity = str(uuid.uuid4())
        self.assertEqual(suite.validate_domain("ghostftp-durability-linux", identity, identity),
                         identity)
        with self.assertRaises(ValueError):
            suite.validate_domain("production", identity, identity)
        with self.assertRaises(ValueError):
            suite.validate_domain("ghostftp-durability-linux", identity, str(uuid.uuid4()))

    def test_root_must_be_dedicated_absolute(self):
        for platform, root in [("linux", "/"), ("linux", "/tmp/../data"),
                               ("windows", "C:\\"), ("windows", r"\\server\share\tests")]:
            with self.assertRaises(ValueError):
                suite.guest_path(platform, root, "test")
        self.assertEqual(str(suite.guest_path("linux", "/var/tmp/test", "run")),
                         "/var/tmp/test/ghostftp-vm-run")

    def test_windows_literal_and_linux_quoting(self):
        root = suite.guest_path("windows", r"C:\Test Files\O'Brien", "run")
        command = suite.guest_command("windows", "verify", root,
                                      r"C:\Test Files\test.exe", "backup_removed", None)
        encoded = command.split()[-1]
        script = base64.b64decode(encoded).decode("utf-16-le")
        self.assertIn("O''Brien", script)
        self.assertIn("'vm_recovery_hard_reset_probe'", script)
        root = suite.guest_path("linux", "/tmp/test files", "run")
        command = suite.guest_command("linux", "prepare", root, "/tmp/test bin",
                                      "backup_removed", None)
        self.assertIn("'/tmp/test bin'", command)

    def test_identity_mismatch_never_reaches_destroy(self):
        identity = str(uuid.uuid4())
        args = SimpleNamespace(domain="ghostftp-durability-test", uuid=identity,
                               confirm_reset=identity, ssh_target="test@guest",
                               platform="linux", guest_root="/tmp/fixtures",
                               libvirt_uri="qemu:///system", test_binary="/tmp/test",
                               case="backup_removed", linux_dbus_address=None)
        with patch.object(suite, "run", return_value=str(uuid.uuid4())) as command:
            with self.assertRaisesRegex(RuntimeError, "UUID mismatch"):
                suite.exercise(args, {"run_id": str(uuid.uuid4())})
            self.assertEqual(command.call_count, 1)
            self.assertNotIn("destroy", command.call_args.args[0])


if __name__ == "__main__":
    unittest.main()
