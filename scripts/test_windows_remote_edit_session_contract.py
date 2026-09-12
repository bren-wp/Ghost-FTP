#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def function_source(source: str, name: str) -> str:
    marker = f"func (a *app) {name}("
    start = source.find(marker)
    if start < 0:
        return ""
    next_func = source.find("\nfunc (", start + len(marker))
    if next_func < 0:
        return source[start:]
    return source[start:next_func]


class WindowsRemoteEditSessionContractTests(unittest.TestCase):
    def test_app_tracks_remote_edit_session_busy_state(self) -> None:
        source = read("internal/desktop/windows.go")
        self.assertIn("remoteEditBusy", source)

    def test_remote_edit_readiness_rejects_reentry(self) -> None:
        source = read("internal/desktop/remote_edit_windows.go")
        body = function_source(source, "remoteEditSelectionReady")
        self.assertTrue(body)
        self.assertIn("a.remoteEditBusy", body)

    def test_remote_edit_session_has_code_level_begin_and_finish_guards(self) -> None:
        source = read("internal/desktop/remote_edit_windows.go")
        self.assertIn("func (a *app) beginRemoteEditSession() bool", source)
        self.assertIn("func (a *app) finishRemoteEditSession()", source)
        action = function_source(source, "remoteEditAction")
        self.assertTrue(action)
        self.assertIn("beginRemoteEditSession()", action)

    def test_terminal_remote_edit_paths_release_session(self) -> None:
        source = read("internal/desktop/remote_edit_windows.go")
        for name in (
            "remoteEditAction",
            "showRemoteTextEditor",
            "saveRemoteTextEditor",
            "reloadRemoteTextEditor",
        ):
            body = function_source(source, name)
            self.assertTrue(body, name)
            self.assertIn("finishRemoteEditSession()", body, name)

    def test_remote_edit_button_state_is_refreshed_when_session_changes(self) -> None:
        source = read("internal/desktop/remote_edit_windows.go")
        begin = function_source(source, "beginRemoteEditSession")
        finish = function_source(source, "finishRemoteEditSession")
        self.assertIn("a.updateActionControls()", begin)
        self.assertIn("a.updateActionControls()", finish)


if __name__ == "__main__":
    unittest.main()
