#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    return (ROOT / relative).read_text(encoding="utf-8")


def function_body(source: str, signature: str) -> str:
    start = source.index(signature)
    opening = source.index("{", start)
    depth = 0
    for index in range(opening, len(source)):
        char = source[index]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return source[opening + 1 : index]
    raise AssertionError(f"unterminated function: {signature}")


class WindowsNativeDialogOwnerContractTests(unittest.TestCase):
    def test_private_key_picker_uses_validated_active_owner(self) -> None:
        source = read("internal/platform/windows.go")
        body = function_body(source, "func ChoosePrivateKey()")
        self.assertRegex(body, r"Owner:\s+premiumDialogOwner\(\),")

    def test_directory_picker_uses_validated_active_owner(self) -> None:
        source = read("internal/platform/windows.go")
        body = function_body(source, "func ChooseDirectory()")
        self.assertRegex(body, r"Owner:\s+premiumDialogOwner\(\),")

    def test_task_dialog_fallback_uses_validated_active_owner(self) -> None:
        source = read("internal/platform/windows.go")
        body = function_body(source, "func taskDialogCall(")
        self.assertIn("owner := premiumDialogOwner()", body)
        self.assertIn("taskDialog.Call(\n\t\towner, 0,", body)
        self.assertNotIn("taskDialog.Call(\n\t\t0, 0,", body)

    def test_message_box_fallback_uses_validated_active_owner(self) -> None:
        source = read("internal/platform/windows.go")
        body = function_body(source, "func MessageBox(")
        self.assertIn("messageBox.Call(premiumDialogOwner(),", body)
        self.assertNotIn("messageBox.Call(0,", body)

    def test_owner_resolution_is_fail_closed_for_non_window_contexts(self) -> None:
        source = read("internal/platform/dialog_premium_windows.go")
        body = function_body(source, "func premiumDialogOwner()")
        self.assertIn("premiumGetActiveWindow.Call()", body)
        self.assertIn("if owner == 0", body)
        self.assertIn("premiumIsWindow.Call(owner)", body)
        self.assertIn("if valid == 0", body)
        self.assertGreaterEqual(body.count("return 0"), 2)
        self.assertIn("return owner", body)


if __name__ == "__main__":
    unittest.main()
