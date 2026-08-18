#!/usr/bin/env python3
from __future__ import annotations

import argparse
from pathlib import Path

from verify_release import (
    CONSOLE_SUBSYSTEM,
    GUI_SUBSYSTEM,
    assert_no_telemetry_markers,
    read_pe,
    sha256,
)


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("exe", type=Path)
    ap.add_argument("--arch", choices=("x64", "x86"), required=True)
    ap.add_argument("--subsystem", choices=("gui", "console"), required=True)
    args = ap.parse_args()

    subsystem = GUI_SUBSYSTEM if args.subsystem == "gui" else CONSOLE_SUBSYSTEM
    data, signed, arch, mitigations, actual_subsystem = read_pe(args.exe, args.arch, subsystem)
    assert_no_telemetry_markers(args.exe, data)
    print("CLIENT_PE_OK=YES")
    print(f"WINDOWS_ARCH={arch}")
    print(f"WINDOWS_SUBSYSTEM={actual_subsystem}")
    print("PE_MITIGATIONS=" + ",".join(mitigations.keys()))
    print(f"AUTHENTICODE_SIGNED={'YES' if signed else 'NO'}")
    print("TELEMETRY_VENDOR_SIGNATURES=ABSENT")
    print(f"SHA256={sha256(data)}")


if __name__ == "__main__":
    main()
