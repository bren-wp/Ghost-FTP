#!/usr/bin/env python3
"""Fail-closed provjera sadržaja ByFTP Windows release ZIP-a."""

from __future__ import annotations

import argparse
import hashlib
import re
import sys
import zipfile
from pathlib import Path, PurePosixPath

MAX_ENTRIES = 4096
MAX_UNCOMPRESSED_BYTES = 1 << 30
HASH_LINE_RE = re.compile(r"^([0-9a-fA-F]{64})  (.+)$")
MANIFEST = "BUNDLE-SHA256.txt"


def fail(message: str) -> None:
    raise ValueError("BUNDLE_PROVJERA_NIJE_PROSLA: " + message)


def normalize_member(name: str) -> str:
    if "\x00" in name:
        fail("ZIP sadrži NUL u nazivu stavke")
    normalized = name.replace("\\", "/").rstrip("/")
    if not normalized:
        fail("ZIP sadrži praznu putanju")
    path = PurePosixPath(normalized)
    if path.is_absolute() or any(part in ("", ".", "..") for part in path.parts):
        fail(f"ZIP sadrži nesigurnu putanju: {name!r}")
    if path.parts and ":" in path.parts[0]:
        fail(f"ZIP sadrži apsolutnu/drive putanju: {name!r}")
    return path.as_posix()


def required_members(version: str, arch: str) -> set[str]:
    return {
        f"ByFTP-{version}-Portable-{arch}.exe",
        f"ByFTP-{version}-Setup-{arch}.exe",
        "RELEASE-NOTES.txt",
        "BUILD-METADATA.txt",
        "README.md",
        "CHANGELOG.md",
        "LICENSE",
        "Dokumentacija/PRIVATNOST.md",
        "Dokumentacija/SIGURNOST.md",
        "Dokumentacija/PODRSKA.md",
        "Dokumentacija/TESTIRANJE.md",
        MANIFEST,
    }


def sha256_member(zf: zipfile.ZipFile, info: zipfile.ZipInfo) -> str:
    digest = hashlib.sha256()
    with zf.open(info, "r") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def verify_bundle(zip_path: Path, version: str, arch: str) -> None:
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"neispravna verzija: {version!r}")
    if arch not in {"x64", "x86"}:
        fail(f"neispravna Windows arhitektura: {arch!r}")
    if not zip_path.is_file():
        fail(f"ZIP ne postoji: {zip_path}")

    with zipfile.ZipFile(zip_path, "r") as zf:
        infos = zf.infolist()
        if len(infos) > MAX_ENTRIES:
            fail(f"ZIP ima previše stavki: {len(infos)}")
        total = sum(info.file_size for info in infos)
        if total > MAX_UNCOMPRESSED_BYTES:
            fail(f"ZIP je prevelik nakon raspakiravanja: {total} B")

        files: dict[str, zipfile.ZipInfo] = {}
        for info in infos:
            normalized = normalize_member(info.filename)
            if info.is_dir():
                continue
            if info.flag_bits & 0x1:
                fail(f"ZIP sadrži šifriranu stavku: {normalized}")
            if normalized in files:
                fail(f"ZIP sadrži dupliciranu putanju: {normalized}")
            files[normalized] = info

        missing_required = sorted(required_members(version, arch) - files.keys())
        if missing_required:
            fail("nedostaju obavezne stavke: " + ", ".join(missing_required))
        forbidden = {
            f"ByFTP-{version}-Uninstall-{arch}.exe",
            "verification.txt",
            f"verification-{arch}.txt",
        }
        present_forbidden = sorted(forbidden & files.keys())
        if present_forbidden:
            fail("bundle sadrži interni/neželjeni artefakt: " + ", ".join(present_forbidden))

        manifest_info = files.get(MANIFEST)
        if manifest_info is None:
            fail(f"nedostaje {MANIFEST}")
        try:
            manifest_text = zf.read(manifest_info).decode("ascii")
        except UnicodeDecodeError as exc:
            fail(f"{MANIFEST} nije ASCII")
            raise AssertionError from exc

        expected: dict[str, str] = {}
        for line_number, raw in enumerate(manifest_text.splitlines(), 1):
            if not raw.strip():
                continue
            match = HASH_LINE_RE.fullmatch(raw)
            if not match:
                fail(f"neispravan redak {MANIFEST}:{line_number}")
            digest, member = match.groups()
            member = normalize_member(member)
            if member == MANIFEST:
                fail(f"{MANIFEST} ne smije hashirati samoga sebe")
            if member in expected:
                fail(f"duplicirana stavka u {MANIFEST}: {member}")
            expected[member] = digest.lower()

        actual_payload = set(files) - {MANIFEST}
        if set(expected) != actual_payload:
            missing = sorted(actual_payload - set(expected))
            extra = sorted(set(expected) - actual_payload)
            details = []
            if missing:
                details.append("bez hasha: " + ", ".join(missing))
            if extra:
                details.append("hash bez datoteke: " + ", ".join(extra))
            fail("manifest i ZIP se ne podudaraju (" + "; ".join(details) + ")")

        for member in sorted(actual_payload):
            actual = sha256_member(zf, files[member])
            if actual != expected[member]:
                fail(f"SHA-256 se ne podudara za {member}")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("zip_path", type=Path)
    parser.add_argument("--version", required=True)
    parser.add_argument("--arch", choices=("x64", "x86"), required=True)
    args = parser.parse_args()
    try:
        verify_bundle(args.zip_path, args.version, args.arch)
    except (ValueError, zipfile.BadZipFile, OSError) as exc:
        raise SystemExit(str(exc)) from exc
    print(f"BUNDLE_VERIFICATION=PASS ({args.zip_path.name}, {args.arch})")
    return 0


if __name__ == "__main__":
    sys.exit(main())
