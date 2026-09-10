#!/usr/bin/env python3
from __future__ import annotations

import re
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

RETIRED_PUBLIC_PATTERNS = (
    re.compile(r"Ghost FTP\s+\*\*?1\.\d+\.\d+"),
    re.compile(r"Ghost FTP\s+1\.\d+\.\d+"),
    re.compile(r"Ghost-FTP-1\.\d+\.\d+"),
    re.compile(r"ghostftp-v1\.\d+\.\d+"),
    re.compile(r"ghcr\.io/bren-wp/ghost-ftp:1\.\d+\.\d+"),
    re.compile(r"\bVERSION=1\.\d+\.\d+\b"),
    re.compile(r"\bTAG=ghostftp-v1\.\d+\.\d+\b"),
)


class NoRetiredPublicVersionsTests(unittest.TestCase):
    def test_active_markdown_does_not_advertise_retired_1x_public_versions(self) -> None:
        markdown = sorted(path for path in ROOT.rglob("*.md") if ".git" not in path.parts)
        self.assertTrue(markdown)
        failures: list[str] = []
        for path in markdown:
            text = path.read_text(encoding="utf-8")
            for pattern in RETIRED_PUBLIC_PATTERNS:
                match = pattern.search(text)
                if match:
                    failures.append(
                        f"{path.relative_to(ROOT)}: retired public version marker {match.group(0)!r}"
                    )
        self.assertEqual(failures, [], "\n".join(failures))

    def test_current_public_line_is_version_bound(self) -> None:
        version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
        readme = (ROOT / "README.md").read_text(encoding="utf-8")
        versioning = (ROOT / "docs" / "VERSIONING.md").read_text(encoding="utf-8")
        release_workflow = (ROOT / ".github" / "workflows" / "release.yml").read_text(encoding="utf-8")
        self.assertRegex(version, r"^0\.0\.[1-9]\d*$")
        self.assertIn(f"Current Ghost FTP version: **{version}**", readme)
        self.assertIn("Development status: **Active**", readme)
        self.assertIn("Release channel: **Current**", readme)
        self.assertIn("prerelease=false", readme)
        self.assertIn(f"Current source candidate: **{version}**", versioning)
        self.assertIn(f"VERSION={version}", versioning)
        self.assertIn(f"TAG=ghostftp-v{version}", versioning)
        self.assertIn("CHANNEL=Current", versioning)
        self.assertIn("PRERELEASE=false", versioning)
        self.assertIn("major version `0` does not imply prerelease", versioning)
        self.assertNotIn("--prerelease", release_workflow)
        self.assertIn("LATEST_ONLY_RELEASE_RETENTION", (ROOT / "scripts" / "audit_release.py").read_text(encoding="utf-8"))


if __name__ == "__main__":
    unittest.main()
