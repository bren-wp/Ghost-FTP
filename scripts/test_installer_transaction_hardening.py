#!/usr/bin/env python3
"""Zaključava installer backup, directory identity, activation i rollback invarijante."""

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("INSTALLER_TRANSACTION_HARDENING_NIJE_PROSAO: " + message)


def main() -> int:
    transaction = (ROOT / "cmd" / "installer" / "transaction.go").read_text(encoding="utf-8")
    installer = (ROOT / "cmd" / "installer" / "main.go").read_text(encoding="utf-8")
    directory_guard = (ROOT / "cmd" / "installer" / "install_directory_guard.go").read_text(encoding="utf-8")

    for marker in (
        "sameStableInstallerFile(info, opened)",
        "src.Seek(0, io.SeekStart)",
        "digest != verifyDigest",
        "digestStableInstallerFile(b.target)",
        "!b.activated",
        "verifyInstalledForRollback()",
        "b.installedDigest",
        "directory       *installDirectoryGuard",
        "installerDirectoryGuardForTarget(target)",
        "backupExistingBound(target, guard)",
        "b.verifyDirectory()",
    ):
        if marker not in transaction:
            fail(f"nedostaje installer transaction guard: {marker}")

    for marker in (
        "security.EnsureNoRedirectDirectory(g.root, g.dir)",
        "g.handle.Stat()",
        "os.Lstat(g.dir)",
        "os.SameFile(opened, current)",
    ):
        if marker not in directory_guard:
            fail(f"nedostaje installer directory identity guard: {marker}")

    if "return fileBackup{target: target, directory: guard}, nil" not in transaction:
        fail("fresh target snapshot nije vezan uz identitet instalacijske mape")
    if "if b.target == \"\" || !b.activated" not in transaction:
        fail("rollback ponovno može dirati fresh target prije installer aktivacije")

    fresh = installer.find("if backup.existed()")
    replace = installer.find("platform.ReplaceFile(tmp, path)", fresh)
    no_replace = installer.find("platform.RenameNoReplace(tmp, path)", fresh)
    record = installer.find("backup.recordActivated", fresh)
    if min(fresh, replace, no_replace, record) < 0 or not (fresh < replace < no_replace < record):
        fail("installer activation nije vezana uz existing/fresh no-replace i ownership zapis")

    for call in (
        "installFile(appPath, app, &appBackup)",
        "installFile(unPath, un, &unBackup)",
    ):
        if call not in installer:
            fail(f"produkcijski installer zaobilazi transaction-bound install: {call}")

    print("INSTALLER_TRANSACTION_HARDENING=PROSAO")
    print("INSTALLER_DIRECTORY_REPLACEMENT=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
