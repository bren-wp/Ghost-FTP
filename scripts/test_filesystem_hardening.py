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
            raise AssertionError(f"{path}: missing security marker {marker!r}")


def forbid(path: str, markers: tuple[str, ...]) -> None:
    text = read(path)
    for marker in markers:
        if marker in text:
            raise AssertionError(f"{path}: forbidden pattern {marker!r}")


def require_absent(path: str) -> None:
    if (ROOT / path).exists():
        raise AssertionError(f"retired platform path must remain absent: {path}")


def run_checks() -> None:
    # Linux no-replace moves must remain race-safe on the three released
    # package architectures. Other Linux architectures retain the hard-link
    # fallback without silently overwriting an existing destination.
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

    # Windows must use MoveFileExW without a replace-existing flag and request
    # write-through semantics so local activation cannot regress to a
    # check-then-os.Rename race.
    require("internal/platform/filemove_windows.go", (
        'NewProc("MoveFileExW")',
        "const moveFileWriteThrough = 0x8",
        "moveFileNoReplaceW.Call",
        "return os.ErrExist",
    ))
    forbid("internal/platform/filemove_windows.go", (
        "MOVEFILE_REPLACE_EXISTING",
        "os.Rename(src, dst)",
    ))

    # 2.x intentionally supports only Windows and Linux application targets.
    # Regression tests must not silently recreate platform-specific source for
    # retired application targets.
    for retired in ("android", "ios", "macos", "internal/platform/filemove_darwin.go"):
        require_absent(retired)

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
        ".GhostFTP-private-key-*.tmp",
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
