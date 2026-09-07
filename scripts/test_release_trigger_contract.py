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


if __name__ == "__main__":
    unittest.main()
