from __future__ import annotations

import importlib.util
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERIFY_RELEASE = ROOT / "scripts" / "verify_release.py"

spec = importlib.util.spec_from_file_location("ghostftp_verify_release", VERIFY_RELEASE)
assert spec is not None and spec.loader is not None
verify_release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify_release)


class PublicReleaseSigningPolicyTests(unittest.TestCase):
    def test_public_release_requires_both_artifacts_signed(self) -> None:
        with self.assertRaisesRegex(ValueError, "must be Authenticode signed"):
            verify_release.require_public_release_signatures(
                False,
                False,
                verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
            )
        with self.assertRaisesRegex(ValueError, "must be Authenticode signed"):
            verify_release.require_public_release_signatures(
                True,
                False,
                verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
            )
        with self.assertRaisesRegex(ValueError, "must be Authenticode signed"):
            verify_release.require_public_release_signatures(
                False,
                True,
                verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
            )

    def test_public_release_accepts_both_artifacts_signed(self) -> None:
        verify_release.require_public_release_signatures(
            True,
            True,
            verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
        )

    def test_non_release_builds_remain_signing_optional(self) -> None:
        for workflow in ("Ghost FTP CI", "", "Local build"):
            verify_release.require_public_release_signatures(False, False, workflow)


if __name__ == "__main__":
    unittest.main()
