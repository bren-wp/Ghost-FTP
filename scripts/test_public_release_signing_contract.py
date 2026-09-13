#!/usr/bin/env python3
"""Regression contract for fail-closed Windows signing in official releases."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RELEASE_WORKFLOW = ROOT / ".github" / "workflows" / "release.yml"
VERIFY_RELEASE = ROOT / "scripts" / "verify_release.py"


class PublicReleaseSigningContractTest(unittest.TestCase):
    def test_official_release_requires_protected_authenticode_identity(self) -> None:
        workflow = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        self.assertIn("name: Publish Ghost FTP", workflow)
        self.assertIn("Require protected Authenticode identity", workflow)
        self.assertIn(
            "Official Ghost FTP publication requires GHOSTFTP_SIGNING_PFX_BASE64.",
            workflow,
        )
        self.assertIn(
            "Official Ghost FTP publication requires GHOSTFTP_SIGNING_PASSWORD.",
            workflow,
        )
        self.assertIn('"state=signed"', workflow)
        self.assertNotIn('"state=unsigned"', workflow)
        self.assertNotIn("Publishing current release with explicitly unsigned Windows artifacts", workflow)

    def test_official_release_verifies_signatures_before_publication(self) -> None:
        workflow = RELEASE_WORKFLOW.read_text(encoding="utf-8")

        self.assertIn("Get-AuthenticodeSignature -FilePath $path", workflow)
        self.assertIn("$signature.Status -ne 'Valid'", workflow)
        self.assertNotIn("if ('${{ steps.signing.outputs.state }}' -eq 'signed')", workflow)
        self.assertIn("test \"$WINDOWS_SIGNING_STATE\" = 'signed'", workflow)
        self.assertIn("WINDOWS_AUTHENTICODE=${WINDOWS_SIGNING_STATE}", workflow)

    def test_release_verifier_keeps_public_workflow_fail_closed(self) -> None:
        verifier = VERIFY_RELEASE.read_text(encoding="utf-8")

        self.assertIn('PUBLIC_WINDOWS_RELEASE_WORKFLOW = "Publish Ghost FTP"', verifier)
        self.assertIn("requires trusted Authenticode", verifier)
        self.assertIn("require_public_release_signatures", verifier)


if __name__ == "__main__":
    unittest.main()
