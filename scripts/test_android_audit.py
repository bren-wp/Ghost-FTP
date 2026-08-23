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
        for marker in (
            "ANDROID_AUDIT=PASS",
            "ANDROID_SFTP_HOST_KEY_PINNING=REQUIRED",
            "ANDROID_FTPS_PLATFORM_TRUST_AND_ENDPOINT_CHECKING=ENABLED",
            "ANDROID_FTP_LOGIN_ROOT=ENFORCED",
            "ANDROID_BROAD_STORAGE_PERMISSION=BLOCKED",
            "ANDROID_BACKUP_AND_DEVICE_TRANSFER=BLOCKED",
            "ANDROID_ACTIVITY_LIFECYCLE_CLEANUP=ENFORCED",
            "ANDROID_PICKER_PENDING_STATE=CLEARED",
            "ANDROID_PASSWORD_PERSISTENCE=BLOCKED",
            "ANDROID_VERSION_SOURCE=ROOT_VERSION",
        ):
            self.assertIn(marker, result.stdout)


if __name__ == "__main__":
    unittest.main()
