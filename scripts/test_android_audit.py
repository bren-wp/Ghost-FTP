from __future__ import annotations

import subprocess
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class AndroidAuditTests(unittest.TestCase):
    def test_android_audit_passes_repository_contract(self) -> None:
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "audit_android.py")],
            cwd=ROOT,
            text=True,
            capture_output=True,
            check=False,
        )
        self.assertEqual(0, result.returncode, result.stdout + result.stderr)
        self.assertIn("ANDROID_AUDIT=PASS", result.stdout)
        self.assertIn("ANDROID_SFTP_HOST_KEY_PINNING=REQUIRED", result.stdout)
        self.assertIn("ANDROID_BROAD_STORAGE_PERMISSION=BLOCKED", result.stdout)


if __name__ == "__main__":
    unittest.main()
