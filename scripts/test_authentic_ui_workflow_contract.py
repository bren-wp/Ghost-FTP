#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]
WORKFLOW = ROOT / ".github" / "workflows" / "ui-screenshots.yml"


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class AuthenticUIWorkflowContractTests(unittest.TestCase):
    def test_release_prep_capture_is_evidence_only(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        trigger = workflow.split("permissions:", 1)[0]
        persist = workflow.split("  persist:", 1)[1]

        self.assertIn("- 'release-prep/**'", trigger)
        self.assertIn("github.event_name == 'pull_request'", persist)
        self.assertNotIn("github.event_name == 'push'", persist)
        self.assertIn("refusing a stale screenshot commit", persist)
        self.assertNotIn("[skip ci]", workflow)

    def test_pull_request_ui_changes_always_request_authentic_capture(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
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
            "'scripts/capture_linux_screenshots.sh'",
            "'scripts/capture_android_screenshots.sh'",
            "'scripts/assemble_ui_evidence.py'",
            "'docs/UI-SCREENSHOT-REQUEST'",
            "'.github/workflows/ui-screenshots.yml'",
        ):
            self.assertIn(path, trigger)

    def test_capture_jobs_checkout_exact_pr_head(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        exact_ref = "github.event_name == 'pull_request' && github.event.pull_request.head.sha || github.sha"
        capture_jobs = workflow.split("  persist:", 1)[0]
        self.assertEqual(capture_jobs.count(exact_ref), 3)

    def test_verified_captures_are_always_published_as_artifacts(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        for artifact in (
            "ghostftp-authentic-ui-windows",
            "ghostftp-authentic-ui-linux",
            "ghostftp-authentic-ui-android",
        ):
            self.assertIn(artifact, workflow)
        self.assertEqual(workflow.count("actions/upload-artifact@"), 3)

        windows = workflow
        linux = read("scripts/capture_linux_screenshots.sh")
        android = read("scripts/capture_android_screenshots.sh")
        assembly = read("scripts/assemble_ui_evidence.py")
        for name in (
            "Ghost-FTP-main-workspace.png",
            "Ghost-FTP-site-manager.png",
            "Ghost-FTP-bookmarks.png",
            "Ghost-FTP-settings.png",
            "Ghost-FTP-about.png",
        ):
            self.assertIn(name, windows)
        for name in (
            "ghost-ftp-linux-main-workspace.png",
            "ghost-ftp-linux-bookmarks.png",
            "ghost-ftp-linux-settings.png",
        ):
            self.assertIn(name, linux)
        self.assertIn("capture 'ghost-ftp-android-files.png'", android)
        self.assertIn("capture 'ghost-ftp-android-navigation.png'", android)
        self.assertIn("Sites Bookmarks Transfers Settings About", android)
        self.assertIn('capture "ghost-ftp-android-${lower}.png"', android)
        for name in (
            "ghost-ftp-android-files.png",
            "ghost-ftp-android-navigation.png",
            "ghost-ftp-android-sites.png",
            "ghost-ftp-android-bookmarks.png",
            "ghost-ftp-android-transfers.png",
            "ghost-ftp-android-settings.png",
            "ghost-ftp-android-about.png",
        ):
            self.assertIn(name, assembly)

    def test_persistence_is_sha_bound_and_records_provenance(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        assembly = read("scripts/assemble_ui_evidence.py")
        persist = workflow.split("  persist:", 1)[1]
        self.assertIn("SOURCE_SHA", persist)
        self.assertIn("remote_sha", persist)
        self.assertIn("UI-SCREENSHOT-PROVENANCE.json", persist)
        self.assertIn("AUTHENTIC_UI_SCREENSHOTS=PERSISTED", persist)
        self.assertIn("AUTHENTIC_UI_EVIDENCE=PERSISTED", persist)
        self.assertIn('"capture_source_sha"', assembly)
        self.assertIn('"sha256"', assembly)
        self.assertIn("MAPPING", assembly)

    def test_android_capture_drives_real_runtime_navigation(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        capture = read("scripts/capture_android_screenshots.sh")
        self.assertIn("bash scripts/capture_android_screenshots.sh", workflow)
        self.assertIn('adb install -r "$APK_PATH"', capture)
        self.assertIn("uiautomator dump", capture)
        self.assertIn("tap_ui 'Open navigation'", capture)
        self.assertIn("Sites Bookmarks Transfers Settings About", capture)
        self.assertIn("adb exec-out screencap -p", capture)

    def test_linux_capture_uses_packaged_native_runtime(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        capture = read("scripts/capture_linux_screenshots.sh")
        self.assertIn("bash linux/BUILD.sh", workflow)
        self.assertIn("bash scripts/capture_linux_screenshots.sh", workflow)
        self.assertIn("Linux-amd64.tar.gz", capture)
        self.assertIn("Xvfb", capture)
        self.assertIn("xdotool", capture)
        self.assertIn("import -window", capture)
        self.assertNotIn('export HOME="${RUNNER_TEMP', capture)
        self.assertIn('export XDG_DATA_HOME="$HOME/.ghostftp-ui-evidence-data"', capture)
        self.assertIn('x11_socket="/tmp/.X11-unix/X${display_number}"', capture)
        self.assertIn("two consecutive native-window", capture)

    def test_linux_bookmarks_and_settings_evidence_is_user_reachable_and_distinct(self) -> None:
        gui = read("internal/desktop/gui_linux.go")
        capture = read("scripts/capture_linux_screenshots.sh")
        self.assertIn("u.renderBookmarksHeaderButton()", gui)
        self.assertIn("u.handleBookmarksHeaderMouse(x, y)", gui)
        self.assertIn("open_distinct_overlay 'Bookmarks'", capture)
        self.assertIn("open_distinct_overlay 'Settings'", capture)
        self.assertIn('! cmp -s "$main_png" "$output"', capture)
        self.assertIn('cmp -s "$bookmarks_png" "$settings_png"', capture)

    def test_workflow_avoids_yaml_sensitive_embedded_heredocs(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        self.assertNotIn("<<'PY'", workflow)
        self.assertNotIn('<<"PY"', workflow)

    def test_bookmarks_manager_is_captured_through_real_runtime_command(self) -> None:
        capture = read("scripts/capture_windows_screenshots.ps1")
        self.assertIn("$bookmarksCommand = 97", capture)
        self.assertIn('TitleContains "Bookmarks"', capture)
        self.assertIn('Ghost-FTP-bookmarks.png', capture)
        self.assertIn("PostMessage($main, $wmCommand", capture)
        self.assertIn("PostMessage($bookmarksWindow, 0x0010", capture)


if __name__ == "__main__":
    unittest.main()
