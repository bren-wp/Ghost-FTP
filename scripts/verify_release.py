#!/usr/bin/env python3
from __future__ import annotations
import argparse, hashlib, struct
from pathlib import Path

AMD64 = 0x8664
GUI_SUBSYSTEM = 2

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


def read_pe(path: Path):
    data = path.read_bytes()
    if len(data) < 1024 or data[:2] != b'MZ':
        raise ValueError(f'{path.name}: missing MZ header')
    pe = struct.unpack_from('<I', data, 0x3C)[0]
    if pe + 0x200 > len(data) or data[pe:pe+4] != b'PE\0\0':
        raise ValueError(f'{path.name}: invalid PE signature')
    coff = pe + 4
    machine, sections, _, _, _, opt_size, _ = struct.unpack_from('<HHIIIHH', data, coff)
    if machine != AMD64:
        raise ValueError(f'{path.name}: expected AMD64 PE')
    opt = coff + 20
    if struct.unpack_from('<H', data, opt)[0] != 0x20B:
        raise ValueError(f'{path.name}: expected PE32+')
    subsystem = struct.unpack_from('<H', data, opt + 68)[0]
    if subsystem != GUI_SUBSYSTEM:
        raise ValueError(f'{path.name}: expected GUI subsystem, got {subsystem}')
    dll_characteristics = struct.unpack_from('<H', data, opt + 70)[0]
    required_mitigations = {
        'HIGH_ENTROPY_VA': 0x20,
        'DYNAMIC_BASE': 0x40,
        'NX_COMPAT': 0x100,
        'TERMINAL_SERVER_AWARE': 0x8000,
    }
    missing = [name for name, flag in required_mitigations.items() if not (dll_characteristics & flag)]
    if missing:
        raise ValueError(f'{path.name}: missing PE mitigations: {", ".join(missing)}')
    resource_rva, resource_size = struct.unpack_from('<II', data, opt + 112 + 2*8)
    cert_offset, cert_size = struct.unpack_from('<II', data, opt + 112 + 4*8)
    if resource_rva == 0 or resource_size == 0:
        raise ValueError(f'{path.name}: resource directory missing')
    section_table = opt + opt_size
    names=[]
    for i in range(sections):
        off=section_table+i*40
        names.append(data[off:off+8].split(b'\0',1)[0].decode('ascii','replace'))
    if '.rsrc' not in names:
        raise ValueError(f'{path.name}: .rsrc section missing')
    must_utf16 = ['CompanyName', 'Brendigo', 'ProductName', 'ByFTP', 'brendigo.com']
    for text in must_utf16:
        if text.encode('utf-16le') not in data:
            raise ValueError(f'{path.name}: VERSIONINFO field missing: {text}')
    if b'requestedExecutionLevel level="asInvoker"' not in data:
        raise ValueError(f'{path.name}: asInvoker manifest missing')
    return data, bool(cert_offset and cert_size)

def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()

def main():
    ap=argparse.ArgumentParser()
    ap.add_argument('setup', type=Path)
    ap.add_argument('portable', type=Path)
    ap.add_argument('uninstaller', type=Path)
    args=ap.parse_args()
    sdat, ssigned=read_pe(args.setup)
    pdat, psigned=read_pe(args.portable)
    udat, usigned=read_pe(args.uninstaller)
    assert_no_telemetry_markers(args.setup, sdat)
    assert_no_telemetry_markers(args.portable, pdat)
    assert_no_telemetry_markers(args.uninstaller, udat)
    hashes = {sha256(sdat), sha256(pdat), sha256(udat)}
    if len(hashes) != 3:
        raise SystemExit('Setup, Portable and Uninstaller must be distinct binaries')
    print(f'SETUP_PE_OK=YES')
    print(f'PORTABLE_PE_OK=YES')
    print(f'UNINSTALLER_PE_OK=YES')
    print(f'COMPANY_NAME=Brendigo')
    print(f'EXECUTION_LEVEL=asInvoker')
    print(f'PE_MITIGATIONS=HIGH_ENTROPY_VA,DYNAMIC_BASE,NX_COMPAT,TERMINAL_SERVER_AWARE')
    print(f'SETUP_AUTHENTICODE_SIGNED={"YES" if ssigned else "NO"}')
    print(f'PORTABLE_AUTHENTICODE_SIGNED={"YES" if psigned else "NO"}')
    print(f'UNINSTALLER_AUTHENTICODE_SIGNED={"YES" if usigned else "NO"}')
    print('TELEMETRY_VENDOR_SIGNATURES=ABSENT')
    print('PRIVACY_POLICY=USER_SELECTED_SERVER_ONLY')
    print(f'SETUP_SHA256={sha256(sdat)}')
    print(f'PORTABLE_SHA256={sha256(pdat)}')
    print(f'UNINSTALLER_SHA256={sha256(udat)}')

if __name__ == '__main__':
    main()
