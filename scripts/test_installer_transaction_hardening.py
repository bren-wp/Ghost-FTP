#!/usr/bin/env python3
"""Zaključava 1.0.9 installer backup, activation i rollback invarijante."""

from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]


def fail(message: str) -> None:
    raise SystemExit("INSTALLER_TRANSACTION_HARDENING_NIJE_PROSAO: " + message)


def main() -> int:
    transaction = (ROOT / "cmd" / "installer" / "transaction.go").read_text(encoding="utf-8")
    installer = (ROOT / "cmd" / "installer" / "main.go").read_text(encoding="utf-8")

    for marker in (
        "os.SameFile(info, opened)",
        "src.Seek(0, io.SeekStart)",
        "digest != verifyDigest",
        "digestStableInstallerFile(b.target)",
        "!b.activated",
        "verifyInstalledForRollback()",
        "b.installedDigest",
    ):
        if marker not in transaction:
            fail(f"nedostaje installer transaction guard: {marker}")

    if "return fileBackup{target: target}, nil" not in transaction:
        fail("fresh target snapshot više nije eksplicitno zabilježen")
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
    return 0


if __name__ == "__main__":
    sys.exit(main())
