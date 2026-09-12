#!/usr/bin/env python3
"""Regression tests for least-privilege release workflow permissions."""

from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RELEASE_WORKFLOW = ROOT / ".github/workflows/release.yml"


class ReleasePermissionsTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.workflow = RELEASE_WORKFLOW.read_text(encoding="utf-8")
        marker = "\n  publish:\n"
        if marker not in cls.workflow:
            raise AssertionError("release workflow is missing the publish job")
        cls.pre_publish, cls.publish = cls.workflow.split(marker, 1)

    def test_workflow_default_is_read_only(self) -> None:
        self.assertRegex(
            self.workflow,
            re.compile(r"(?m)^permissions:\n  contents: read\n(?:\n|concurrency:)")
        )
        self.assertNotIn("packages: write", self.pre_publish)
        self.assertNotIn("contents: write", self.pre_publish)

    def test_publish_job_has_only_required_write_scopes(self) -> None:
        required = (
            "    permissions:\n"
            "      contents: write\n"
            "      packages: write\n"
        )
        self.assertIn(required, self.publish)
        self.assertEqual(self.workflow.count("contents: write"), 1)
        self.assertEqual(self.workflow.count("packages: write"), 1)


if __name__ == "__main__":
    unittest.main()
