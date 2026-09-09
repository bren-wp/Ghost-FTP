#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class InstallerDirectoryIdentityContractTests(unittest.TestCase):
    def test_guard_retains_open_directory_identity(self):
        source = (ROOT / "cmd" / "installer" / "install_directory_guard.go").read_text(encoding="utf-8")
        self.assertIn("handle *os.File", source)
        self.assertIn("os.Open(dirAbs)", source)
        self.assertIn("g.handle.Stat()", source)
        self.assertIn("os.SameFile(opened, current)", source)

    def test_guard_revalidates_redirect_chain(self):
        source = (ROOT / "cmd" / "installer" / "install_directory_guard.go").read_text(encoding="utf-8")
        self.assertIn("security.EnsureNoRedirectDirectory(g.root, g.dir)", source)
        self.assertIn("security.IsReparsePoint(g.dir)", source)
        self.assertIn("current.Mode()&os.ModeSymlink", source)

    def test_transaction_binds_backup_activation_and_rollback(self):
        source = (ROOT / "cmd" / "installer" / "transaction.go").read_text(encoding="utf-8")
        self.assertIn("installerDirectoryGuardForTarget(target)", source)
        self.assertIn("backupExistingBound(target, guard)", source)
        self.assertIn("directory:      guard", source)
        self.assertGreaterEqual(source.count("b.verifyDirectory()"), 8)


if __name__ == "__main__":
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(InstallerDirectoryIdentityContractTests)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)
    print("INSTALLER_DIRECTORY_IDENTITY=BLOCKED")
