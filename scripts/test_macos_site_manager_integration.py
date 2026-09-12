#!/usr/bin/env python3
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class MacOSSiteManagerIntegrationTests(unittest.TestCase):
    def test_preparer_wires_site_manager_into_native_app(self) -> None:
        main_source = ROOT / "macos" / "Sources" / "GhostFTPApp" / "main.swift"
        site_source = ROOT / "macos" / "Sources" / "GhostFTPApp" / "SiteManager.swift"
        preparer = ROOT / "macos" / "prepare_site_manager_sources.py"
        with tempfile.TemporaryDirectory() as temporary:
            output_main = Path(temporary) / "main.swift"
            output_site = Path(temporary) / "SiteManager.swift"
            subprocess.run(
                [
                    "python3",
                    str(preparer),
                    str(main_source),
                    str(site_source),
                    str(output_main),
                    str(output_site),
                ],
                cwd=ROOT,
                check=True,
            )
            main_text = output_main.read_text(encoding="utf-8")
            site_text = output_site.read_text(encoding="utf-8")

        for marker in (
            'NSButton(title: "Site Manager"',
            "#selector(siteManagerTapped)",
            "SiteManagerWindowController()",
            "controller.onConnected = { [weak self] protocolName, remoteStart, localStart in",
            "self.refreshConnectionState()",
            "self.refreshRemote(remoteTarget)",
            "siteManagerController?.close()",
        ):
            self.assertIn(marker, main_text)

        for marker in (
            "var onConnected: ((String, String, String) -> Void)?",
            "GhostFTPIsConnected() == 0",
            "self.profiles.first(where: { $0.id == profileID })",
            "self.onConnected?(connected.protocolName, connected.remotePath, connected.localPath)",
        ):
            self.assertIn(marker, site_text)

    def test_build_uses_fail_closed_generated_sources(self) -> None:
        build = (ROOT / "macos" / "BUILD.sh").read_text(encoding="utf-8")
        for marker in (
            "prepare_site_manager_sources.py",
            'GENERATED_SOURCE_DIR="$OUT/generated-swift"',
            'GENERATED_SOURCE="$GENERATED_SOURCE_DIR/main.swift"',
            'GENERATED_SITE_MANAGER_SOURCE="$GENERATED_SOURCE_DIR/SiteManager.swift"',
            'python3 "$PREPARE_SITE_MANAGER_SOURCES"',
            'grep -F \'NSButton(title: "Site Manager"\' "$GENERATED_SOURCE"',
            '"$GENERATED_SOURCE" "$GENERATED_SITE_MANAGER_SOURCE"',
        ):
            self.assertIn(marker, build)

    def test_saved_ftp_runtime_secret_uses_darwin_broker_and_session_ownership(self) -> None:
        runtime_other = (ROOT / "internal" / "security" / "runtime_secret_other.go").read_text(encoding="utf-8")
        runtime_darwin = (ROOT / "internal" / "security" / "runtime_secret_darwin.go").read_text(encoding="utf-8")
        curl = (ROOT / "internal" / "remote" / "curl_ftp.go").read_text(encoding="utf-8")
        manager = (ROOT / "internal" / "remote" / "manager.go").read_text(encoding="utf-8")
        self.assertIn("//go:build !windows && !darwin", runtime_other)
        for marker in ("return ProtectString(value)", "return UnprotectBytes(encoded)", "ForgetProtectedSecret(encoded)"):
            self.assertIn(marker, runtime_darwin)
        for marker in ("ownsPasswordBlob bool", "if c.ownsPasswordBlob", "c.ownsPasswordBlob = false"):
            self.assertIn(marker, curl)
        for marker in ("transferResolvedSecretOwnershipToCurl", "s.ownsPasswordBlob = true", "transferResolvedSecretOwnershipToCurl(&resolved, curlSession)"):
            self.assertIn(marker, manager)

    def test_preparer_rejects_missing_anchor(self) -> None:
        preparer = ROOT / "macos" / "prepare_site_manager_sources.py"
        site_source = ROOT / "macos" / "Sources" / "GhostFTPApp" / "SiteManager.swift"
        with tempfile.TemporaryDirectory() as temporary:
            broken_main = Path(temporary) / "broken-main.swift"
            broken_main.write_text("import AppKit\n", encoding="utf-8")
            output_main = Path(temporary) / "main.swift"
            output_site = Path(temporary) / "SiteManager.swift"
            completed = subprocess.run(
                [
                    "python3",
                    str(preparer),
                    str(broken_main),
                    str(site_source),
                    str(output_main),
                    str(output_site),
                ],
                cwd=ROOT,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )
        self.assertNotEqual(completed.returncode, 0)
        self.assertIn("refusing to build", completed.stderr)


if __name__ == "__main__":
    unittest.main()
