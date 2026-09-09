#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "ui-screenshots.yml"


class AuthenticUIWorkflowContractTests(unittest.TestCase):
    def test_release_prep_capture_is_evidence_only(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        persist = workflow.split("- name: Persist authentic screenshots in repository", 1)[1].split(
            "- name: Publish authentic UI screenshots", 1
        )[0]
        self.assertIn("github.event_name == 'push'", persist)
        self.assertIn("!startsWith(github.ref_name, 'release-prep/')", persist)
        self.assertIn("!contains(github.ref_name, '-prep-')", persist)
        self.assertIn("refusing a stale screenshot commit", persist)

    def test_pull_request_ui_changes_always_request_authentic_capture(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        trigger = workflow.split("permissions:", 1)[0]
        self.assertIn("pull_request:", trigger)
        self.assertIn("- main", trigger)
        for path in (
            "'VERSION'",
            "'internal/desktop/**'",
            "'internal/i18n/**'",
            "'internal/platform/**'",
            "'scripts/capture_windows_screenshots.ps1'",
            "'.github/workflows/ui-screenshots.yml'",
        ):
            self.assertIn(path, trigger)

    def test_verified_captures_are_always_published_as_artifact(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        publish = workflow.split("- name: Publish authentic UI screenshots", 1)[1]
        self.assertNotIn("if:", publish.split("with:", 1)[0])
        self.assertIn("ghostftp-authentic-ui-screenshots", publish)
        for name in (
            "Ghost-FTP-main-workspace.png",
            "Ghost-FTP-site-manager.png",
            "Ghost-FTP-settings.png",
            "Ghost-FTP-about.png",
        ):
            self.assertIn(name, workflow)


if __name__ == "__main__":
    unittest.main()
