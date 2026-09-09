#!/usr/bin/env python3
"""Fail-closed audit for trusted Linux network transport executable resolution."""

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
    def test_trusted_transport_resolution_contract(self) -> None:
        resolver = read("internal/remote/transport_tools_linux.go")
        tools = read("internal/remote/tools.go")
        sftp = read("internal/remote/sftp.go")
        regressions = read("internal/remote/tools_other_test.go")

        self.assertTrue(resolver.startswith("//go:build linux\n"))

        discovery = go_function_body(
            resolver,
            "func findTrustedTransportExecutable(name string) (string, error)",
        )
        for marker in (
            "exec.LookPath(name)",
            "trustedLinuxTransportExecutable(candidate)",
            '"/usr/bin"',
            '"/bin"',
            '"/usr/sbin"',
            '"/sbin"',
        ):
            self.assertIn(marker, resolver if marker.startswith('"/') else discovery)

        validation = go_function_body(
            resolver,
            "func trustedLinuxTransportExecutableDepth(candidate string, depth int) (string, bool)",
        )
        for marker in (
            "trustedLinuxDirectoryChain(filepath.Dir(candidate), depth)",
            "os.Lstat(candidate)",
            "linuxFileUID(info)",
            "uid != 0",
            "os.ModeSymlink",
            "os.Readlink(candidate)",
            "filepath.EvalSymlinks(candidate)",
            "trustedLinuxMetadata(uid, info.Mode(), false)",
            "os.Stat(evaluated)",
        ):
            self.assertIn(marker, validation)

        directory_chain = go_function_body(
            resolver,
            "func trustedLinuxDirectoryChain(dir string, depth int) bool",
        )
        for marker in (
            "os.Lstat(current)",
            "uid != 0",
            "os.Readlink(current)",
            "trustedLinuxDirectoryChain(filepath.Clean(target), depth+1)",
            "os.Stat(current)",
            "trustedLinuxMetadata(resolvedUID, resolvedInfo.Mode(), true)",
        ):
            self.assertIn(marker, directory_chain)

        metadata = go_function_body(
            resolver,
            "func trustedLinuxMetadata(uid uint32, mode os.FileMode, directory bool) bool",
        )
        for marker in (
            "uid != 0",
            "mode.Perm()&0022 != 0",
            "mode.IsDir()",
            "mode.IsRegular()",
            "mode.Perm()&0111 != 0",
        ):
            self.assertIn(marker, metadata)

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
            self.assertIn(test_name, regressions)

        print("LINUX_TRANSPORT_PATH_SHADOWING=BLOCKED")


if __name__ == "__main__":
    unittest.main()
