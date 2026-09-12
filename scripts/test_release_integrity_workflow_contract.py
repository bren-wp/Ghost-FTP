from __future__ import annotations

import unittest
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "release-integrity.yml"


class ReleaseIntegrityWorkflowContractTests(unittest.TestCase):
    def test_release_integrity_workflow_is_read_only_and_exact_run_bound(self) -> None:
        text = WORKFLOW.read_text(encoding="utf-8")
        for marker in (
            "name: Verify Ghost FTP Release Integrity",
            "workflow_run:",
            "- Publish Ghost FTP",
            "contents: read",
            "actions: read",
            "github.event.workflow_run.conclusion == 'success'",
            "ref: ${{ github.event.workflow_run.head_sha }}",
            "persist-credentials: false",
            "actions/download-artifact@484a0b528fb4d7bd804637ccb632e47a0e638317",
            "pattern: ghostftp-release-*",
            "github-token: ${{ github.token }}",
            "run-id: ${{ github.event.workflow_run.id }}",
            "digest-mismatch: error",
            "gh api \"repos/$GITHUB_REPOSITORY/releases/tags/$tag\" > release.json",
            "gh api \"repos/$GITHUB_REPOSITORY/commits/$tag\" --jq .sha",
            "verify_release_digest_readback.py",
            "--expected-commit \"$SOURCE_RUN_SHA\"",
            "RELEASE_TAG_COMMIT_READBACK=PASS",
        ):
            self.assertIn(marker, text)

        for forbidden in (
            "contents: write",
            "actions: write",
            "packages: write",
            "id-token: write",
            "pull_request_target:",
            "--clobber",
            "gh release upload",
            "gh release edit",
            "gh release delete",
        ):
            self.assertNotIn(forbidden, text)

    def test_integrity_workflow_has_no_application_runtime_trigger(self) -> None:
        text = WORKFLOW.read_text(encoding="utf-8")
        self.assertNotIn("push:\n", text)
        self.assertNotIn("pull_request:\n", text)
        self.assertNotIn("workflow_dispatch:\n", text)


if __name__ == "__main__":
    unittest.main()
