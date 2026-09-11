#!/usr/bin/env python3
"""Assemble exact cross-platform UI screenshots and write cryptographic provenance."""

from __future__ import annotations

import argparse
import hashlib
import json
import shutil
from pathlib import Path

MAPPING = (
    ("windows/Ghost-FTP-main-workspace.png", "ghost-ftp-main-workspace.png"),
    ("windows/Ghost-FTP-site-manager.png", "ghost-ftp-site-manager.png"),
    ("windows/Ghost-FTP-bookmarks.png", "ghost-ftp-bookmarks.png"),
    ("windows/Ghost-FTP-settings.png", "ghost-ftp-settings.png"),
    ("windows/Ghost-FTP-about.png", "ghost-ftp-about.png"),
    ("linux/ghost-ftp-linux-main-workspace.png", "ghost-ftp-linux-main-workspace.png"),
    ("linux/ghost-ftp-linux-bookmarks.png", "ghost-ftp-linux-bookmarks.png"),
    ("linux/ghost-ftp-linux-settings.png", "ghost-ftp-linux-settings.png"),
    ("android/ghost-ftp-android-files.png", "ghost-ftp-android-files.png"),
    ("android/ghost-ftp-android-navigation.png", "ghost-ftp-android-navigation.png"),
    ("android/ghost-ftp-android-sites.png", "ghost-ftp-android-sites.png"),
    ("android/ghost-ftp-android-bookmarks.png", "ghost-ftp-android-bookmarks.png"),
    ("android/ghost-ftp-android-transfers.png", "ghost-ftp-android-transfers.png"),
    ("android/ghost-ftp-android-settings.png", "ghost-ftp-android-settings.png"),
    ("android/ghost-ftp-android-about.png", "ghost-ftp-android-about.png"),
)


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--staging", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-sha", required=True)
    parser.add_argument("--workflow-run-id", type=int, required=True)
    args = parser.parse_args()

    if len(args.source_sha) != 40 or any(ch not in "0123456789abcdef" for ch in args.source_sha.lower()):
        raise SystemExit("capture source SHA is not a full hexadecimal Git SHA")

    args.output.mkdir(parents=True, exist_ok=True)
    images: list[dict[str, object]] = []
    for relative_source, target_name in MAPPING:
        source = args.staging / relative_source
        if not source.is_file() or source.stat().st_size < 2048:
            raise SystemExit(f"missing or implausibly small UI evidence: {source}")
        target = args.output / target_name
        shutil.copyfile(source, target)
        images.append(
            {
                "path": target.as_posix(),
                "bytes": target.stat().st_size,
                "sha256": sha256(target),
            }
        )

    manifest = {
        "schema": 1,
        "evidence": "authentic-runtime-capture",
        "capture_source_sha": args.source_sha.lower(),
        "workflow_run_id": args.workflow_run_id,
        "workflow": "Ghost FTP Authentic Cross-Platform UI Screenshots",
        "images": images,
    }
    manifest_path = args.output.parent / "UI-SCREENSHOT-PROVENANCE.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    print(f"UI_EVIDENCE_IMAGES={len(images)}")
    print(f"UI_EVIDENCE_SOURCE_SHA={args.source_sha.lower()}")
    print(f"UI_EVIDENCE_MANIFEST={manifest_path.as_posix()}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
