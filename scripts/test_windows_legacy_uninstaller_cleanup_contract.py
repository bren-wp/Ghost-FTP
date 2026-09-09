#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WindowsLegacyUninstallerCleanupContractTests(unittest.TestCase):
    def test_registry_and_digest_proof_precede_cleanup(self):
        helper = (ROOT / "cmd/installer/legacy_uninstaller.go").read_text(encoding="utf-8")
        main = (ROOT / "cmd/installer/main.go").read_text(encoding="utf-8")
        snapshot = (ROOT / "cmd/installer/registry_snapshot.go").read_text(encoding="utf-8")

        self.assertIn('snapshot.stringValue(uninstallKey, "UninstallString")', helper)
        self.assertIn("sameCanonicalLegacyCommand(registered, legacyPath)", helper)
        self.assertIn("platform.VerifiedRegularFileSHA256(legacyPath)", helper)
        self.assertIn("platform.RemoveVerifiedRegularFileMatchingSHA256(proof.path, proof.digest)", helper)
        self.assertIn("func (s registrySnapshot) stringValue", snapshot)
        self.assertIn("legacyUninstaller := captureLegacyUninstallerProof(dir, registryBackup)", main)
        self.assertIn("cleanupLegacyUninstaller(legacyUninstaller)", main)

    def test_pathname_only_legacy_delete_is_gone(self):
        main = (ROOT / "cmd/installer/main.go").read_text(encoding="utf-8")
        helper = (ROOT / "cmd/installer/legacy_uninstaller.go").read_text(encoding="utf-8")

        self.assertNotIn("os.Remove(legacyPath)", main)
        self.assertNotIn("security.IsReparsePoint(legacyPath)", main)
        self.assertNotIn("os.Remove(proof.path)", helper)
        self.assertIn("previous Installed Apps registration did not prove Ghost FTP ownership", helper)


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_LEGACY_UNINSTALLER_CLEANUP=OWNERSHIP_BOUND")
