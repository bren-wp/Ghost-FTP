#!/usr/bin/env python3
from pathlib import Path
import subprocess
import sys
import unittest

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "macos" / "prepare_site_manager_sources.py"


class MacOSPythonSourceWarningTests(unittest.TestCase):
    def test_source_preparation_script_compiles_without_syntax_warnings(self) -> None:
        result = subprocess.run(
            [
                sys.executable,
                "-W",
                "error::SyntaxWarning",
                "-m",
                "py_compile",
                str(SCRIPT),
            ],
            cwd=ROOT,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            check=False,
        )
        self.assertEqual(
            result.returncode,
            0,
            f"macOS source-preparation script emitted a SyntaxWarning or failed to compile:\n{result.stderr}",
        )

    def test_swift_interpolation_is_escaped_at_python_source_level(self) -> None:
        source = SCRIPT.read_text(encoding="utf-8")
        self.assertIn('"Remote bookmark opened: \\\\(path)"', source)
        self.assertIn('"Local bookmark opened: \\\\(path)"', source)


if __name__ == "__main__":
    unittest.main()
