#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WindowsUninstallContractTests(unittest.TestCase):
    def setUp(self):
        self.constants = (ROOT / "cmd/installer/uninstall_constants.go").read_text(encoding="utf-8")
        self.registration = (ROOT / "cmd/installer/uninstall_registration_windows.go").read_text(encoding="utf-8")
        self.snapshot = (ROOT / "cmd/installer/registry_snapshot.go").read_text(encoding="utf-8")
        self.runtime = (ROOT / "internal/platform/integrated_uninstall_windows.go").read_text(encoding="utf-8")

    def test_interactive_uninstaller_is_not_advertised_as_quiet(self):
        self.assertIn('fmt.Sprintf("\\\"%s\\\" --uninstall", appPath)', self.registration)
        self.assertNotIn('{"QuietUninstallString", quoted}', self.registration)
        self.assertIn('DeleteRegistryValue(uninstallKey, "QuietUninstallString")', self.registration)

        # The integrated uninstall command is deliberately interactive,
        # so a Windows QuietUninstallString would be a false OS integration contract.
        self.assertIn('strings.TrimSpace(args[1]), "--uninstall"', self.runtime)
        self.assertIn("ConfirmDialog(", self.runtime)
        self.assertIn("InfoDialog(", self.runtime)

    def test_installer_records_and_rolls_back_executable_ownership_digest(self):
        self.assertIn('const installedExecutableDigestValue = "InstalledExecutableSHA256"', self.constants)
        self.assertIn("platform.VerifiedRegularFileSHA256(appPath)", self.registration)
        self.assertIn("{installedExecutableDigestValue, digest}", self.registration)
        self.assertIn("{uninstallKey, installedExecutableDigestValue}", self.snapshot)

    def test_runtime_requires_registry_and_digest_ownership(self):
        self.assertIn('GetRegistryString(ghostFTPAppPathsKey, "")', self.runtime)
        self.assertIn('GetRegistryString(ghostFTPUninstallKey, "InstallLocation")', self.runtime)
        self.assertIn('GetRegistryString(ghostFTPUninstallKey, "UninstallString")', self.runtime)
        self.assertIn("ghostFTPInstalledDigestValue", self.runtime)
        self.assertIn("SameExecutableIdentity(actual, expected)", self.runtime)
        self.assertIn("VerifiedRegularFileSHA256(actual)", self.runtime)
        self.assertIn("installed executable no longer matches its ownership digest", self.runtime)

    def test_final_cleanup_is_verified_helper_and_object_bound(self):
        self.assertIn('const integratedUninstallFinalizeArg = "--uninstall-finalize"', self.runtime)
        self.assertIn("prepareIntegratedUninstallHelper", self.runtime)
        self.assertIn("openVerifiedRegularFile(exe, false)", self.runtime)
        self.assertIn("verifiedOpenFileSHA256(source)", self.runtime)
        self.assertIn("waitForUninstallParent", self.runtime)
        self.assertIn("RemoveVerifiedRegularFileMatchingSHA256(expected, digest)", self.runtime)
        self.assertNotIn("scheduleDeleteOnReboot(exe)", self.runtime)
        self.assertNotIn("moveFileDelayUntilReboot", self.runtime)
        self.assertNotIn('NewProc("MoveFileExW")', self.runtime)

    def test_finalizer_digest_is_not_put_on_command_line(self):
        self.assertIn("helperEnvironment(expectedDigest)", self.runtime)
        self.assertIn("integratedUninstallDigestEnv", self.runtime)
        self.assertIn("strconv.Itoa(os.Getpid())", self.runtime)
        self.assertNotIn("exec.Command(helperPath, integratedUninstallFinalizeArg, strconv.Itoa(os.Getpid()), expectedDigest)", self.runtime)

    def test_stale_values_remain_inside_registry_rollback_snapshot(self):
        self.assertIn('{uninstallKey, "QuietUninstallString"}', self.snapshot)
        self.assertIn("{uninstallKey, installedExecutableDigestValue}", self.snapshot)


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_INTEGRATED_UNINSTALL=OWNERSHIP_BOUND")
