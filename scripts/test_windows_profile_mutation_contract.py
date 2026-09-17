from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[1]
WINDOWS = (ROOT / "internal" / "desktop" / "windows.go").read_text(encoding="utf-8")
ACTIONS = (ROOT / "internal" / "desktop" / "action_state_windows.go").read_text(encoding="utf-8")
PROFILES = (ROOT / "internal" / "desktop" / "connection_profiles_windows.go").read_text(encoding="utf-8")
CLOSE_POLICY = (ROOT / "internal" / "desktop" / "window_close_policy.go").read_text(encoding="utf-8")
CLOSE_POLICY_TEST = (ROOT / "internal" / "desktop" / "window_close_policy_test.go").read_text(encoding="utf-8")


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
        body = close_case.group(1)
        derive_call = "deriveWindowCloseAction(a.profileMutationBusy, a.connected, a.connectionBusy, a.hasActiveTransfers())"
        self.assertIn(derive_call, body)
        self.assertIn("case windowCloseWaitForProfileMutation:", body)
        self.assertLess(
            body.find("case windowCloseWaitForProfileMutation:"),
            body.find("destroyWindow.Call(hwnd)"),
        )

        # The extracted policy is production code, not a test-only shim. Keep the
        # original fail-closed contract explicit and independently table-tested.
        self.assertIn("if profileMutationBusy", CLOSE_POLICY)
        self.assertIn("return windowCloseWaitForProfileMutation", CLOSE_POLICY)
        self.assertIn('name: "profile mutation defers"', CLOSE_POLICY_TEST)
        self.assertIn('name:                "profile mutation wins over active work"', CLOSE_POLICY_TEST)


if __name__ == "__main__":
    unittest.main()
