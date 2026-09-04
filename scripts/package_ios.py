#!/usr/bin/env python3
"""Validate an unsigned GhostFTP iOS app bundle and create deterministic release archives."""

from __future__ import annotations

import argparse
import hashlib
import os
import plistlib
import re
import stat
import sys
import zipfile
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
MACHO_MAGICS = {
    b"\xcf\xfa\xed\xfe",  # MH_MAGIC_64 little endian
    b"\xfe\xed\xfa\xcf",  # MH_CIGAM_64
    b"\xca\xfe\xba\xbe",  # FAT_MAGIC
    b"\xbe\xba\xfe\xca",  # FAT_CIGAM
}


def fail(message: str) -> None:
    raise SystemExit("IOS_PACKAGE_FAILED: " + message)


def canonical_version() -> str:
    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    return version


def validate_app(app: Path, version: str) -> str:
    if not app.is_dir() or app.name != "GhostFTP.app":
        fail("expected a GhostFTP.app directory")

    for path in app.rglob("*"):
        if path.is_symlink():
            fail(f"app bundle contains a symlink: {path.relative_to(app)}")

    info_path = app / "Info.plist"
    if not info_path.is_file():
        fail("GhostFTP.app is missing Info.plist")
    with info_path.open("rb") as handle:
        info = plistlib.load(handle)

    if info.get("CFBundleIdentifier") != "com.ghostftp.client":
        fail("unexpected CFBundleIdentifier")
    if info.get("CFBundleShortVersionString") != version:
        fail("iOS app version does not match VERSION")
    executable_name = info.get("CFBundleExecutable")
    if executable_name != "GhostFTP":
        fail("unexpected CFBundleExecutable")

    executable = app / executable_name
    if not executable.is_file():
        fail("GhostFTP.app is missing its executable")
    if not os.access(executable, os.X_OK):
        fail("GhostFTP executable is not executable")
    with executable.open("rb") as handle:
        if handle.read(4) not in MACHO_MAGICS:
            fail("GhostFTP executable is not a Mach-O binary")

    return executable_name


def archive_tree(source: Path, destination: Path, archive_root: str) -> None:
    destination.parent.mkdir(parents=True, exist_ok=True)
    destination.unlink(missing_ok=True)
    entries = [source] + sorted(source.rglob("*"), key=lambda item: item.as_posix())
    with zipfile.ZipFile(destination, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        for path in entries:
            relative = path.relative_to(source)
            archive_name = archive_root if relative == Path(".") else f"{archive_root}/{relative.as_posix()}"
            if path.is_dir():
                info = zipfile.ZipInfo(archive_name.rstrip("/") + "/", date_time=(1980, 1, 1, 0, 0, 0))
                info.create_system = 3
                info.external_attr = (stat.S_IFDIR | 0o755) << 16
                archive.writestr(info, b"")
                continue
            mode = 0o755 if os.access(path, os.X_OK) else 0o644
            info = zipfile.ZipInfo(archive_name, date_time=(1980, 1, 1, 0, 0, 0))
            info.create_system = 3
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = (stat.S_IFREG | mode) << 16
            archive.writestr(info, path.read_bytes())


def validate_archive(path: Path, required: set[str]) -> None:
    if not path.is_file() or path.stat().st_size == 0:
        fail(f"archive was not created: {path.name}")
    with zipfile.ZipFile(path) as archive:
        names = set(archive.namelist())
        missing = required - names
        if missing:
            fail(f"{path.name} is missing: {', '.join(sorted(missing))}")
        for name in names:
            if name.startswith("/") or "../" in name or name.endswith("/.."):
                fail(f"{path.name} contains an unsafe path: {name}")


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--app", required=True, type=Path)
    parser.add_argument("--output-dir", required=True, type=Path)
    args = parser.parse_args()

    version = canonical_version()
    executable_name = validate_app(args.app, version)
    output_dir = args.output_dir
    output_dir.mkdir(parents=True, exist_ok=True)

    ipa = output_dir / f"GhostFTP-{version}-iOS-arm64-unsigned.ipa"
    app_zip = output_dir / f"GhostFTP-{version}-iOS-arm64-unsigned-app.zip"
    archive_tree(args.app, ipa, "Payload/GhostFTP.app")
    archive_tree(args.app, app_zip, "GhostFTP.app")

    validate_archive(
        ipa,
        {
            "Payload/GhostFTP.app/Info.plist",
            f"Payload/GhostFTP.app/{executable_name}",
        },
    )
    validate_archive(
        app_zip,
        {
            "GhostFTP.app/Info.plist",
            f"GhostFTP.app/{executable_name}",
        },
    )

    print(f"IOS_PACKAGE=PASS ({version})")
    print(f"IPA={ipa.name} SHA256={sha256(ipa)}")
    print(f"APP_ZIP={app_zip.name} SHA256={sha256(app_zip)}")
    print("IOS_SIGNING=UNSIGNED_EXTERNAL_APPLE_IDENTITY_REQUIRED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
