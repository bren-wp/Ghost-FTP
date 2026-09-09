#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class WindowsShortcutOwnershipContractTests(unittest.TestCase):
    def setUp(self):
        self.source = (ROOT / "internal" / "platform" / "shortcut_windows.go").read_text(encoding="utf-8")

    def test_created_shortcuts_are_bound_to_digest_markers(self):
        for marker in (
            'desktopShortcutDigestValue   = "DesktopShortcutSHA256"',
            'startMenuShortcutDigestValue = "StartMenuShortcutSHA256"',
            "stableShortcutDigest(linkPath)",
            "SetRegistryString(ghostFTPUninstallKey, digestValue, digest)",
        ):
            self.assertIn(marker, self.source)

    def test_uninstall_requires_matching_ownership_digest(self):
        self.assertIn("GetRegistryString(ghostFTPUninstallKey, digestValue)", self.source)
        self.assertIn("removeShortcutMatchingDigest(path, expected)", self.source)
        self.assertIn("!strings.EqualFold(current, expectedDigest)", self.source)
        self.assertIn("shortcut je promijenjen nakon instalacije i neće biti obrisan", self.source)

    def test_foreign_same_name_shortcut_is_not_overwritten(self):
        self.assertIn("readShellLinkTarget(linkPath)", self.source)
        self.assertIn("!sameShortcutTarget(existingTarget, target)", self.source)
        self.assertIn("nije Ghost FTP shortcut i nije prepisan", self.source)

    def test_shortcut_digest_rejects_redirects(self):
        self.assertIn("before.Mode()&os.ModeSymlink", self.source)
        self.assertIn("shortcutReparsePoint(path)", self.source)
        self.assertIn("os.SameFile(before, opened)", self.source)


if __name__ == "__main__":
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(WindowsShortcutOwnershipContractTests)
    result = unittest.TextTestRunner(verbosity=2).run(suite)
    if not result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_SHORTCUT_OWNERSHIP=PRESERVED")
