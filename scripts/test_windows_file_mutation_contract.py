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


class WindowsFileMutationContractTests(unittest.TestCase):
    def test_app_tracks_local_and_remote_mutations_independently(self) -> None:
        source = read("internal/desktop/windows.go")
        self.assertIn("localMutationBusy", source)
        self.assertIn("remoteMutationBusy", source)

    def test_mutation_buttons_follow_side_specific_busy_state(self) -> None:
        source = read("internal/desktop/action_state_windows.go")
        self.assertIn("localMutationReady := localReady && !a.localMutationBusy", source)
        self.assertIn("remoteMutationReady := remoteReady && !a.remoteMutationBusy", source)
        self.assertIn("setControlEnabled(a.localMkdir, localMutationReady)", source)
        self.assertIn("setControlEnabled(a.remoteMkdir, remoteMutationReady)", source)

    def test_local_mutations_take_a_code_level_guard(self) -> None:
        source = read("internal/desktop/files_actions_windows.go")
        self.assertIn("func (a *app) beginLocalMutation() bool", source)
        self.assertIn("func (a *app) finishLocalMutation()", source)
        for name in ("localMkdirAction", "localRenameAction", "localDeleteAction"):
            body = function_source(source, name)
            self.assertTrue(body, name)
            self.assertIn("beginLocalMutation()", body, name)

    def test_remote_mutations_guard_before_prompt_and_before_async_start(self) -> None:
        source = read("internal/desktop/files_actions_windows.go")
        self.assertIn("func (a *app) remoteMutationActionReady() bool", source)
        self.assertIn("func (a *app) beginRemoteMutation() bool", source)
        self.assertIn("func (a *app) finishRemoteMutation()", source)
        for name in (
            "remoteMkdirAction",
            "remoteRenameAction",
            "remoteDeleteAction",
            "remoteChmodAction",
        ):
            body = function_source(source, name)
            self.assertTrue(body, name)
            self.assertIn("remoteMutationActionReady()", body, name)
        for name in ("runRemoteMutationWithTimeout", "runRemoteBatchMutationWithTimeout"):
            body = function_source(source, name)
            self.assertTrue(body, name)
            self.assertIn("beginRemoteMutation()", body, name)

    def test_async_mutation_guards_are_released_from_dispatch(self) -> None:
        source = read("internal/desktop/files_actions_windows.go")
        self.assertGreaterEqual(source.count("a.finishLocalMutation()"), 4)
        self.assertEqual(source.count("a.finishRemoteMutation()"), 2)
        self.assertIn("a.updateActionControls()", source)


if __name__ == "__main__":
    unittest.main()
