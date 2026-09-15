#!/usr/bin/env python3
"""Build and verify Ghost FTP Native Messaging bridge binaries."""

from __future__ import annotations

import argparse
import hashlib
import os
from pathlib import Path
import shutil
import struct
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
SOURCE = "./cmd/ghostftp-native-host"

TARGETS = (
    ("windows", "amd64", "Windows-x64.exe", "pe", 0x8664),
    ("windows", "386", "Windows-x86.exe", "pe", 0x014C),
    ("windows", "arm64", "Windows-arm64.exe", "pe", 0xAA64),
    ("linux", "amd64", "Linux-amd64", "elf", 62),
    ("linux", "386", "Linux-i386", "elf", 3),
    ("linux", "arm64", "Linux-arm64", "elf", 183),
)


def read_version() -> str:
    value = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    parts = value.split(".")
    if len(parts) != 3 or any(not part.isdigit() for part in parts):
        raise ValueError("VERSION must be a numeric semantic version")
    return value


def go_binary() -> str:
    binary = shutil.which("go")
    if not binary:
        raise RuntimeError("Go is not available in PATH")
    return binary


def go_telemetry(go: str) -> str:
    result = subprocess.run(
        [go, "telemetry"],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )
    return result.stdout.strip()


def pe_machine(data: bytes) -> int:
    if len(data) < 0x40 or data[:2] != b"MZ":
        raise ValueError("binary is not a PE executable")
    pe_offset = struct.unpack_from("<I", data, 0x3C)[0]
    if pe_offset + 6 > len(data) or data[pe_offset : pe_offset + 4] != b"PE\0\0":
        raise ValueError("PE signature is missing")
    return struct.unpack_from("<H", data, pe_offset + 4)[0]


def elf_machine(data: bytes) -> int:
    if len(data) < 20 or data[:4] != b"\x7fELF":
        raise ValueError("binary is not an ELF executable")
    endian = data[5]
    if endian == 1:
        fmt = "<H"
    elif endian == 2:
        fmt = ">H"
    else:
        raise ValueError("ELF endianness is invalid")
    return struct.unpack_from(fmt, data, 18)[0]


def verify_binary(path: Path, kind: str, expected_machine: int) -> None:
    data = path.read_bytes()
    if len(data) < 4096:
        raise ValueError(f"bridge binary is unexpectedly small: {path.name}")
    actual = pe_machine(data) if kind == "pe" else elf_machine(data)
    if actual != expected_machine:
        raise ValueError(
            f"{path.name}: machine mismatch: got 0x{actual:x}, expected 0x{expected_machine:x}"
        )


def build_binary(go: str, version: str, goos: str, goarch: str, destination: Path) -> None:
    env = os.environ.copy()
    env.update(
        {
            "GOTOOLCHAIN": "local",
            "GOPROXY": "off",
            "GOSUMDB": "off",
            "CGO_ENABLED": "0",
            "GOOS": goos,
            "GOARCH": goarch,
            "GOWORK": "off",
        }
    )
    if goarch == "386":
        env["GO386"] = "sse2"
    else:
        env.pop("GO386", None)
    if goarch == "amd64":
        env["GOAMD64"] = "v1"
    else:
        env.pop("GOAMD64", None)

    command = [
        go,
        "build",
        "-mod=readonly",
        "-trimpath",
        "-buildvcs=false",
        "-ldflags",
        f"-s -w -buildid= -X main.version={version}",
        "-o",
        str(destination),
        SOURCE,
    ]
    subprocess.run(command, cwd=ROOT, env=env, check=True)


def write_hashes(paths: list[Path], destination: Path) -> None:
    lines = []
    for path in sorted(paths, key=lambda item: item.name):
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        lines.append(f"{digest}  {path.name}")
    destination.write_text("\n".join(lines) + "\n", encoding="ascii", newline="\n")


def build(output_dir: Path) -> list[Path]:
    version = read_version()
    go = go_binary()
    telemetry = go_telemetry(go)
    if telemetry != "off":
        raise RuntimeError(f"Go telemetry must be disabled before production bridge builds (current: {telemetry})")

    output_dir.mkdir(parents=True, exist_ok=True)
    for stale in output_dir.glob("Ghost-FTP-*-Native-Host-*"):
        if stale.is_file():
            stale.unlink()
    hash_path = output_dir / "SHA256.txt"
    hash_path.unlink(missing_ok=True)

    outputs: list[Path] = []
    for goos, goarch, suffix, kind, machine in TARGETS:
        destination = output_dir / f"Ghost-FTP-{version}-Native-Host-{suffix}"
        build_binary(go, version, goos, goarch, destination)
        verify_binary(destination, kind, machine)
        if goos != "windows":
            destination.chmod(0o755)
        outputs.append(destination)

    write_hashes(outputs, hash_path)
    outputs.append(hash_path)
    return outputs


def main() -> int:
    parser = argparse.ArgumentParser(description="Build Ghost FTP browser bridge native-host binaries")
    parser.add_argument("--output", type=Path, default=ROOT / "dist" / "browser-bridge")
    args = parser.parse_args()

    try:
        outputs = build(args.output)
    except (OSError, UnicodeError, ValueError, RuntimeError, subprocess.CalledProcessError) as exc:
        print(f"BROWSER_BRIDGE_BUILD=FAILED: {exc}", file=sys.stderr)
        return 1

    print("BROWSER_BRIDGE_BUILD=PASS")
    print("BROWSER_BRIDGE_TARGETS=windows-x64,windows-x86,windows-arm64,linux-amd64,linux-i386,linux-arm64")
    for output in outputs:
        print(f"BROWSER_BRIDGE_ARTIFACT={output}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
