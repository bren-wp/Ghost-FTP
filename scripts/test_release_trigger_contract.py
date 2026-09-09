#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class ReleaseTriggerContractTests(unittest.TestCase):
    def test_publish_workflow_is_dispatch_only(self):
        release = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
        header = release.split("\npermissions:", 1)[0]
        self.assertIn("\n  workflow_dispatch:\n", header)
        self.assertNotIn("\n  push:\n", header)
        self.assertNotIn("\n  create:\n", header)
        self.assertNotIn("\n  pull_request:\n", header)

    def test_existing_release_is_never_clobbered(self):
        release = (ROOT / ".github/workflows/release.yml").read_text(encoding="utf-8")
        self.assertNotIn("gh release upload", release)
        self.assertNotIn("--clobber", release)
        self.assertIn("release already exists; refusing to rewrite published assets", release)

    def test_release_branch_trigger_verifies_exact_main_before_dispatch(self):
        trigger = (ROOT / ".github/workflows/release-branch-trigger.yml").read_text(encoding="utf-8")
        required = [
            "on:\n  create:",
            "github.event.ref_type == 'branch'",
            "startsWith(github.event.ref, 'release/ghostftp-v')",
            "CREATED_REF: ${{ github.event.ref }}",
            "prefix='release/ghostftp-v'",
            'version="${CREATED_REF#${prefix}}"',
            'source_version="$(tr -d \'\\r\\n\' < VERSION)"',
            'test "$version" = "$source_version"',
            'main_sha="$(git rev-parse HEAD)"',
            'test "$GITHUB_SHA" = "$main_sha"',
            "gh workflow run release.yml",
            '--repo "$GITHUB_REPOSITORY"',
            "--ref main",
            '-f version="$version"',
        ]
        for marker in required:
            self.assertIn(marker, trigger)

    def test_release_branch_trigger_waits_for_exact_release_before_retention(self):
        trigger = (ROOT / ".github/workflows/release-branch-trigger.yml").read_text(encoding="utf-8")
        required = [
            "wait_for_new_run()",
            "--json databaseId,headSha",
            '[[ -n "$id" && "$sha" = "$main_sha" ]]',
            'release_run_id="$(wait_for_new_run release.yml "$release_before")"',
            'gh run watch "$release_run_id"',
            '--exit-status',
            'release_conclusion="$(gh run view "$release_run_id"',
            'test "$release_conclusion" = \'success\'',
            "gh workflow run release-retention.yml",
            'retention_run_id="$(wait_for_new_run release-retention.yml "$retention_before")"',
            'gh run watch "$retention_run_id"',
            'test "$retention_conclusion" = \'success\'',
            "RELEASE_RETENTION_CHAIN=PASS",
        ]
        for marker in required:
            self.assertIn(marker, trigger)

        release_watch = trigger.index('gh run watch "$release_run_id"')
        release_success = trigger.index('test "$release_conclusion" = \'success\'')
        retention_dispatch = trigger.index("gh workflow run release-retention.yml")
        retention_watch = trigger.index('gh run watch "$retention_run_id"')
        self.assertLess(release_watch, release_success)
        self.assertLess(release_success, retention_dispatch)
        self.assertLess(retention_dispatch, retention_watch)
        self.assertNotIn("--force", trigger)


if __name__ == "__main__":
    unittest.main()
