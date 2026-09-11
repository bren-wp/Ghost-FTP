#!/usr/bin/env python3
"""Assemble and verify exact cross-platform UI screenshots with cryptographic provenance."""

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


def verify_manifest(manifest_path: Path, source_sha: str, workflow_run_id: int) -> int:
    manifest = json.loads(manifest_path.read_text(encoding="utf-8"))
    expected_sha = source_sha.lower()
    if manifest.get("schema") != 1:
        raise SystemExit("evidence manifest schema mismatch")
    if manifest.get("evidence") != "authentic-runtime-capture":
        raise SystemExit("evidence manifest type mismatch")
    if manifest.get("capture_source_sha") != expected_sha:
        raise SystemExit("evidence manifest source SHA mismatch")
    if manifest.get("workflow_run_id") != workflow_run_id:
        raise SystemExit("evidence manifest workflow run mismatch")

    images = manifest.get("images")
    if not isinstance(images, list) or len(images) != len(MAPPING):
        raise SystemExit(
            f"evidence manifest must contain exactly {len(MAPPING)} verified images"
        )

    expected_names = {target_name for _, target_name in MAPPING}
    observed_names: set[str] = set()
    for item in images:
        if not isinstance(item, dict):
            raise SystemExit("evidence manifest image entry is invalid")
        raw_path = item.get("path")
        if not isinstance(raw_path, str) or not raw_path:
            raise SystemExit("evidence manifest image path is invalid")
        path = Path(raw_path)
        if path.name not in expected_names or path.name in observed_names:
            raise SystemExit(f"unexpected or duplicate evidence image: {path}")
        if not path.is_file():
            raise SystemExit(f"evidence image is missing: {path}")
        size = path.stat().st_size
        if size < 2048 or item.get("bytes") != size:
            raise SystemExit(f"evidence image size mismatch: {path}")
        digest = sha256(path)
        if item.get("sha256") != digest:
            raise SystemExit(f"evidence image SHA-256 mismatch: {path}")
        observed_names.add(path.name)

    if observed_names != expected_names:
        missing = ", ".join(sorted(expected_names - observed_names))
        raise SystemExit(f"evidence bundle is incomplete: {missing}")
    return len(images)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--staging", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--source-sha", required=True)
    parser.add_argument("--workflow-run-id", type=int, required=True)
    args = parser.parse_args()

    source_sha = args.source_sha.lower()
    if len(source_sha) != 40 or any(ch not in "0123456789abcdef" for ch in source_sha):
        raise SystemExit("capture source SHA is not a full hexadecimal Git SHA")
    if args.workflow_run_id <= 0:
        raise SystemExit("workflow run ID must be positive")

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
        "capture_source_sha": source_sha,
        "workflow_run_id": args.workflow_run_id,
        "workflow": "Ghost FTP Authentic Cross-Platform UI Screenshots",
        "images": images,
    }
    manifest_path = args.output.parent / "UI-SCREENSHOT-PROVENANCE.json"
    manifest_path.write_text(json.dumps(manifest, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    verified_count = verify_manifest(manifest_path, source_sha, args.workflow_run_id)
    print(f"UI_EVIDENCE_IMAGES={verified_count}")
    print(f"UI_EVIDENCE_SOURCE_SHA={source_sha}")
    print(f"UI_EVIDENCE_MANIFEST={manifest_path.as_posix()}")
    print(
        f"AUTHENTIC_UI_EVIDENCE=VERIFIED SOURCE_SHA={source_sha} "
        f"IMAGES={verified_count}"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
