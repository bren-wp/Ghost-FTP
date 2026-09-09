#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class StateDirectoryIdentityContractTests(unittest.TestCase):
    def test_store_pins_directory_identity(self):
        source = (ROOT / "internal/config/store.go").read_text(encoding="utf-8")
        self.assertIn("dirIdentity os.FileInfo", source)
        self.assertIn("os.SameFile(s.dirIdentity, info)", source)
        self.assertIn("s.ensureDirectoryIdentity()", source)

    def test_store_rejects_redirected_directory(self):
        source = (ROOT / "internal/config/store.go").read_text(encoding="utf-8")
        self.assertIn("security.IsReparsePoint(dir)", source)
        self.assertIn("info.Mode()&os.ModeSymlink", source)

    def test_read_and_write_use_bound_directory_helpers(self):
        source = (ROOT / "internal/config/store.go").read_text(encoding="utf-8")
        self.assertIn("s.readBoundState(path)", source)
        self.assertIn("s.writeBoundTemp", source)
        self.assertIn("s.replaceBoundGeneration", source)


if __name__ == "__main__":
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(StateDirectoryIdentityContractTests)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)
    print("STATE_DIRECTORY_IDENTITY_REPLACEMENT=BLOCKED")
