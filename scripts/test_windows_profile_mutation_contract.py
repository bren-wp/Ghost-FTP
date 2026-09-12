from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[1]
WINDOWS = (ROOT / "internal" / "desktop" / "windows.go").read_text(encoding="utf-8")
ACTIONS = (ROOT / "internal" / "desktop" / "action_state_windows.go").read_text(encoding="utf-8")
PROFILES = (ROOT / "internal" / "desktop" / "connection_profiles_windows.go").read_text(encoding="utf-8")


def function_body(source: str, name: str) -> str:
    match = re.search(rf"func \(a \*app\) {re.escape(name)}\([^)]*\).*?\n\}}", source, re.S)
    if not match:
        raise AssertionError(f"missing function: {name}")
    return match.group(0)


class WindowsProfileMutationContractTests(unittest.TestCase):
    def test_app_tracks_profile_mutation_state(self):
        self.assertIn("profileMutationBusy", WINDOWS)

    def test_profile_actions_are_disabled_while_mutation_is_active(self):
        self.assertIn("!a.profileMutationBusy", ACTIONS)

    def test_save_and_delete_use_mutation_guard(self):
        save = function_body(PROFILES, "saveCurrentProfile")
        delete = function_body(PROFILES, "removeCurrentProfile")
        for body in (save, delete):
            self.assertIn("profileMutationBusy", body)
            self.assertIn("setProfileMutationBusy(true)", body)
            self.assertIn("setProfileMutationBusy(false)", body)

    def test_conflicting_profile_inputs_are_locked(self):
        helper = function_body(PROFILES, "setProfileMutationBusy")
        for control in (
            "a.connect",
            "a.profilesCombo",
            "a.protocol",
            "a.host",
            "a.port",
            "a.user",
            "a.pass",
            "a.keyPath",
            "a.chooseKey",
            "a.passphrase",
        ):
            self.assertIn(control, helper)

    def test_shutdown_waits_for_profile_persistence(self):
        close_case = re.search(
            r"case wmClose:\s*(.*?)\s*case wmDestroy:",
            WINDOWS,
            re.S,
        )
        self.assertIsNotNone(close_case, "missing WM_CLOSE lifecycle")
        self.assertIn("if a.profileMutationBusy", close_case.group(1))
        self.assertLess(
            close_case.group(1).find("if a.profileMutationBusy"),
            close_case.group(1).find("destroyWindow.Call(hwnd)"),
        )


if __name__ == "__main__":
    unittest.main()
