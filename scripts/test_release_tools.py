#!/usr/bin/env python3

from __future__ import annotations

import hashlib
import tempfile
import unittest
import zipfile
from pathlib import Path
from types import SimpleNamespace
from unittest import mock

import audit_web
from verify_bundle import MANIFEST, required_members, verify_bundle

VERSION = "9.8.7"


def payload_members(arch: str = "x64") -> dict[str, bytes]:
    members = {}
    for name in required_members(VERSION, arch) - {MANIFEST}:
        members[name] = ("sadržaj:" + name).encode("utf-8")
    return members


def manifest_for(members: dict[str, bytes], *, corrupt: str | None = None) -> str:
    lines = []
    for name, data in sorted(members.items()):
        digest = "0" * 64 if name == corrupt else hashlib.sha256(data).hexdigest()
        lines.append(f"{digest}  {name}")
    return "\n".join(lines) + "\n"


def write_bundle(path: Path, members: dict[str, bytes], *, manifest_override: str | None = None) -> None:
    manifest = manifest_override if manifest_override is not None else manifest_for(members)
    with zipfile.ZipFile(path, "w", compression=zipfile.ZIP_DEFLATED) as zf:
        for name, data in members.items():
            zf.writestr(name, data)
        zf.writestr(MANIFEST, manifest.encode("ascii"))


class AuditWebSubprocessTests(unittest.TestCase):
    def test_run_checked_pins_utf8_decoding(self) -> None:
        captured: dict[str, object] = {}

        def fake_run(command: list[str], **kwargs: object) -> SimpleNamespace:
            captured.update(kwargs)
            return SimpleNamespace(returncode=0, stdout="Greška: čćžšđ")

        with mock.patch.object(audit_web.subprocess, "run", side_effect=fake_run):
            audit_web.run_checked(["fake-command"], label="UTF-8 regression")

        self.assertEqual(captured.get("encoding"), "utf-8")
        self.assertEqual(captured.get("errors"), "replace")
        self.assertIs(captured.get("text"), True)

    def test_tracked_files_pins_utf8_decoding(self) -> None:
        captured: dict[str, object] = {}

        def fake_check_output(command: list[str], **kwargs: object) -> str:
            captured.update(kwargs)
            return "ByFTP WEB/README.md\n"

        with mock.patch.object(audit_web.subprocess, "check_output", side_effect=fake_check_output):
            self.assertEqual(audit_web.tracked_web_files(), ["ByFTP WEB/README.md"])

        self.assertEqual(captured.get("encoding"), "utf-8")
        self.assertEqual(captured.get("errors"), "replace")
        self.assertIs(captured.get("text"), True)


class VerifyBundleTests(unittest.TestCase):
    def test_valid_x64_and_x86_bundles_pass(self) -> None:
        for arch in ("x64", "x86"):
            with self.subTest(arch=arch), tempfile.TemporaryDirectory() as tmp:
                bundle = Path(tmp) / "bundle.zip"
                write_bundle(bundle, payload_members(arch))
                verify_bundle(bundle, VERSION, arch)

    def test_hash_mismatch_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            bundle = Path(tmp) / "bundle.zip"
            members = payload_members()
            write_bundle(bundle, members, manifest_override=manifest_for(members, corrupt="README.md"))
            with self.assertRaises(ValueError):
                verify_bundle(bundle, VERSION, "x64")

    def test_traversal_member_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            bundle = Path(tmp) / "bundle.zip"
            members = payload_members()
            members["../izlaz.txt"] = "ne smije proći".encode("utf-8")
            write_bundle(bundle, members)
            with self.assertRaises(ValueError):
                verify_bundle(bundle, VERSION, "x64")

    def test_unmanifested_file_fails(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            bundle = Path(tmp) / "bundle.zip"
            members = payload_members()
            original = dict(members)
            members["neocekivano.txt"] = "višak".encode("utf-8")
            write_bundle(bundle, members, manifest_override=manifest_for(original))
            with self.assertRaises(ValueError):
                verify_bundle(bundle, VERSION, "x64")

    def test_internal_uninstaller_or_verification_fails(self) -> None:
        for forbidden in (f"ByFTP-{VERSION}-Uninstall-x64.exe", "verification.txt", "verification-x64.txt"):
            with self.subTest(forbidden=forbidden), tempfile.TemporaryDirectory() as tmp:
                bundle = Path(tmp) / "bundle.zip"
                members = payload_members()
                members[forbidden] = b"interni artefakt"
                write_bundle(bundle, members)
                with self.assertRaises(ValueError):
                    verify_bundle(bundle, VERSION, "x64")


if __name__ == "__main__":
    unittest.main()
