#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


class WindowsModalKeyboardRuntimeContractTests(unittest.TestCase):
    def test_runtime_probe_uses_real_windows_focus_and_message_apis(self) -> None:
        source = read("scripts/test_windows_modal_keyboard_runtime.ps1")

        for marker in (
            'GetGUIThreadInfo',
            'GetWindowThreadProcessId',
            'IsWindowEnabled',
            'IsIconic',
            'PostMessage',
            'SetForegroundWindow',
        ):
            self.assertIn(marker, source)
        self.assertIn("$vkTab = 0x09", source)
        self.assertIn("$vkReturn = 0x0D", source)
        self.assertIn("$vkEscape = 0x1B", source)

    def test_site_manager_and_bookmarks_run_tab_enter_escape_cycles(self) -> None:
        source = read("scripts/test_windows_modal_keyboard_runtime.ps1")

        self.assertIn("$siteManagerCommand = 701", source)
        self.assertIn("$bookmarksCommand = 97", source)
        self.assertIn("Focus-CloseWithTab", source)
        self.assertIn("Close button did not retain keyboard focus", source)
        self.assertIn("Could not post Enter", source)
        self.assertIn("Could not post Escape", source)
        self.assertIn("Verify-ModalKeyboardContract -Main $main -Process $process -Command $siteManagerCommand -Title 'Site Manager'", source)
        self.assertIn("Verify-ModalKeyboardContract -Main $main -Process $process -Command $bookmarksCommand -Title 'Bookmarks'", source)

    def test_owner_restore_regression_is_runtime_verified(self) -> None:
        source = read("scripts/test_windows_modal_keyboard_runtime.ps1")

        self.assertIn("Main window remained enabled while $Title was open.", source)
        self.assertIn("IsWindowEnabled($Main)", source)
        self.assertIn("IsIconic($Main)", source)
        self.assertIn("Main window was not restored and enabled after $Context.", source)
        self.assertIn("WINDOWS_MODAL_KEYBOARD_RUNTIME=PASS", source)

    def test_workflow_builds_verified_production_binary_and_runs_probe(self) -> None:
        workflow = read(".github/workflows/windows-modal-keyboard-runtime.yml")

        self.assertIn("Build verified production Windows packages", workflow)
        self.assertIn(".\\BUILD-WINDOWS.ps1", workflow)
        self.assertIn("dist\\internal\\Ghost-FTP-$version-Portable-x64.exe", workflow)
        self.assertIn(".\\scripts\\test_windows_modal_keyboard_runtime.ps1 -Executable $exe", workflow)
        self.assertIn("ref: ${{ github.event_name == 'pull_request' && github.event.pull_request.head.sha || github.sha }}", workflow)
        self.assertIn("contents: read", workflow)


if __name__ == "__main__":
    unittest.main()
