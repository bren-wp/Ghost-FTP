#!/usr/bin/env python3
from __future__ import annotations

import sys
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


def main() -> int:
    require("internal/platform/filemove_linux.go", (
        "SYS_RENAMEAT2",
        "renameNoReplace = 1",
    ))
    forbid("internal/platform/filemove_linux.go", (
        "os.Lstat(dst)",
        "os.Rename(src, dst)",
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
    print("FILESYSTEM_HARDENING_REGRESSION=PROSAO")
    return 0


if __name__ == "__main__":
    try:
        sys.exit(main())
    except AssertionError as exc:
        raise SystemExit(f"FILESYSTEM_HARDENING_REGRESSION_NIJE_PROSAO: {exc}")
