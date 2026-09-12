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
        evidence = workflow.split("  evidence:", 1)[1]

        self.assertIn("- 'release-prep/**'", trigger)
        self.assertIn("SOURCE_SHA:", evidence)
        self.assertIn("github.event_name == 'pull_request'", evidence)
        self.assertIn("ref: ${{ env.SOURCE_SHA }}", evidence)
        self.assertIn("Confirm exact tested SHA", evidence)
        self.assertIn("ghostftp-authentic-ui-verified-bundle", evidence)
        self.assertNotIn("git push", workflow)
        self.assertNotIn("git commit", workflow)
        self.assertNotIn("github-actions[bot]", workflow)
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
        capture_jobs, evidence = workflow.split("  evidence:", 1)
        self.assertEqual(capture_jobs.count(exact_ref), 3)
        self.assertIn(f"SOURCE_SHA: ${{{{ {exact_ref} }}}}", evidence)
        self.assertIn("ref: ${{ env.SOURCE_SHA }}", evidence)

    def test_verified_captures_are_always_published_as_artifacts(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        capture_jobs, evidence = workflow.split("  evidence:", 1)
        for artifact in (
            "ghostftp-authentic-ui-windows",
            "ghostftp-authentic-ui-linux",
            "ghostftp-authentic-ui-android",
        ):
            self.assertIn(artifact, capture_jobs)
        self.assertEqual(capture_jobs.count("actions/upload-artifact@"), 3)
        self.assertEqual(evidence.count("actions/upload-artifact@"), 1)
        self.assertIn("ghostftp-authentic-ui-verified-bundle", evidence)

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

    def test_evidence_is_sha_bound_verified_and_never_mutates_pr_head(self) -> None:
        workflow = read(".github/workflows/ui-screenshots.yml")
        assembly = read("scripts/assemble_ui_evidence.py")
        evidence = workflow.split("  evidence:", 1)[1]
        self.assertIn("SOURCE_SHA", evidence)
        self.assertIn("ref: ${{ env.SOURCE_SHA }}", evidence)
        self.assertIn("AUTHENTIC_UI_SOURCE_SHA=", evidence)
        self.assertIn("scripts/assemble_ui_evidence.py", evidence)
        self.assertIn("ghostftp-authentic-ui-verified-bundle", evidence)
        self.assertIn("permissions:\n  contents: read", workflow)
        for forbidden in (
            "contents: write",
            "git push",
            "git commit",
            "github-actions[bot]",
            "AUTHENTIC_UI_SCREENSHOTS=PERSISTED",
        ):
            self.assertNotIn(forbidden, workflow)
        self.assertIn('"capture_source_sha"', assembly)
        self.assertIn('"sha256"', assembly)
        self.assertIn("verify_manifest", assembly)
        self.assertIn("AUTHENTIC_UI_EVIDENCE=VERIFIED", assembly)
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

    def test_android_capture_teardown_waits_before_bounded_avd_cleanup(self) -> None:
        capture = read("scripts/capture_android_screenshots.sh")
        self.assertIn("cleanup_avd_home()", capture)
        self.assertIn("for attempt in $(seq 1 5); do", capture)
        self.assertIn('if rm -rf "$AVD_HOME"; then', capture)
        self.assertIn('wait "$emulator_pid" 2>/dev/null || true', capture)
        cleanup = capture[capture.index("cleanup() {"):capture.index("trap cleanup EXIT")]
        self.assertLess(cleanup.index('wait "$emulator_pid"'), cleanup.index("cleanup_avd_home"))

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
