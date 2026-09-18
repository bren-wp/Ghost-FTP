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
            'NSButton(title: "Files"',
            'NSButton(title: "Connections"',
            'NSButton(title: "Transfer Queue"',
            'NSButton(title: "Connection info"',
            "makeMasterNavigationRail()",
            "filesNavigationTapped()",
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
            'grep -F \'NSButton(title: "Files"\' "$GENERATED_SOURCE"',
            'grep -F \'NSButton(title: "Connections"\' "$GENERATED_SOURCE"',
            'grep -F \'makeMasterNavigationRail()\' "$GENERATED_SOURCE"',
            '"$GENERATED_SOURCE" "$GENERATED_SITE_MANAGER_SOURCE"',
        ):
            self.assertIn(marker, build)

    def test_native_source_uses_explicit_dark_gold_product_palette(self) -> None:
        main = (ROOT / "macos" / "Sources" / "GhostFTPApp" / "main.swift").read_text(encoding="utf-8")
        for marker in (
            "0x0B0F17",
            "0x121824",
            "0x161D2A",
            "0xF6C445",
            "0xFFD768",
            "0x2B2515",
            "0x4AD79B",
            "0xFF6878",
        ):
            self.assertIn(marker, main)
        self.assertNotIn("NSColor.controlAccentColor", main)

    def test_master_workspace_embeds_real_transfer_queue(self) -> None:
        main = (ROOT / "macos" / "Sources" / "GhostFTPApp" / "main.swift").read_text(encoding="utf-8")
        preparer = (ROOT / "macos" / "prepare_site_manager_sources.py").read_text(encoding="utf-8")
        build = (ROOT / "macos" / "BUILD.sh").read_text(encoding="utf-8")

        for marker in (
            "private let embeddedTransferTable = NSTableView(frame: .zero)",
            "private func makeEmbeddedTransferQueue() -> NSView",
            'id: "queue_file", title: "File"',
            'id: "queue_direction", title: "Direction"',
            'id: "queue_progress", title: "Progress"',
            'id: "queue_status", title: "Status"',
            'id: "queue_speed", title: "Speed"',
            'id: "queue_eta", title: "ETA"',
            "updateEmbeddedTransferQueue()",
            "embeddedPauseResumeTapped()",
            "if tableView === embeddedTransferTable",
        ):
            self.assertIn(marker, main)

        for marker in (
            "let embeddedQueue = makeEmbeddedTransferQueue()",
            "NSStackView(views: [connectionStack, workspace, embeddedQueue])",
            "embeddedQueue.widthAnchor.constraint(equalTo: mainColumn.widthAnchor)",
            "embeddedQueue.heightAnchor.constraint(greaterThanOrEqualToConstant: 180)",
        ):
            self.assertIn(marker, preparer)

        self.assertIn("grep -F 'makeEmbeddedTransferQueue()'", build)
        self.assertIn("grep -F 'queue_progress'", build)

    def test_macos_settings_present_dark_before_light(self) -> None:
        windows = (ROOT / "macos" / "Sources" / "GhostFTPApp" / "ApplicationWindows.swift").read_text(encoding="utf-8")
        self.assertIn('appearancePopup.addItems(withTitles: ["Dark", "Light"])', windows)
        self.assertIn('GhostFTPSettingsAppearance()) == "dark" ? 0 : 1', windows)
        self.assertIn('appearancePopup.indexOfSelectedItem == 0 ? "dark" : "light"', windows)

    def test_saved_ftp_runtime_secret_uses_darwin_broker_and_session_ownership(self) -> None:
        runtime_other = (ROOT / "internal" / "security" / "runtime_secret_other.go").read_text(encoding="utf-8")
        runtime_darwin = (ROOT / "internal" / "security" / "runtime_secret_darwin.go").read_text(encoding="utf-8")
        curl = (ROOT / "internal" / "remote" / "curl_ftp.go").read_text(encoding="utf-8")
        manager = (ROOT / "internal" / "remote" / "manager.go").read_text(encoding="utf-8")
        ownership_test = (ROOT / "internal" / "remote" / "curl_ftp_secret_ownership_other_test.go").read_text(encoding="utf-8")
        self.assertIn("//go:build !windows && !darwin", runtime_other)
        for marker in ("return ProtectString(value)", "return UnprotectBytes(encoded)", "ForgetProtectedSecret(encoded)"):
            self.assertIn(marker, runtime_darwin)
        for marker in ("ownsPasswordBlob bool", "if c.ownsPasswordBlob", "c.ownsPasswordBlob = false"):
            self.assertIn(marker, curl)
        for marker in ("transferResolvedSecretOwnershipToCurl", "s.ownsPasswordBlob = true", "transferResolvedSecretOwnershipToCurl(&resolved, curlSession)"):
            self.assertIn(marker, manager)
        for marker in (
            "TestCurlFTPClosePreservesBorrowedRuntimeSecret",
            "TestCurlFTPCloseForgetsOwnedRuntimeSecret",
            "UnprotectRuntimeBytes(blob)",
        ):
            self.assertIn(marker, ownership_test)

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
