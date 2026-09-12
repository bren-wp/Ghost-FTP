#!/usr/bin/env python3
from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(relative: str) -> str:
    path = ROOT / relative
    if not path.is_file():
        raise AssertionError(f"missing required macOS engine bridge file: {relative}")
    return path.read_text(encoding="utf-8")


class MacOSEngineBridgeContractTests(unittest.TestCase):
    def test_bridge_is_typed_in_process_and_uses_shared_engine(self) -> None:
        bridge = read("macos/Bridge/main.go")
        for marker in (
            '"github.com/bren-wp/Ghost-FTP/internal/api"',
            '"github.com/bren-wp/Ghost-FTP/internal/model"',
            "//export GhostFTPCreateEngine",
            "//export GhostFTPConnect",
            "//export GhostFTPDisconnect",
            "//export GhostFTPCancelPendingTrust",
            "//export GhostFTPIsConnected",
            "//export GhostFTPLastError",
            "//export GhostFTPPendingFingerprint",
            "//export GhostFTPShutdown",
            "//export GhostFTPFreeCString",
            "api.DataDir()",
            "api.New(",
            "engine.Connect(",
            "engine.Disconnect(",
            "engine.CancelPendingTrust()",
            "engine.ActiveConnection()",
        ):
            self.assertIn(marker, bridge)
        self.assertNotIn("encoding/json", bridge)
        self.assertNotIn("net/http", bridge)
        self.assertNotIn("http.ListenAndServe", bridge)

    def test_bridge_does_not_persist_plaintext_secrets(self) -> None:
        bridge = read("macos/Bridge/main.go")
        self.assertIn("model.ConnectionConfig{", bridge)
        self.assertIn("Password:", bridge)
        self.assertIn("Passphrase:", bridge)
        self.assertNotIn("WriteFile", bridge)
        self.assertNotIn("os.Setenv", bridge)
        self.assertNotIn("password =", bridge.lower())
        self.assertNotIn("passphrase =", bridge.lower())

    def test_darwin_platform_boundary_is_explicit_and_private(self) -> None:
        darwin = read("internal/platform/darwin.go")
        for marker in (
            "//go:build darwin",
            'filepath.Join(home, "Library", "Application Support")',
            "syscall.Umask(0o077)",
            "func LocalAppData()",
            "func HardenProcessPrivacy()",
            "func ChoosePrivateKey()",
            "func ChooseDirectory()",
        ):
            self.assertIn(marker, darwin)
        self.assertNotIn("XDG_DATA_HOME", darwin)

    def test_persistent_macos_profiles_fail_closed_until_keychain(self) -> None:
        profile_crypto = read("internal/config/profile_crypto_darwin.go")
        for marker in (
            "//go:build darwin",
            "saved profiles are unavailable until macOS Keychain protection is enabled",
            "func protectProfileData([]byte, string) (string, error)",
            "func unprotectProfileData(string, string) ([]byte, error)",
            "return \"\", errDarwinPersistentProfilesUnavailable",
            "return nil, errDarwinPersistentProfilesUnavailable",
        ):
            self.assertIn(marker, profile_crypto)
        self.assertNotIn("base64", profile_crypto)
        self.assertNotIn("WriteFile", profile_crypto)
        self.assertNotIn("ProtectString", profile_crypto)

    def test_macos_build_links_universal_go_engine_dylib(self) -> None:
        build = read("macos/BUILD.sh")
        for marker in (
            "./macos/Bridge",
            "-buildmode=c-shared",
            "GOOS=darwin",
            'GOARCH="$arch"',
            "build_go_arch arm64 arm64",
            "build_go_arch amd64 x86_64",
            "libGhostFTPEngine.dylib",
            "GhostFTPEngine.h",
            "module.modulemap",
            "lipo -create",
            "@rpath/libGhostFTPEngine.dylib",
            "@executable_path/../Frameworks",
            'FRAMEWORKS="$CONTENTS/Frameworks"',
        ):
            self.assertIn(marker, build)

    def test_appkit_quick_connect_is_wired_to_the_bridge(self) -> None:
        swift = read("macos/Sources/GhostFTPApp/main.swift")
        for marker in (
            "import GhostFTPEngine",
            "NSSecureTextField",
            "GhostFTPCreateEngine()",
            "GhostFTPConnect(",
            "GhostFTPDisconnect()",
            "GhostFTPCancelPendingTrust()",
            "GhostFTPIsConnected()",
            "GhostFTPPendingFingerprint()",
            "GhostFTPLastError()",
            "GhostFTPFreeCString(",
            "DispatchQueue",
            'passwordField.stringValue = ""',
            'passphraseField.stringValue = ""',
        ):
            self.assertIn(marker, swift)
        self.assertNotIn("URLSession", swift)
        self.assertNotIn("UserDefaults", swift)


if __name__ == "__main__":
    unittest.main()
