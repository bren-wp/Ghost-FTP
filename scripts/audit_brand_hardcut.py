#!/usr/bin/env python3
"""Fail closed if the retired product identifier reappears anywhere in tracked source."""

from __future__ import annotations

import re
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RETIRED = re.compile(r"by[\s_-]?ftp", re.IGNORECASE)


def fail(message: str) -> None:
    raise SystemExit("BRAND_HARDCUT_FAILED: " + message)


def main() -> int:
    raw = subprocess.check_output(["git", "ls-files", "-z"], cwd=ROOT)
    violations: list[str] = []

    for item in raw.split(b"\0"):
        if not item:
            continue
        rel = item.decode("utf-8", "strict")
        if RETIRED.search(rel):
            violations.append("path:" + rel)
            continue

        path = ROOT / rel
        if not path.is_file():
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except (UnicodeDecodeError, OSError):
            continue
        if RETIRED.search(text):
            violations.append("content:" + rel)

    if violations:
        fail("retired identifier found: " + ", ".join(violations))

    print("BRAND_HARDCUT=PASS")
    print("PUBLIC_BRAND=Ghost FTP")
    print("TECHNICAL_IDENTITY=GhostFTP")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
