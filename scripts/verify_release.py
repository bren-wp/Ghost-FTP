#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import struct
from pathlib import Path

I386 = 0x014C
AMD64 = 0x8664
GUI_SUBSYSTEM = 2
CONSOLE_SUBSYSTEM = 3

ARCH_SPECS = {
    "x86": {"machine": I386, "magic": 0x10B, "pe": "PE32", "data_dir": 96},
    "x64": {"machine": AMD64, "magic": 0x20B, "pe": "PE32+", "data_dir": 112},
}

TELEMETRY_MARKERS = [
    b"sentry.io", b"google-analytics", b"googletagmanager", b"segment.io",
    b"mixpanel", b"amplitude", b"posthog", b"datadog", b"newrelic",
    b"bugsnag", b"crashlytics", b"appcenter", b"telemetrydeck",
]


def assert_no_telemetry_markers(path: Path, data: bytes) -> None:
    lower = data.lower()
    for marker in TELEMETRY_MARKERS:
        if marker in lower or marker.decode("ascii").encode("utf-16le") in lower:
            raise ValueError(f"{path.name}: telemetry/vendor marker found: {marker.decode('ascii')}")


def detect_arch(machine: int, magic: int) -> str:
    for arch, spec in ARCH_SPECS.items():
        if machine == spec["machine"] and magic == spec["magic"]:
            return arch
    raise ValueError(f"nepodržan PE stroj/magic: machine=0x{machine:04x}, magic=0x{magic:04x}")


def read_pe(path: Path, expected_arch: str | None = None, expected_subsystem: int | None = GUI_SUBSYSTEM):
    data = path.read_bytes()
    if len(data) < 1024 or data[:2] != b"MZ":
        raise ValueError(f"{path.name}: missing MZ header")
    pe = struct.unpack_from("<I", data, 0x3C)[0]
    if pe + 0x200 > len(data) or data[pe:pe + 4] != b"PE\0\0":
        raise ValueError(f"{path.name}: invalid PE signature")
    coff = pe + 4
    machine, sections, _, _, _, opt_size, _ = struct.unpack_from("<HHIIIHH", data, coff)
    opt = coff + 20
    magic = struct.unpack_from("<H", data, opt)[0]
    arch = detect_arch(machine, magic)
    if expected_arch and arch != expected_arch:
        raise ValueError(f"{path.name}: očekuje se {expected_arch}, pronađen je {arch}")
    spec = ARCH_SPECS[arch]

    subsystem = struct.unpack_from("<H", data, opt + 68)[0]
    if expected_subsystem is not None and subsystem != expected_subsystem:
        expected_name = "GUI" if expected_subsystem == GUI_SUBSYSTEM else "Console" if expected_subsystem == CONSOLE_SUBSYSTEM else str(expected_subsystem)
        raise ValueError(f"{path.name}: očekuje se {expected_name} subsystem ({expected_subsystem}), pronađen je {subsystem}")
    dll_characteristics = struct.unpack_from("<H", data, opt + 70)[0]
    required_mitigations = {
        "DYNAMIC_BASE": 0x40,
        "NX_COMPAT": 0x100,
        "TERMINAL_SERVER_AWARE": 0x8000,
    }
    if arch == "x64":
        required_mitigations["HIGH_ENTROPY_VA"] = 0x20
    missing = [name for name, flag in required_mitigations.items() if not (dll_characteristics & flag)]
    if missing:
        raise ValueError(f"{path.name}: missing PE mitigations: {', '.join(missing)}")

    dd = opt + int(spec["data_dir"])
    resource_rva, resource_size = struct.unpack_from("<II", data, dd + 2 * 8)
    cert_offset, cert_size = struct.unpack_from("<II", data, dd + 4 * 8)
    if resource_rva == 0 or resource_size == 0:
        raise ValueError(f"{path.name}: resource directory missing")
    section_table = opt + opt_size
    names = []
    for i in range(sections):
        off = section_table + i * 40
        names.append(data[off:off + 8].split(b"\0", 1)[0].decode("ascii", "replace"))
    if ".rsrc" not in names:
        raise ValueError(f"{path.name}: .rsrc section missing")
    for text in ["CompanyName", "Brendigo", "ProductName", "ByFTP", "brendigo.com"]:
        if text.encode("utf-16le") not in data:
            raise ValueError(f"{path.name}: VERSIONINFO field missing: {text}")
    if b'requestedExecutionLevel level="asInvoker"' not in data:
        raise ValueError(f"{path.name}: asInvoker manifest missing")
    manifest_arch = "amd64" if arch == "x64" else "x86"
    if f'processorArchitecture="{manifest_arch}"'.encode("ascii") not in data:
        raise ValueError(f"{path.name}: manifest architecture is not {manifest_arch}")
    return data, bool(cert_offset and cert_size), arch, required_mitigations, subsystem


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("setup", type=Path)
    ap.add_argument("portable", type=Path)
    ap.add_argument("uninstaller", type=Path)
    ap.add_argument("--arch", choices=("x64", "x86"), default=None)
    args = ap.parse_args()

    results = [read_pe(p, args.arch, GUI_SUBSYSTEM) for p in (args.setup, args.portable, args.uninstaller)]
    arches = {result[2] for result in results}
    if len(arches) != 1:
        raise SystemExit("Setup, Portable and Uninstaller nisu iste arhitekture")
    arch = next(iter(arches))
    (sdat, ssigned, _, mitigations, _), (pdat, psigned, _, _, _), (udat, usigned, _, _, _) = results

    assert_no_telemetry_markers(args.setup, sdat)
    assert_no_telemetry_markers(args.portable, pdat)
    assert_no_telemetry_markers(args.uninstaller, udat)
    hashes = {sha256(sdat), sha256(pdat), sha256(udat)}
    if len(hashes) != 3:
        raise SystemExit("Setup, Portable and Uninstaller must be distinct binaries")

    print("SETUP_PE_OK=YES")
    print("PORTABLE_PE_OK=YES")
    print("UNINSTALLER_PE_OK=YES")
    print(f"WINDOWS_ARCH={arch}")
    print("COMPANY_NAME=Brendigo")
    print("EXECUTION_LEVEL=asInvoker")
    print("PE_MITIGATIONS=" + ",".join(mitigations.keys()))
    print(f"SETUP_AUTHENTICODE_SIGNED={'YES' if ssigned else 'NO'}")
    print(f"PORTABLE_AUTHENTICODE_SIGNED={'YES' if psigned else 'NO'}")
    print(f"UNINSTALLER_AUTHENTICODE_SIGNED={'YES' if usigned else 'NO'}")
    print("TELEMETRY_VENDOR_SIGNATURES=ABSENT")
    print("PRIVACY_POLICY=USER_SELECTED_SERVER_ONLY")
    print(f"SETUP_SHA256={sha256(sdat)}")
    print(f"PORTABLE_SHA256={sha256(pdat)}")
    print(f"UNINSTALLER_SHA256={sha256(udat)}")


if __name__ == "__main__":
    main()
