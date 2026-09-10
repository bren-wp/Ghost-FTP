#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERIFY_PATH = ROOT / "scripts" / "verify_release.py"

spec = importlib.util.spec_from_file_location("ghostftp_verify_release", VERIFY_PATH)
if spec is None or spec.loader is None:
    raise RuntimeError("unable to load verify_release.py")
verify_release = importlib.util.module_from_spec(spec)
spec.loader.exec_module(verify_release)


class WindowsInstallerArtifactContractTests(unittest.TestCase):
    def test_canonical_public_setup_and_portable_names_are_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for name in (
                "Ghost-FTP-0.0.2-Setup.exe",
                "Ghost-FTP-0.0.2-Portable.exe",
            ):
                (root / name).write_bytes(b"fixture")
            (root / "SHA256.txt").write_text("fixture\n", encoding="ascii")
            (root / "internal").mkdir()

            verify_release.assert_windows_artifact_directory_clean(root)

    def assert_extra_public_executable_rejected(self, name: str) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            (root / "Ghost-FTP-0.0.2-Setup.exe").write_bytes(b"setup")
            (root / "Ghost-FTP-0.0.2-Portable.exe").write_bytes(b"portable")
            (root / name).write_bytes(b"unexpected")
            with self.assertRaisesRegex(ValueError, "unexpected Windows executable artifact"):
                verify_release.assert_windows_artifact_directory_clean(root)

    def test_any_extra_public_executable_is_rejected(self) -> None:
        for name in (
            "Uninstall.exe",
            "Uninstaller.exe",
            "unins000.exe",
            "Ghost-FTP-0.0.2-Uninstaller.exe",
            "helper.exe",
            "Ghost-FTP-0.0.2-Setup-x64.exe",
            "Ghost-FTP-0.0.2-Setup-x86.exe",
            "Ghost-FTP-0.0.2-Setup-x32.exe",
            "Ghost-FTP-0.0.2-Portable-x64.exe",
            "Ghost-FTP-0.0.2-Portable-x86.exe",
        ):
            with self.subTest(name=name):
                self.assert_extra_public_executable_rejected(name)

    def test_internal_arch_staging_names_require_explicit_staging_mode(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for name in (
                "Ghost-FTP-0.0.2-Setup-x64.exe",
                "Ghost-FTP-0.0.2-Portable-x64.exe",
                "Ghost-FTP-0.0.2-Setup-x86.exe",
                "Ghost-FTP-0.0.2-Portable-x86.exe",
            ):
                (root / name).write_bytes(b"fixture")

            with self.assertRaisesRegex(ValueError, "unexpected Windows executable artifact"):
                verify_release.assert_windows_artifact_directory_clean(root)
            verify_release.assert_windows_artifact_directory_clean(root, allow_arch_staging=True)

    def test_noncanonical_public_ghostftp_executable_names_are_rejected(self) -> None:
        for name in (
            "Ghost-FTP-0.0.2-Setup-arm64.exe",
            "Ghost-FTP-0.0.2-Setup-debug.exe",
            "Ghost-FTP-0.0.2-beta-Setup.exe",
            "GhostFTP-0.0.2-Setup.exe",
        ):
            with self.subTest(name=name):
                self.assert_extra_public_executable_rejected(name)

    def test_build_pipeline_keeps_native_builder_and_publishes_two_bootstraps(self) -> None:
        public_build = (ROOT / "BUILD-WINDOWS.ps1").read_text(encoding="utf-8")
        native_build = (ROOT / "BUILD-WINDOWS-ARCH-STAGE.ps1").read_text(encoding="utf-8")
        lower = (public_build + native_build).lower()

        self.assertIn("BUILD-WINDOWS-ARCH-STAGE.ps1", public_build)
        self.assertIn("'./cmd/windowsbootstrap'", public_build)
        self.assertIn("'--arch','universal'", public_build)
        self.assertIn("WINDOWS_PUBLIC_EXECUTABLES=2", public_build)
        self.assertIn("$publicFiles.Count -ne 2", public_build)
        self.assertIn("'./cmd/installer'", native_build)
        self.assertIn("'scripts/make_payload.py'", native_build)
        self.assertIn("'scripts/verify_release.py'", native_build)
        self.assertIn("$publicFiles.Count -ne 4", native_build)

        for marker in ("iscc.exe", "makensis", "candle.exe", "light.exe", "wix build"):
            self.assertNotIn(marker, lower)

    def test_bootstrap_selects_native_payload_without_environment_architecture(self) -> None:
        bootstrap = (ROOT / "cmd/windowsbootstrap/main.go").read_text(encoding="utf-8")
        arch = (ROOT / "internal/platform/windows_arch_windows.go").read_text(encoding="utf-8")
        self.assertIn("platform.NativeWindowsArchitecture()", bootstrap)
        self.assertIn('"payload/" + arch + "/GhostFTP.exe"', bootstrap)
        self.assertIn("os.CreateTemp(localAppData", bootstrap)
        self.assertIn("verifyStaged(path, data)", bootstrap)
        self.assertIn("cmd.Run()", bootstrap)
        self.assertIn('NewProc("GetNativeSystemInfo")', arch)
        self.assertNotIn("PROCESSOR_ARCHITECTURE", bootstrap)
        self.assertNotIn("PROCESSOR_ARCHITEW6432", bootstrap)

    def test_integrated_uninstall_is_owned_by_installed_application(self) -> None:
        constants = (ROOT / "cmd/installer/uninstall_constants.go").read_text(encoding="utf-8")
        registration = (ROOT / "cmd/installer/uninstall_registration_windows.go").read_text(encoding="utf-8")
        runtime = (ROOT / "internal/platform/integrated_uninstall_windows.go").read_text(encoding="utf-8")

        self.assertIn('const installedExecutableDigestValue = "InstalledExecutableSHA256"', constants)
        self.assertIn('fmt.Sprintf("\\\"%s\\\" --uninstall", appPath)', registration)
        self.assertIn("{installedExecutableDigestValue, digest}", registration)
        self.assertIn('strings.TrimSpace(args[1]), "--uninstall"', runtime)
        self.assertIn("ghostFTPInstalledDigestValue", runtime)
        self.assertIn("RemoveVerifiedRegularFileMatchingSHA256", runtime)


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_INSTALLER_ARTIFACT_CONTRACT=PASS")
