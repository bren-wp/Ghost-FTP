#!/usr/bin/env python3
"""Fail-closed audit for Linux transport and AskPass executable provenance."""

from __future__ import annotations

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        raise AssertionError(f"missing required security source: {path}")
    return target.read_text(encoding="utf-8")


def go_function_body(source: str, signature: str) -> str:
    start = source.find(signature)
    if start < 0:
        raise AssertionError(f"missing Go function signature: {signature}")
    brace = source.find("{", start)
    if brace < 0:
        raise AssertionError(f"missing Go function body: {signature}")
    depth = 0
    for pos in range(brace, len(source)):
        char = source[pos]
        if char == "{":
            depth += 1
        elif char == "}":
            depth -= 1
            if depth == 0:
                return source[brace + 1 : pos]
    raise AssertionError(f"unterminated Go function body: {signature}")


class LinuxTransportSecurityContract(unittest.TestCase):
    def test_trusted_transport_and_askpass_provenance_contract(self) -> None:
        shared = read("internal/linuxtrust/trusted_executable_linux.go")
        resolver = read("internal/remote/transport_tools_linux.go")
        tools = read("internal/remote/tools.go")
        sftp = read("internal/remote/sftp.go")
        platform = read("internal/platform/other.go")
        transport_regressions = read("internal/remote/tools_other_test.go")
        platform_regressions = read("internal/platform/linux_hardening_test.go")

        self.assertTrue(shared.startswith("//go:build linux\n"))
        self.assertTrue(resolver.startswith("//go:build linux\n"))
        self.assertTrue(platform.startswith("//go:build linux\n"))

        validation = go_function_body(
            shared,
            "func trustedExecutableDepth(candidate string, depth int) (string, bool)",
        )
        for marker in (
            "TrustedDirectoryChain(filepath.Dir(candidate), depth)",
            "os.Lstat(candidate)",
            "fileUID(info)",
            "uid != 0",
            "os.ModeSymlink",
            "os.Readlink(candidate)",
            "filepath.EvalSymlinks(candidate)",
            "TrustedMetadata(uid, info.Mode(), false)",
            "os.Stat(evaluated)",
        ):
            self.assertIn(marker, validation)

        directory_chain = go_function_body(
            shared,
            "func TrustedDirectoryChain(dir string, depth int) bool",
        )
        for marker in (
            "os.Lstat(current)",
            "uid != 0",
            "os.Readlink(current)",
            "TrustedDirectoryChain(filepath.Clean(target), depth+1)",
            "os.Stat(current)",
            "TrustedMetadata(resolvedUID, resolvedInfo.Mode(), true)",
        ):
            self.assertIn(marker, directory_chain)

        metadata = go_function_body(
            shared,
            "func TrustedMetadata(uid uint32, mode os.FileMode, directory bool) bool",
        )
        for marker in (
            "uid != 0",
            "mode.Perm()&0022 != 0",
            "mode.IsDir()",
            "mode.IsRegular()",
            "mode.Perm()&0111 != 0",
        ):
            self.assertIn(marker, metadata)

        discovery = go_function_body(
            resolver,
            "func findTrustedTransportExecutable(name string) (string, error)",
        )
        for marker in (
            "exec.LookPath(name)",
            "trustedLinuxTransportExecutable(candidate)",
        ):
            self.assertIn(marker, discovery)
        for marker in ('"/usr/bin"', '"/bin"', '"/usr/sbin"', '"/sbin"'):
            self.assertIn(marker, resolver)

        transport_wrapper = go_function_body(
            resolver,
            "func trustedLinuxTransportExecutable(candidate string) (string, bool)",
        )
        self.assertIn("linuxtrust.TrustedExecutable(candidate)", transport_wrapper)

        askpass_parent = go_function_body(
            platform,
            "func trustedLinuxAskPassParentPath(parentExe string) bool",
        )
        for marker in (
            "filepath.IsAbs(parentExe)",
            'name != "ssh" && name != "sftp"',
            "linuxtrust.TrustedExecutable(parentExe)",
        ):
            self.assertIn(marker, askpass_parent)

        trusted_parent = go_function_body(
            platform,
            "func TrustedAskPassParent() bool",
        )
        self.assertIn('os.Readlink("/proc/" + strconv.Itoa(os.Getppid()) + "/exe")', trusted_parent)
        self.assertIn("trustedLinuxAskPassParentPath(parentExe)", trusted_parent)

        find_curl = go_function_body(tools, "func findCurl() (string, error)")
        self.assertIn('findTrustedTransportExecutable("curl")', find_curl)
        self.assertNotIn("exec.LookPath(", tools)

        find_openssh = go_function_body(sftp, "func findOpenSSH(name string) (string, error)")
        self.assertIn("findTrustedTransportExecutable(name)", find_openssh)
        self.assertNotIn("exec.LookPath(", sftp)

        for test_name in (
            "TestFindCurlRejectsPATHShadowing",
            "TestFindCurlFallsBackToTrustedSystemBinary",
            "TestFindOpenSSHRejectsPATHShadowing",
            "TestTrustedLinuxTransportRejectsUserControlledSymlink",
            "TestTrustedLinuxTransportAcceptsSystemSymlinkChain",
            "TestTrustedLinuxDirectoryPermissionBoundary",
            "TestTrustedLinuxTransportRejectsNonRegularFile",
            "TestTrustedLinuxTransportRequiresExecutablePermission",
            "TestTrustedLinuxTransportRequiresRootOwnership",
        ):
            self.assertIn(test_name, transport_regressions)

        for test_name in (
            "TestTrustedLinuxAskPassParentPathAcceptsTrustedSystemOpenSSH",
            "TestTrustedLinuxAskPassParentPathRejectsUserControlledExecutable",
            "TestTrustedLinuxAskPassParentPathRejectsUserControlledSymlink",
            "TestTrustedLinuxAskPassParentPathRejectsUnqualifiedOrWrongExecutable",
        ):
            self.assertIn(test_name, platform_regressions)

        print("LINUX_TRANSPORT_PATH_SHADOWING=BLOCKED")
        print("LINUX_ASKPASS_PARENT_PROVENANCE=ENFORCED")


if __name__ == "__main__":
    unittest.main()
