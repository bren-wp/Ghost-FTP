#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "ui-screenshots.yml"


class AuthenticUIWorkflowContractTests(unittest.TestCase):
    def test_release_prep_capture_is_evidence_only(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        trigger = workflow.split("permissions:", 1)[0]
        persist = workflow.split("  persist:", 1)[1]

        self.assertIn("- 'release-prep/**'", trigger)
        self.assertIn("github.event_name == 'pull_request'", persist)
        self.assertNotIn("github.event_name == 'push'", persist)
        self.assertIn("refusing a stale screenshot commit", persist)
        self.assertNotIn("[skip ci]", workflow)

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
            "'android/**'",
            "'linux/**'",
            "'scripts/capture_windows_screenshots.ps1'",
            "'docs/UI-SCREENSHOT-REQUEST'",
            "'.github/workflows/ui-screenshots.yml'",
        ):
            self.assertIn(path, trigger)

    def test_verified_captures_are_always_published_as_artifacts(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        for artifact in (
            "ghostftp-authentic-ui-windows",
            "ghostftp-authentic-ui-linux",
            "ghostftp-authentic-ui-android",
        ):
            self.assertIn(artifact, workflow)
        self.assertEqual(workflow.count("actions/upload-artifact@"), 3)

        for name in (
            "Ghost-FTP-main-workspace.png",
            "Ghost-FTP-site-manager.png",
            "Ghost-FTP-bookmarks.png",
            "Ghost-FTP-settings.png",
            "Ghost-FTP-about.png",
            "ghost-ftp-linux-main-workspace.png",
            "ghost-ftp-linux-bookmarks.png",
            "ghost-ftp-linux-settings.png",
            "ghost-ftp-android-files.png",
            "ghost-ftp-android-navigation.png",
            "ghost-ftp-android-sites.png",
            "ghost-ftp-android-bookmarks.png",
            "ghost-ftp-android-transfers.png",
            "ghost-ftp-android-settings.png",
            "ghost-ftp-android-about.png",
        ):
            self.assertIn(name, workflow)

    def test_persistence_is_sha_bound_and_records_provenance(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        persist = workflow.split("  persist:", 1)[1]
        self.assertIn("SOURCE_SHA", persist)
        self.assertIn("remote_sha", persist)
        self.assertIn("capture_source_sha", persist)
        self.assertIn("UI-SCREENSHOT-PROVENANCE.json", persist)
        self.assertIn("sha256", persist)
        self.assertIn("AUTHENTIC_UI_SCREENSHOTS=PERSISTED", persist)
        self.assertIn("AUTHENTIC_UI_EVIDENCE=PERSISTED", persist)

    def test_android_capture_drives_real_runtime_navigation(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        android = workflow.split("  android:", 1)[1].split("  persist:", 1)[0]
        self.assertIn("adb install -r android/dist/Ghost-FTP-Android.apk", android)
        self.assertIn("uiautomator dump", android)
        self.assertIn("tap_ui 'Open navigation'", android)
        self.assertIn("Sites Bookmarks Transfers Settings About", android)
        self.assertIn("adb exec-out screencap -p", android)

    def test_linux_capture_uses_packaged_native_runtime(self) -> None:
        workflow = WORKFLOW.read_text(encoding="utf-8")
        linux = workflow.split("  linux:", 1)[1].split("  android:", 1)[0]
        self.assertIn("bash linux/BUILD.sh", linux)
        self.assertIn("Linux-amd64.tar.gz", linux)
        self.assertIn("Xvfb", linux)
        self.assertIn("xdotool", linux)
        self.assertIn("import -window", linux)

    def test_bookmarks_manager_is_captured_through_real_runtime_command(self) -> None:
        capture = (ROOT / "scripts" / "capture_windows_screenshots.ps1").read_text(encoding="utf-8")
        self.assertIn("$bookmarksCommand = 97", capture)
        self.assertIn('TitleContains "Bookmarks"', capture)
        self.assertIn('Ghost-FTP-bookmarks.png', capture)
        self.assertIn("PostMessage($main, $wmCommand", capture)
        self.assertIn("PostMessage($bookmarksWindow, 0x0010", capture)


if __name__ == "__main__":
    unittest.main()
