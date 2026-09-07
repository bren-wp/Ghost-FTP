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
            "startsWith(github.ref_name, 'release/ghostftp-v')",
            'version="${GITHUB_REF_NAME#release/ghostftp-v}"',
            'main_sha="$(git rev-parse HEAD)"',
            'test "$GITHUB_SHA" = "$main_sha"',
            'test "$version" = "$source_version"',
            'gh workflow run release.yml --repo "$GITHUB_REPOSITORY" --ref main -f version="$version"',
        ]
        for marker in required:
            self.assertIn(marker, trigger)


if __name__ == "__main__":
    unittest.main()
