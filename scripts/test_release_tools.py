#!/usr/bin/env python3

from __future__ import annotations

import hashlib
import tempfile
import unittest
import zipfile
from pathlib import Path

from verify_bundle import MANIFEST, required_members, verify_bundle

VERSION = "9.8.7"


def payload_members() -> dict[str, bytes]:
    members = {}
    for name in required_members(VERSION) - {MANIFEST}:
        members[name] = ("sadržaj:" + name).encode("utf-8")
    return members


def write_bundle(path: Path, members: dict[str, bytes], *, manifest_override: str | None = None) -> None:
    manifest = manifest_override
    if manifest is None:
        manifest = "\n".join(
            f"{hashlib.sha256(data).hexdigest()}  {name}" for name, data in sorted(members.items())
        ) + "\n"
    with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        for name, data in members.items():
            zf.writestr(name, data)
        zf.writestr(MANIFEST, manifest.encode("ascii"))


class VerifyBundleTests(unittest.TestCase):
    def test_valid_bundle_passes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bundle.zip"
            write_bundle(path, payload_members())
            verify_bundle(path, VERSION)

    def test_hash_mismatch_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bundle.zip"
            members = payload_members()
            write_bundle(path, members)
            # Izmjena nakon generiranja manifesta mora biti otkrivena.
            with zipfile.ZipFile(path, "a", compression=zipfile.ZIP_DEFLATED) as zf:
                zf.writestr("README.md", b"izmijenjeno")
            with self.assertRaises(ValueError):
                verify_bundle(path, VERSION)

    def test_traversal_member_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bundle.zip"
            members = payload_members()
            members["../izlaz.txt"] = b"ne smije proći"
            write_bundle(path, members)
            with self.assertRaises(ValueError):
                verify_bundle(path, VERSION)

    def test_unmanifested_file_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "bundle.zip"
            members = payload_members()
            original = dict(members)
            members["neocekivano.txt"] = b"višak"
            manifest = "\n".join(
                f"{hashlib.sha256(data).hexdigest()}  {name}" for name, data in sorted(original.items())
            ) + "\n"
            write_bundle(path, members, manifest_override=manifest)
            with self.assertRaises(ValueError):
                verify_bundle(path, VERSION)


if __name__ == "__main__":
    unittest.main()
