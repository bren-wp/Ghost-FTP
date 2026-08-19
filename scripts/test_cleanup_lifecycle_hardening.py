#!/usr/bin/env python3
"""Zaključava produkcijske 1.0.8 cleanup i transfer lifecycle invarijante."""

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("CLEANUP_LIFECYCLE_HARDENING_NIJE_PROSAO: " + message)


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def main() -> int:
    curl = read("internal/remote/curl_ftp.go")
    sftp = read("internal/remote/sftp.go")
    revalidation = read("internal/remote/remote_commit_revalidation.go")
    util = read("internal/remote/util.go")
    transfer = read("internal/transfer/manager.go")

    for name, source in (("curl", curl), ("sftp", sftp)):
        if source.count("security.ValidateRemoteFilePath(remotePath)") < 2:
            fail(f"{name} upload/download ne koriste strogu file-path validaciju")
        if source.count("cleanupFailure(err, dir, tempName") < 2:
            fail(f"{name} ne propagira upload/verify cleanup neuspjeh")

    if "cleanupFailure(revalidationErr, dir, tempName, delete)" not in revalidation:
        fail("remote revalidacija skriva cleanup neuspjeh")
    if "committedCleanupFailure(nil, dir, savedName, ops.delete)" not in util:
        fail("post-commit rollback cleanup nije fail-closed")
    if "if isRemoteResidualArtifactError(err)" not in util:
        fail("cleanup nesigurnost nije blokirana iz auto-retry politike")
    if "security.ValidateRemoteFilePath(r.RemotePath)" not in transfer:
        fail("transfer queue ne zahtijeva konkretnu remote datoteku")

    uncertain = transfer.find("remote.HasUncertainRemoteState(err)")
    skipped = transfer.find("errors.Is(err, remote.ErrSkipped)", uncertain)
    cancelled = transfer.find("ctx.Err() != nil", uncertain)
    if uncertain < 0 or skipped < 0 or cancelled < 0 or not (uncertain < skipped < cancelled):
        fail("queue ne daje cleanup nesigurnosti prednost pred skipped/cancelled statusom")

    if "go func() {\n\t\tm.wg.Wait()" in transfer or "m.wg.Wait()" in transfer:
        fail("waitWorkers ponovno stvara zasebni WaitGroup waiter goroutine")
    if "workersIdle" not in transfer or "workerExited" not in transfer:
        fail("nedostaje dijeljeni worker-idle lifecycle signal")

    final_activate = "if err := platform.RenameNoReplace(part, local); err != nil {\n\t\t_ = os.Remove(part)"
    if final_activate not in curl:
        fail("lokalni download part se ne čisti nakon finalne activation greške")

    print("CLEANUP_LIFECYCLE_HARDENING=PROSAO")
    return 0


if __name__ == "__main__":
    sys.exit(main())
