#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
from pathlib import Path
import struct
import unittest

ROOT = Path(__file__).resolve().parents[1]


def load_module():
    path = ROOT / "scripts" / "build_browser_bridge_binaries.py"
    spec = importlib.util.spec_from_file_location("ghostftp_browser_bridge_builder", path)
    if spec is None or spec.loader is None:
        raise RuntimeError("unable to load browser bridge builder")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class BrowserBridgeBinaryTests(unittest.TestCase):
    def setUp(self) -> None:
        self.builder = load_module()

    def test_target_contract_has_windows_and_linux_architectures(self) -> None:
        targets = {(goos, goarch) for goos, goarch, *_ in self.builder.TARGETS}
        self.assertEqual(
            targets,
            {
                ("windows", "amd64"),
                ("windows", "386"),
                ("windows", "arm64"),
                ("linux", "amd64"),
                ("linux", "386"),
                ("linux", "arm64"),
            },
        )

    def test_pe_machine_parser_rejects_wrong_signature(self) -> None:
        with self.assertRaises(ValueError):
            self.builder.pe_machine(b"not-pe")

    def test_pe_machine_parser_reads_machine(self) -> None:
        data = bytearray(256)
        data[:2] = b"MZ"
        struct.pack_into("<I", data, 0x3C, 128)
        data[128:132] = b"PE\0\0"
        struct.pack_into("<H", data, 132, 0x8664)
        self.assertEqual(self.builder.pe_machine(bytes(data)), 0x8664)

    def test_elf_machine_parser_reads_little_and_big_endian(self) -> None:
        little = bytearray(64)
        little[:4] = b"\x7fELF"
        little[5] = 1
        struct.pack_into("<H", little, 18, 62)
        self.assertEqual(self.builder.elf_machine(bytes(little)), 62)

        big = bytearray(64)
        big[:4] = b"\x7fELF"
        big[5] = 2
        struct.pack_into(">H", big, 18, 183)
        self.assertEqual(self.builder.elf_machine(bytes(big)), 183)

    def test_elf_machine_parser_rejects_invalid_endianness(self) -> None:
        data = bytearray(64)
        data[:4] = b"\x7fELF"
        data[5] = 0
        with self.assertRaises(ValueError):
            self.builder.elf_machine(bytes(data))

    def test_build_command_contract_is_offline_and_reproducible(self) -> None:
        source = (ROOT / "scripts" / "build_browser_bridge_binaries.py").read_text(encoding="utf-8")
        for marker in (
            '"GOPROXY": "off"',
            '"GOSUMDB": "off"',
            '"CGO_ENABLED": "0"',
            '"-trimpath"',
            '"-buildvcs=false"',
            '-buildid=',
            'go_telemetry',
            'telemetry != "off"',
            'hashlib.sha256',
        ):
            self.assertIn(marker, source)


if __name__ == "__main__":
    unittest.main()
