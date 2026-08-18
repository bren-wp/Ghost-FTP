#!/usr/bin/env python3
from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def require(path: str, markers: tuple[str, ...]) -> None:
    text = read(path)
    for marker in markers:
        if marker not in text:
            raise AssertionError(f"{path}: nedostaje sigurnosni marker {marker!r}")


def forbid(path: str, markers: tuple[str, ...]) -> None:
    text = read(path)
    for marker in markers:
        if marker in text:
            raise AssertionError(f"{path}: zabranjeni obrazac {marker!r}")


def run_checks() -> None:
    require("internal/platform/filemove_linux.go", (
        "sysRenameat2",
        "renameNoReplace = 1",
        "syscall.Syscall6",
    ))
    forbid("internal/platform/filemove_linux.go", (
        "os.Lstat(dst)",
        "os.Rename(src, dst)",
        "SYS_RENAMEAT2",
    ))
    for path, number in (
        ("internal/platform/sysnum_linux_amd64.go", "316"),
        ("internal/platform/sysnum_linux_arm64.go", "276"),
        ("internal/platform/sysnum_linux_386.go", "353"),
    ):
        require(path, (f"const sysRenameat2 = {number}",))
    require("internal/platform/filemove_linux_otherarch.go", (
        "os.Link(src, dst)",
        "os.Remove(src)",
    ))
    require("internal/platform/filemove_darwin.go", (
        "os.Link(src, dst)",
        "os.Remove(src)",
    ))
    require("internal/security/remove_tree.go", (
        "readStableDirectory",
        "f.ReadDir(-1)",
        "os.SameFile",
        "lokalna mapa je zamijenjena",
    ))
    forbid("internal/security/remove_tree.go", ("os.ReadDir(target)",))
    require("internal/remote/sftp.go", (
        "maxPrivateKeySize",
        "snapshotPrivateKey",
        "io.LimitReader",
        "os.SameFile",
        ".byftp-private-key-*.tmp",
        "privateKeyCopy",
    ))
    require("internal/remote/manager.go", (
        "EnsureNoRedirectDirectory",
        'runtime.GOOS == "windows"',
        "cleanupStaleSFTPArtifacts(knownHostsDir)",
    ))
    forbid("internal/remote/manager.go", (
        'cleanupStaleSFTPArtifacts(filepath.Join(dataDir, "known_hosts"))',
    ))


class FilesystemHardeningTests(unittest.TestCase):
    def test_source_invariants(self) -> None:
        run_checks()


if __name__ == "__main__":
    unittest.main()
