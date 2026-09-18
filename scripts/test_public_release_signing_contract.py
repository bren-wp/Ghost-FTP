#!/usr/bin/env python3
"""Regression contract for explicit Windows compatibility publication."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RELEASE_WORKFLOW = ROOT / ".github" / "workflows" / "release.yml"
VERIFY_RELEASE = ROOT / "scripts" / "verify_release.py"


class PublicReleaseSigningContractTest(unittest.TestCase):
    def test_release_supports_signed_or_explicit_unsigned_compatibility(self) -> None:
        workflow = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        self.assertIn("name: Publish Ghost FTP", workflow)
        self.assertIn("Resolve Windows signing identity", workflow)
        self.assertIn("WINDOWS_RELEASE_SIGNING=UNSIGNED_COMPATIBILITY", workflow)
        self.assertIn("'state=unsigned'", workflow)
        self.assertIn("'state=signed'", workflow)
        self.assertIn("Windows signing configuration is incomplete", workflow)
        self.assertIn("GHOSTFTP_ALLOW_UNSIGNED_PUBLIC_RELEASE", workflow)
        self.assertIn("No-key compatibility publication is authorized only for Ghost FTP 0.0.8", workflow)

    def test_release_verifies_declared_signature_state_before_publication(self) -> None:
        workflow = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        self.assertIn("Get-AuthenticodeSignature -FilePath $path", workflow)
        self.assertIn("$env:SIGNING_STATE -eq 'signed'", workflow)
        self.assertIn("$env:SIGNING_STATE -eq 'unsigned'", workflow)
        self.assertIn("$signature.Status -ne 'Valid'", workflow)
        self.assertIn("$signature.Status -ne 'NotSigned'", workflow)
        self.assertIn("WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}", workflow)

    def test_release_verifier_is_strict_unless_compatibility_gate_is_explicit(self) -> None:
        verifier = VERIFY_RELEASE.read_text(encoding="utf-8")

        self.assertIn('PUBLIC_WINDOWS_RELEASE_WORKFLOW = "Publish Ghost FTP"', verifier)
        self.assertIn('PUBLIC_WINDOWS_COMPATIBILITY_ENV = "GHOSTFTP_ALLOW_UNSIGNED_PUBLIC_RELEASE"', verifier)
        self.assertIn("explicit compatibility-release gate is enabled", verifier)
        self.assertIn("inconsistent Authenticode state", verifier)
        self.assertIn("require_public_release_signatures", verifier)


if __name__ == "__main__":
    unittest.main()
