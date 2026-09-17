#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


class WindowsArm64UniversalContractTests(unittest.TestCase):
    def test_native_architecture_detection_includes_arm64(self) -> None:
        source = read("internal/platform/windows_arch.go")
        tests = read("internal/platform/windows_arch_test.go")
        bootstrap = read("cmd/windowsbootstrap/main.go")
        bootstrap_tests = read("cmd/windowsbootstrap/main_test.go")
        self.assertIn("windowsProcessorArchitectureARM64 = 12", source)
        self.assertIn('return "arm64", nil', source)
        self.assertIn('{windowsProcessorArchitectureARM64, "arm64"}', tests)
        self.assertIn('case "x86", "x64", "arm64":', bootstrap)
        self.assertIn('"arm64": "payload/arm64/GhostFTP.exe"', bootstrap_tests)

    def test_pe_tooling_verifies_native_arm64(self) -> None:
        resources = read("scripts/pe_resources.py")
        verifier = read("scripts/verify_release.py")
        self.assertIn("ARM64 = 0xAA64", resources)
        self.assertIn('processor_architecture not in {"amd64", "x86", "arm64"}', resources)
        self.assertIn("machine == ARM64 and magic == 0x20B", resources)
        self.assertIn("ARM64 = 0xAA64", verifier)
        self.assertIn('"arm64": {"machine": ARM64', verifier)
        self.assertIn('manifest_arch = {"x64": "amd64", "x86": "x86", "arm64": "arm64"}[arch]', verifier)

    def test_native_builder_creates_three_internal_payload_families(self) -> None:
        stage = read("BUILD-WINDOWS-ARCH-STAGE.ps1")
        public = read("BUILD-WINDOWS.ps1")
        for marker in (
            "Build-GhostFTPArchitecture -GoArch 'amd64' -Label 'x64'",
            "Build-GhostFTPArchitecture -GoArch '386' -Label 'x86'",
            "Build-GhostFTPArchitecture -GoArch 'arm64' -Label 'arm64'",
            "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
        ):
            self.assertIn(marker, stage)
        for marker in (
            "$Arm64",
            "arm64\\GhostFTP.exe",
            "WINDOWS_PUBLIC_SETUP=UNIVERSAL_X86_X64_ARM64",
            "WINDOWS_PUBLIC_PORTABLE=UNIVERSAL_X86_X64_ARM64",
            "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
            "WINDOWS_PUBLIC_EXECUTABLES=2",
        ):
            self.assertIn(marker, public)

    def test_public_release_keeps_exactly_two_windows_executables(self) -> None:
        ci = read(".github/workflows/ci.yml")
        release = read(".github/workflows/release.yml")
        self.assertIn("(?:x64|x86|x32|arm64)", ci)
        self.assertIn("(?:x64|x86|x32|arm64)", release)
        self.assertIn("WINDOWS_SETUP=universal-x86-x64-arm64", release)
        self.assertIn("WINDOWS_PORTABLE=universal-x86-x64-arm64", release)
        self.assertIn("WINDOWS_BOOTSTRAP_PE=x86", release)
        self.assertIn("WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64", release)
        self.assertIn("WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci", release)
        self.assertIn("test \"$(find release -maxdepth 1 -type f -name '*.exe' | wc -l | tr -d ' ')\" = '2'", release)
        self.assertNotIn("Ghost-FTP-${VERSION}-Setup-arm64.exe", release)
        self.assertNotIn("Ghost-FTP-${VERSION}-Portable-arm64.exe", release)

    def test_release_docs_state_exact_arm64_metadata(self) -> None:
        markers = (
            "WINDOWS_NATIVE_PAYLOADS=x64,x86,arm64",
            "WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci",
        )
        for rel in (
            "README.md",
            "docs/INSTALLATION.md",
            "docs/GITHUB-RELEASES.md",
            "docs/RELEASE-VERIFICATION.md",
            "docs/ARCHITECTURE.md",
            "docs/PLATFORM-PARITY.md",
            "docs/TESTING.md",
            "docs/SIGNING.md",
            "docs/PACKAGES.md",
            "docs/VERSIONING.md",
        ):
            text = read(rel)
            for marker in markers:
                with self.subTest(document=rel, marker=marker):
                    self.assertIn(marker, text)

    def test_narrative_docs_state_arm64_support_without_overclaim(self) -> None:
        for rel in (
            "docs/SUPPORT.md", "docs/ROADMAP.md", "docs/CONTRIBUTING.md",
            "docs/REFERENCE-UI.md", "docs/SECURITY.md", "scripts/README.md",
        ):
            text = read(rel)
            with self.subTest(document=rel, marker="architectures"):
                self.assertIn("x64, x86 and ARM64", text)
            with self.subTest(document=rel, marker="evidence"):
                self.assertIn("WINDOWS_ARM64_RUNTIME_EVIDENCE=not-native-ci", text)

    def test_007_cross_platform_shape_does_not_change_windows_public_shape(self) -> None:
        for rel in (
            "README.md",
            "docs/README.md",
            "docs/INSTALLATION.md",
            "docs/GITHUB-RELEASES.md",
            "docs/RELEASE-VERIFICATION.md",
        ):
            text = read(rel)
            self.assertIn("13 platform artifacts / 16 public files", text, rel)
            self.assertNotIn("18 platform artifacts / 21 public files", text, rel)
            self.assertIn("0.0.7", text, rel)
        release = read(".github/workflows/release.yml")
        self.assertIn("PUBLIC_PLATFORM_ARTIFACTS=13", release)
        self.assertIn("PUBLIC_RELEASE_FILES=16", release)
        self.assertIn("WINDOWS_PUBLIC_EXECUTABLES=2", read("BUILD-WINDOWS.ps1"))


if __name__ == "__main__":
    result = unittest.main(exit=False)
    if not result.result.wasSuccessful():
        raise SystemExit(1)
    print("WINDOWS_ARM64_UNIVERSAL_CONTRACT=PASS")
