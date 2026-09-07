#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WindowsUninstallContractTests(unittest.TestCase):
    def test_interactive_uninstaller_is_not_advertised_as_quiet(self):
        registration = (ROOT / "cmd/installer/uninstall_registration_windows.go").read_text(encoding="utf-8")
        runtime = (ROOT / "internal/platform/integrated_uninstall_windows.go").read_text(encoding="utf-8")

        self.assertIn('fmt.Sprintf("\\\"%s\\\" --uninstall", appPath)', registration)
        self.assertNotIn('{"QuietUninstallString", quoted}', registration)
        self.assertIn('DeleteRegistryValue(uninstallKey, "QuietUninstallString")', registration)

        # The current integrated uninstall command is deliberately interactive,
        # so a Windows QuietUninstallString would be a false OS integration contract.
        self.assertIn('strings.TrimSpace(args[1]), "--uninstall"', runtime)
        self.assertIn("ConfirmDialog(", runtime)
        self.assertIn("InfoDialog(", runtime)

    def test_stale_quiet_value_remains_inside_registry_rollback_snapshot(self):
        snapshot = (ROOT / "cmd/installer/registry_snapshot.go").read_text(encoding="utf-8")
        self.assertIn('{uninstallKey, "QuietUninstallString"}', snapshot)


if __name__ == "__main__":
    unittest.main()
