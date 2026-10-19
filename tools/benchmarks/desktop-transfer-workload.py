#!/usr/bin/env python3
"""Opt-in Windows/Linux transfer fixture + measurements. Never contacts a cloud."""
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import time

SMALL_BYTES = 4096
LARGE_BYTES = 4 * 1024**3 + 1024
CHUNK = 4 * 1024 * 1024


def digest_file(path):
    result = hashlib.sha256()
    with path.open("rb") as stream:
        while block := stream.read(CHUNK):
            result.update(block)
    return result.hexdigest()


def fixture(root, count, large):
    # Refuse existing roots; never delete or overwrite user files.
    root.mkdir(parents=True, mode=0o700, exist_ok=False)
    small_dir = root / "small"
    small_dir.mkdir()
    entries = []
    started = time.monotonic()
    for index in range(count):
        name = f"small/{index:05d}.bin"
        data = hashlib.sha256(f"GhostFTP/{index}".encode()).digest() * (SMALL_BYTES // 32)
        with (root / name).open("xb") as target:
            target.write(data)
        entries.append({"path": name, "size": len(data), "sha256": hashlib.sha256(data).hexdigest()})
    if large:
        (root / "large").mkdir()
        path = root / "large" / "sparse-over-4gib.bin"
        with path.open("xb") as output:
            output.write(b"Ghost FTP integrity fixture\n")
            output.seek(LARGE_BYTES - 1)
            output.write(b"Z")
        entries.append({
            "path": "large/sparse-over-4gib.bin",
            "size": LARGE_BYTES,
            "sha256": digest_file(path),
        })
    manifest = {"schema": 1, "count": len(entries), "entries": entries}
    (root / "manifest.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf8")
    print(json.dumps({"operation": "prepare", "directory": str(root), "files": len(entries),
                      "logical_bytes": sum(i["size"] for i in entries),
                      "wall_seconds": round(time.monotonic() - started, 3),
                      "measurement": "local fixture creation only"}, indent=2))


def validate(root, manifest_path):
    if root.is_symlink() or not root.is_dir():
        raise ValueError("verification root must be a real directory")
    manifest = json.loads(manifest_path.read_text(encoding="utf8"))
    if manifest.get("schema") != 1 or not isinstance(manifest.get("entries"), list):
        raise ValueError("unsupported manifest")
    errors, count, logical_bytes = [], 0, 0
    start = time.monotonic()
    for item in manifest["entries"]:
        rel = item["path"]
        if not (re.fullmatch(r"small/[0-9]{5}\.bin", rel) or
                rel == "large/sparse-over-4gib.bin"):
            raise ValueError("unexpected manifest path")
        path = root.joinpath(*rel.split("/"))
        if path.is_symlink() or path.parent.is_symlink():
            errors.append(f"{rel}: symlink refused")
            continue
        try:
            stat = path.stat()
            if not path.is_file() or stat.st_size != item["size"]:
                errors.append(f"{rel}: wrong type or size")
            elif digest_file(path) != item["sha256"]:
                errors.append(f"{rel}: SHA-256 mismatch")
            else:
                count += 1
                logical_bytes += stat.st_size
        except OSError:
            errors.append(f"{rel}: missing or unreadable")
    data = {"operation": "verify", "files_verified": count,
            "files_expected": len(manifest["entries"]), "bytes_verified": logical_bytes,
            "sha256_verified": not errors,
            "wall_seconds": round(time.monotonic() - start, 3),
            "errors": errors[:20], "measurement": "on-disk integrity only"}
    print(json.dumps(data, indent=2))
    return not errors


def sample(pid, seconds, interval, parallelism, expected_bytes):
    try:
        import psutil
    except ImportError as exc:
        raise RuntimeError("Install psutil for actual target-process CPU/RSS sampling") from exc
    process = psutil.Process(pid)
    before = process.cpu_times()
    begin = time.monotonic()
    peak_rss, samples = 0, 0
    while time.monotonic() - begin < seconds:
        rss = process.memory_info().rss
        peak_rss = max(peak_rss, rss)
        samples += 1
        time.sleep(interval)
    after = process.cpu_times()
    elapsed = time.monotonic() - begin
    cpu_seconds = after.user + after.system - before.user - before.system
    print(json.dumps({"operation": "observe", "pid": pid, "parallelism_label": parallelism,
                      "wall_seconds": round(elapsed, 3), "cpu_seconds": round(cpu_seconds, 3),
                      "cpu_percent_one_core": round(100 * cpu_seconds / elapsed, 2),
                      "peak_rss_bytes_sampled": peak_rss, "samples": samples,
                      "assumed_payload_bytes": expected_bytes,
                      "assumed_mib_per_second": round(expected_bytes / elapsed / (1024**2), 3)
                      if expected_bytes is not None else None,
                      "ui_event_latency_ms": None,
                      "note": "Cannot attribute bytes to this process or measure UI responsiveness without separate instrumentation"},
                     indent=2))


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="action", required=True)
    make = sub.add_parser("prepare")
    make.add_argument("--root", type=Path, required=True)
    make.add_argument("--small-count", type=int, default=10000)
    make.add_argument("--large", action="store_true", help="Explicitly create and hash >4 GiB sparse file")
    check = sub.add_parser("verify")
    check.add_argument("--root", type=Path, required=True)
    check.add_argument("--manifest", type=Path, required=True)
    watch = sub.add_parser("observe")
    watch.add_argument("--pid", required=True, type=int)
    watch.add_argument("--seconds", required=True, type=int)
    watch.add_argument("--interval", type=float, default=0.25)
    watch.add_argument("--parallelism", required=True, type=int, choices=[1, 4, 8, 16, 32])
    watch.add_argument("--expected-bytes", type=int, default=None)
    a = parser.parse_args()
    if a.action == "prepare":
        if not 1 <= a.small_count <= 10000:
            parser.error("--small-count must be 1..10000")
        fixture(a.root, a.small_count, a.large)
    elif a.action == "verify":
        if not validate(a.root, a.manifest):
            sys.exit(1)
    else:
        if a.seconds < 1 or not (0.05 <= a.interval <= 10):
            parser.error("--seconds and --interval must be positive and bounded")
        sample(a.pid, a.seconds, a.interval, a.parallelism, a.expected_bytes)


if __name__ == "__main__":
    main()
