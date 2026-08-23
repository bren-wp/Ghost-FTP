#!/usr/bin/env python3
"""Validate and stage versioned ByFTP Android APK build artifacts."""

from __future__ import annotations

import argparse
import re
import shutil
import sys
import zipfile
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]
REQUIRED_APK_ENTRIES = ("AndroidManifest.xml", "classes.dex", "resources.arsc")


class PackageError(RuntimeError):
    pass


def read_version(version_file: Path) -> str:
    version = version_file.read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        raise PackageError(f"invalid semantic version: {version!r}")
    return version


def validate_apk(path: Path) -> None:
    if not path.is_file() or path.stat().st_size <= 0:
        raise PackageError(f"missing or empty APK: {path}")
    if not zipfile.is_zipfile(path):
        raise PackageError(f"APK is not a valid ZIP container: {path}")

    with zipfile.ZipFile(path, "r") as archive:
        names = archive.namelist()
        if len(names) != len(set(names)):
            raise PackageError(f"APK contains duplicate ZIP entries: {path}")

        name_set = set(names)
        for required in REQUIRED_APK_ENTRIES:
            if required not in name_set:
                raise PackageError(f"APK is missing required entry {required}: {path}")

        for name in names:
            normalized = name.replace("\\", "/")
            member = PurePosixPath(normalized)
            if member.is_absolute() or ".." in member.parts:
                raise PackageError(f"APK contains unsafe ZIP entry {name!r}: {path}")


def stage_apks(debug_apk: Path, release_apk: Path, output_dir: Path, version: str) -> tuple[Path, Path]:
    validate_apk(debug_apk)
    validate_apk(release_apk)
    output_dir.mkdir(parents=True, exist_ok=True)

    debug_out = output_dir / f"ByFTP-{version}-Android-debug.apk"
    release_out = output_dir / f"ByFTP-{version}-Android-release-unsigned.apk"
    shutil.copy2(debug_apk, debug_out)
    shutil.copy2(release_apk, release_out)

    validate_apk(debug_out)
    validate_apk(release_out)
    return debug_out, release_out


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--debug", type=Path, required=True, help="Path to app-debug.apk")
    parser.add_argument("--release", type=Path, required=True, help="Path to app-release-unsigned.apk")
    parser.add_argument("--output-dir", type=Path, default=ROOT / "dist")
    parser.add_argument("--version-file", type=Path, default=ROOT / "VERSION")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    try:
        version = read_version(args.version_file)
        debug_out, release_out = stage_apks(args.debug, args.release, args.output_dir, version)
    except (OSError, zipfile.BadZipFile, PackageError) as error:
        print(f"ANDROID_PACKAGE_FAILED: {error}", file=sys.stderr)
        return 1

    print(f"ANDROID_PACKAGE_VERSION={version}")
    print(f"ANDROID_DEBUG_APK={debug_out}")
    print(f"ANDROID_RELEASE_UNSIGNED_APK={release_out}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
