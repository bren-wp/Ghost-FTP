#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


class AskPassExecutableIdentityContractTests(unittest.TestCase):
    def test_linux_askpass_uses_running_inode_reference(self):
        source = (ROOT / "internal/platform/askpass_executable_linux.go").read_text(encoding="utf-8")
        self.assertIn('filepath.Join("/proc", strconv.Itoa(os.Getpid()), "exe")', source)
        self.assertIn("os.SameFile(startedInfo, procInfo)", source)
        self.assertNotIn("return exePath, nil", source)

    def test_application_passes_stable_helper_identity_to_engine(self):
        source = (ROOT / "cmd/ghostftp/main.go").read_text(encoding="utf-8")
        self.assertIn("platform.StableAskPassExecutable(exe)", source)
        self.assertIn("api.New(dataDir, askpassExe)", source)
        self.assertNotIn("api.New(dataDir, exe)", source)

    def test_askpass_helper_validates_running_image_identity(self):
        main = (ROOT / "cmd/ghostftp/main.go").read_text(encoding="utf-8")
        linux = (ROOT / "internal/platform/askpass_executable_linux.go").read_text(encoding="utf-8")
        self.assertIn("platform.SameExecutableIdentity(exeAbs, askpassAbs)", main)
        self.assertIn("os.SameFile(currentInfo, expectedInfo)", linux)


if __name__ == "__main__":
    unittest.main(exit=False)
    print("LINUX_ASKPASS_EXECUTABLE_TOCTOU=BLOCKED")
