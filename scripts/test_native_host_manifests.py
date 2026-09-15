#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


def load_module():
    path = ROOT / "scripts" / "build_native_host_manifests.py"
    spec = importlib.util.spec_from_file_location("ghostftp_native_host_manifests", path)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load native-host manifest builder")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class NativeHostManifestTests(unittest.TestCase):
    def setUp(self) -> None:
        self.builder = load_module()

    def test_firefox_manifest_uses_fixed_extension_identity(self) -> None:
        manifest = self.builder.firefox_manifest("/opt/ghostftp/ghostftp-native-host")
        self.assertEqual(manifest["name"], "com.ghostftp.bridge")
        self.assertEqual(manifest["type"], "stdio")
        self.assertEqual(manifest["allowed_extensions"], ["ghostftp-connection-helper@ghostftp.com"])
        self.assertNotIn("allowed_origins", manifest)

    def test_chromium_manifest_requires_exact_ids_and_never_uses_wildcard(self) -> None:
        extension_id = "abcdefghijklmnopabcdefghijklmnop"
        manifest = self.builder.chromium_manifest("/opt/ghostftp/ghostftp-native-host", [extension_id])
        self.assertEqual(manifest["allowed_origins"], [f"chrome-extension://{extension_id}/"])
        self.assertNotIn("*", json.dumps(manifest))
        self.assertNotIn("allowed_extensions", manifest)

    def test_chromium_ids_are_deduplicated_and_sorted(self) -> None:
        first = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
        second = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        self.assertEqual(self.builder.normalize_chromium_ids([first, second, first]), [second, first])

    def test_invalid_chromium_id_is_rejected(self) -> None:
        for value in ("", "abc", "z" * 32, "a" * 31, "a" * 33, "A" * 31 + "1"):
            with self.subTest(value=value), self.assertRaises(ValueError):
                self.builder.normalize_chromium_ids([value])

    def test_host_path_must_be_absolute_and_control_free(self) -> None:
        with self.assertRaises(ValueError):
            self.builder.normalize_host_path("relative/ghostftp-native-host")
        with self.assertRaises(ValueError):
            self.builder.normalize_host_path("/opt/ghostftp/host\nnext")

    def test_build_without_chromium_id_writes_only_firefox_manifest(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            outputs = self.builder.build(Path(tmp), "/opt/ghostftp/ghostftp-native-host", [], False)
            self.assertEqual([path.name for path in outputs], ["com.ghostftp.bridge.firefox.json"])
            self.assertFalse((Path(tmp) / "com.ghostftp.bridge.chromium.json").exists())

    def test_require_chromium_fails_closed_without_id(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            with self.assertRaises(ValueError):
                self.builder.build(Path(tmp), "/opt/ghostftp/ghostftp-native-host", [], True)

    def test_build_is_deterministic(self) -> None:
        extension_id = "abcdefghijklmnopabcdefghijklmnop"
        with tempfile.TemporaryDirectory() as left_tmp, tempfile.TemporaryDirectory() as right_tmp:
            left = self.builder.build(Path(left_tmp), "/opt/ghostftp/ghostftp-native-host", [extension_id], True)
            right = self.builder.build(Path(right_tmp), "/opt/ghostftp/ghostftp-native-host", [extension_id], True)
            self.assertEqual([path.name for path in left], [path.name for path in right])
            for a, b in zip(left, right):
                self.assertEqual(a.read_bytes(), b.read_bytes())


if __name__ == "__main__":
    unittest.main()
