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
    def test_canonical_setup_and_portable_names_are_accepted(self) -> None:
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for name in (
                "Ghost-FTP-0.0.2-Setup-x64.exe",
                "Ghost-FTP-0.0.2-Portable-x64.exe",
                "Ghost-FTP-0.0.2-Setup-x86.exe",
                "Ghost-FTP-0.0.2-Portable-x86.exe",
            ):
                (root / name).write_bytes(b"fixture")
            (root / "SHA256.txt").write_text("fixture\n", encoding="ascii")
            (root / "internal").mkdir()

            verify_release.assert_windows_artifact_directory_clean(root)

    def test_any_extra_public_executable_is_rejected(self) -> None:
        forbidden = (
            "Uninstall.exe",
            "Uninstaller.exe",
            "unins000.exe",
            "Ghost-FTP-0.0.2-Uninstaller-x64.exe",
            "helper.exe",
        )
        for name in forbidden:
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temp:
                root = Path(temp)
                (root / "Ghost-FTP-0.0.2-Setup-x64.exe").write_bytes(b"setup")
                (root / "Ghost-FTP-0.0.2-Portable-x64.exe").write_bytes(b"portable")
                (root / name).write_bytes(b"unexpected")
                with self.assertRaisesRegex(ValueError, "unexpected Windows executable artifact"):
                    verify_release.assert_windows_artifact_directory_clean(root)

    def test_build_pipeline_remains_self_hosted_go_installer(self) -> None:
        build = (ROOT / "BUILD-WINDOWS.ps1").read_text(encoding="utf-8")
        lower = build.lower()

        self.assertIn("'./cmd/installer'", build)
        self.assertIn("'scripts/make_payload.py'", build)
        self.assertIn("'scripts/verify_release.py'", build)
        self.assertIn("$publicFiles.Count -ne 4", build)

        # The production Windows build is intentionally self-contained. A future
        # installer replacement is allowed only after explicitly revisiting the
        # integrated-uninstall and artifact contracts rather than slipping in as
        # an undeclared build dependency.
        for marker in ("iscc.exe", "makensis", "candle.exe", "light.exe", "wix build"):
            self.assertNotIn(marker, lower)

    def test_integrated_uninstall_is_owned_by_installed_application(self) -> None:
        registration = (ROOT / "cmd/installer/uninstall_registration_windows.go").read_text(encoding="utf-8")
        runtime = (ROOT / "internal/platform/integrated_uninstall_windows.go").read_text(encoding="utf-8")

        self.assertIn('fmt.Sprintf("\\\"%s\\\" --uninstall", appPath)', registration)
        self.assertIn('strings.TrimSpace(args[1]), "--uninstall"', runtime)
        self.assertIn("InstalledExecutableSHA256", registration)
        self.assertIn("RemoveVerifiedRegularFileMatchingSHA256", runtime)


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_INSTALLER_ARTIFACT_CONTRACT=PASS")
