from __future__ import annotations

import importlib.util
import os
import unittest
from pathlib import Path
from unittest import mock

ROOT = Path(__file__).resolve().parents[1]
VERIFY_RELEASE = ROOT / "scripts" / "verify_release.py"

spec = importlib.util.spec_from_file_location("ghostftp_verify_release", VERIFY_RELEASE)
assert spec is not None and spec.loader is not None
verify_release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify_release)


class PublicReleaseSigningPolicyTests(unittest.TestCase):
    def test_public_release_rejects_unsigned_without_explicit_gate(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            with self.assertRaisesRegex(ValueError, "explicit compatibility-release gate"):
                verify_release.require_public_release_signatures(
                    False,
                    False,
                    verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
                )

    def test_public_release_accepts_explicit_unsigned_compatibility(self) -> None:
        with mock.patch.dict(
            os.environ,
            {verify_release.PUBLIC_WINDOWS_COMPATIBILITY_ENV: "1"},
            clear=True,
        ):
            verify_release.require_public_release_signatures(
                False,
                False,
                verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
            )

    def test_public_release_never_accepts_mixed_signature_state(self) -> None:
        with mock.patch.dict(
            os.environ,
            {verify_release.PUBLIC_WINDOWS_COMPATIBILITY_ENV: "1"},
            clear=True,
        ):
            with self.assertRaisesRegex(ValueError, "inconsistent Authenticode state"):
                verify_release.require_public_release_signatures(
                    True,
                    False,
                    verify_release.PUBLIC_WINDOWS_RELEASE_WORKFLOW,
                )

    def test_public_release_accepts_both_artifacts_signed(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
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
