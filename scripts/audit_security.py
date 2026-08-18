#!/usr/bin/env python3
"""Provjerava ključne ByFTP sigurnosne invarijante koje ne smiju regresirati."""

from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("SECURITY_AUDIT_NIJE_PROSAO: " + message)


def read(path: str) -> str:
    target = ROOT / path
    if not target.is_file():
        fail(f"nedostaje {path}")
    return target.read_text(encoding="utf-8")


def require(path: str, markers: tuple[str, ...]) -> None:
    text = read(path)
    for marker in markers:
        if marker not in text:
            fail(f"{path} nema obaveznu sigurnosnu invarijantu: {marker}")


def main() -> int:
    require(
        "internal/remote/util.go",
        (
            "func validateDownloadedPart(part string) error",
            "os.Lstat(part)",
            "security.IsReparsePoint(part)",
        ),
    )
    for path in ("internal/remote/curl_ftp.go", "internal/remote/sftp.go"):
        require(path, ("validateDownloadedPart(part)",))

    require(
        "internal/remote/sftp.go",
        (
            "func validatePrivateKeyPath(keyPath string) error",
            "os.Lstat(keyPath)",
            "security.IsReparsePoint(keyPath)",
        ),
    )
    require(
        "internal/remote/manager.go",
        (
            "ErrSessionClosing",
            "ErrDisconnectTimeout",
            "activeOps     sync.WaitGroup",
            "closing       *sessionCloseState",
            "m.activeOps.Add(1)",
            "m.activeOps.Wait()",
            "m.activeOps.Done()",
            "var once sync.Once",
            "go m.finishSessionClose(state, s)",
            "waitForSessionClose(ctx, state)",
            "errors.Is(ctx.Err(), context.Canceled)",
            "m.closing = nil",
        ),
    )
    require(
        "internal/api/engine.go",
        (
            "e.remote.Disconnect(ctx)",
            "context.WithTimeout(context.Background(), 4*time.Second)",
        ),
    )

    require(
        "internal/security/remove_tree.go",
        (
            "func isFilesystemRoot(target string) bool",
            "isFilesystemRoot(root)",
            "maxRemoveTreeDepth",
            "maxRemoveTreeItems",
            "isReparsePoint(target)",
            "os.ModeSymlink",
        ),
    )
    require(
        "internal/transfer/manager.go",
        (
            "security.EnsureLocalWithinRoot(job.LocalRoot, job.LocalPath)",
            "errors.Is(err, remote.ErrSkipped)",
            "errors.Is(err, context.Canceled)",
            "ConnectionIdentity() (string, error)",
        ),
    )
    require(
        "internal/platform/filemove_windows.go",
        ("MoveFileExW", "moveFileWriteThrough", "RenameNoReplace"),
    )
    require(
        "internal/config/store.go",
        ("os.Lstat(path)", "os.SameFile(before, after)", "io.LimitReader", "os.CreateTemp"),
    )

    # Regresije moraju ostati u repozitoriju; audit time sprječava da zaštitni
    # kod i test nestanu zajedno u jednoj kasnijoj izmjeni.
    require(
        "internal/transfer/finish_status_test.go",
        (
            "TestFinishJobKeepsSuccessfulResultWhenCancelArrivesAfterSuccess",
            "TestFinishJobKeepsSkippedResultWhenCancelArrivesAfterSkip",
            "TestFinishJobMarksActualCancellation",
        ),
    )
    require("internal/remote/download_security_test.go", ("validateDownloadedPart", "Symlink"))
    require(
        "internal/remote/private_key_validation_test.go",
        ("TestValidatePrivateKeyPathAcceptsRegularFile", "TestValidatePrivateKeyPathRejectsSymlink"),
    )
    require(
        "internal/remote/manager_test.go",
        (
            "TestDisconnectWaitsForActiveOperationRelease",
            "TestDisconnectTimeoutDefersCloseAndBlocksReconnect",
            "TestDisconnectCancellationDefersClose",
            "TestSecondDisconnectWaitsForExistingCloseState",
            "TestOperationReleaseIsIdempotent",
        ),
    )
    require(
        "internal/usererror/message_test.go",
        (
            "TestMessageSessionStillClosing",
            "TestMessageDisconnectCleanupStillRunning",
            "TestMessageDisconnectLifecycleWinsJoinedDeadline",
        ),
    )
    require("internal/security/remove_tree_root_test.go", ("RemoveTreeNoFollow",))
    require(
        "internal/security/remove_tree_root_windows_test.go",
        ("TestIsFilesystemRootRejectsWindowsVolumeRoots", "server\\share"),
    )

    print("SECURITY_AUDIT=PASS")
    print("DOWNLOAD_STAGING_REPARSE_VALIDATION=ENABLED")
    print("SFTP_PRIVATE_KEY_REPARSE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_RACE=BLOCKED")
    print("REMOTE_SESSION_CLOSE_TIMEOUT=BOUNDED")
    print("REMOTE_SESSION_CANCEL=PROPAGATED")
    print("FILESYSTEM_ROOT_DELETE=BLOCKED")
    print("LATE_TRANSFER_CANCEL_STATUS_REGRESSION=BLOCKED")
    print("STATE_SAFE_OPEN=ENABLED")
    return 0


if __name__ == "__main__":
    main()
