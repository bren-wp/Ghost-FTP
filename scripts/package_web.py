#!/usr/bin/env python3
"""Build and verify the deployable ByFTP WEB shared-hosting release archive."""

from __future__ import annotations

import argparse
import json
import re
import stat
import subprocess
import sys
import zipfile
from pathlib import Path, PurePosixPath

ROOT = Path(__file__).resolve().parents[1]
WEB_ROOT = ROOT / "ByFTP WEB"
SEMVER_RE = re.compile(r"^\d+\.\d+\.\d+$")
FIXED_ZIP_TIME = (2026, 1, 1, 0, 0, 0)


def fail(message: str) -> "NoReturn":
    raise SystemExit("WEB_PACKAGE_FAILED: " + message)


def canonical_version() -> str:
    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not SEMVER_RE.fullmatch(version):
        fail(f"invalid VERSION: {version!r}")
    web_version = (WEB_ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if web_version != version:
        fail(f"ByFTP WEB/VERSION mismatch: {web_version!r} != {version!r}")
    return version


def tracked_web_files() -> list[Path]:
    try:
        proc = subprocess.run(
            ["git", "ls-files", "-z", "--", "ByFTP WEB"],
            cwd=ROOT,
            check=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        fail(f"cannot enumerate tracked WEB files: {exc}")

    files: list[Path] = []
    for raw in proc.stdout.split(b"\0"):
        if not raw:
            continue
        try:
            rel_text = raw.decode("utf-8")
        except UnicodeDecodeError as exc:
            fail(f"tracked WEB path is not UTF-8: {exc}")
        rel = PurePosixPath(rel_text)
        if rel.is_absolute() or ".." in rel.parts or not rel.parts or rel.parts[0] != "ByFTP WEB":
            fail(f"unsafe tracked WEB path: {rel_text!r}")
        path = ROOT.joinpath(*rel.parts)
        if path.is_symlink():
            fail(f"tracked WEB symlink is not allowed in public package: {rel_text}")
        if not path.is_file():
            fail(f"tracked WEB entry is not a regular file: {rel_text}")
        files.append(path)

    if not files:
        fail("no tracked WEB files were found")
    return sorted(files, key=lambda p: p.as_posix().casefold())


def archive_name(path: Path) -> str:
    rel = path.relative_to(WEB_ROOT).as_posix()
    pure = PurePosixPath(rel)
    if pure.is_absolute() or ".." in pure.parts or rel in {"", "."}:
        fail(f"unsafe WEB archive path: {rel!r}")
    return rel


def write_archive(output: Path, files: list[Path]) -> None:
    output.parent.mkdir(parents=True, exist_ok=True)
    output.unlink(missing_ok=True)
    seen: set[str] = set()
    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as zf:
        for path in files:
            arcname = archive_name(path)
            folded = arcname.casefold()
            if folded in seen:
                fail(f"case-insensitive duplicate archive path: {arcname}")
            seen.add(folded)
            data = path.read_bytes()
            info = zipfile.ZipInfo(arcname, date_time=FIXED_ZIP_TIME)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.create_system = 3
            info.external_attr = (stat.S_IFREG | 0o644) << 16
            zf.writestr(info, data, compress_type=zipfile.ZIP_DEFLATED, compresslevel=9)


def verify_archive(output: Path, files: list[Path], version: str) -> None:
    expected = [archive_name(path) for path in files]
    required = {
        ".htaccess",
        "VERSION",
        "composer.json",
        "service-worker.js",
        "manifest.webmanifest",
        "index.php",
        "app/bootstrap.php",
        "storage/.htaccess",
    }
    if not required.issubset(expected):
        missing = sorted(required.difference(expected))
        fail("tracked WEB source is missing required package files: " + ", ".join(missing))

    with zipfile.ZipFile(output, "r") as zf:
        names = zf.namelist()
        if names != expected:
            fail("archive entry set/order does not match tracked WEB source")
        if len({name.casefold() for name in names}) != len(names):
            fail("archive contains duplicate paths")
        for name in names:
            pure = PurePosixPath(name)
            if pure.is_absolute() or ".." in pure.parts or "\\" in name:
                fail(f"archive contains unsafe path: {name!r}")
        if zf.read("VERSION").decode("utf-8").strip() != version:
            fail("archived VERSION does not match canonical VERSION")
        try:
            composer = json.loads(zf.read("composer.json").decode("utf-8"))
        except (UnicodeDecodeError, json.JSONDecodeError) as exc:
            fail(f"archived composer.json is invalid: {exc}")
        if composer.get("version") != version:
            fail("archived composer.json version does not match canonical VERSION")
        service_worker = zf.read("service-worker.js").decode("utf-8")
        if f"byftp-static-v{version}" not in service_worker:
            fail("archived service-worker cache namespace does not match canonical VERSION")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output-dir", default="dist")
    args = parser.parse_args()

    version = canonical_version()
    files = tracked_web_files()
    output_dir = Path(args.output_dir)
    if not output_dir.is_absolute():
        output_dir = ROOT / output_dir
    output = output_dir / f"ByFTP-{version}-WEB-shared-hosting.zip"

    write_archive(output, files)
    verify_archive(output, files, version)
    if output.stat().st_size <= 0:
        fail("WEB package is empty")

    print(f"WEB_PACKAGE=PASS ({version})")
    print(f"WEB_PACKAGE_FILES={len(files)}")
    print(f"WEB_PACKAGE_PATH={output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
