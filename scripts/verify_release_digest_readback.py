#!/usr/bin/env python3
from __future__ import annotations

import argparse
import hashlib
import json
import re
from pathlib import Path

SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
TAG_RE = re.compile(r"^ghostftp-v(\d+\.\d+\.\d+)$")
EXPECTED_RELEASE_FILES = 17


def fail(message: str) -> None:
    raise ValueError(message)


def sha256_file(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            h.update(chunk)
    return h.hexdigest()


def parse_build_metadata(path: Path) -> dict[str, str]:
    if not path.is_file():
        fail("BUILD-METADATA.txt is missing from the release bundle")
    result: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw.strip():
            continue
        if "=" not in raw:
            fail(f"invalid BUILD-METADATA.txt line: {raw!r}")
        key, value = raw.split("=", 1)
        key = key.strip()
        value = value.strip()
        if not key or key in result:
            fail(f"invalid or duplicate BUILD-METADATA.txt key: {key!r}")
        result[key] = value
    return result


def parse_manifest(path: Path) -> dict[str, str]:
    if not path.is_file():
        fail("SHA256.txt is missing from the release bundle")
    result: dict[str, str] = {}
    for raw in path.read_text(encoding="utf-8").splitlines():
        if not raw.strip():
            continue
        match = re.fullmatch(r"([0-9a-f]{64})  (.+)", raw)
        if not match:
            fail(f"invalid SHA256.txt line: {raw!r}")
        digest, name = match.groups()
        if name in result:
            fail(f"duplicate SHA256.txt entry: {name}")
        if Path(name).name != name or name in {".", ".."}:
            fail(f"unsafe SHA256.txt filename: {name!r}")
        result[name] = digest
    return result


def verify_release(bundle_dir: Path, release_json_path: Path, expected_commit: str) -> dict[str, str]:
    bundle_dir = bundle_dir.resolve()
    if not bundle_dir.is_dir():
        fail(f"release bundle directory is unavailable: {bundle_dir}")
    if not re.fullmatch(r"[0-9a-f]{40}", expected_commit):
        fail(f"invalid expected commit SHA: {expected_commit!r}")

    files = sorted(path for path in bundle_dir.iterdir() if path.is_file())
    if len(files) != EXPECTED_RELEASE_FILES:
        fail(f"release bundle has {len(files)} files; expected {EXPECTED_RELEASE_FILES}")
    local_names = {path.name for path in files}

    metadata = parse_build_metadata(bundle_dir / "BUILD-METADATA.txt")
    version = metadata.get("VERSION", "")
    tag = metadata.get("RELEASE_TAG", "")
    commit = metadata.get("COMMIT", "")
    public_files = metadata.get("PUBLIC_RELEASE_FILES", "")
    match = TAG_RE.fullmatch(tag)
    if not match or match.group(1) != version:
        fail(f"release tag/version mismatch: tag={tag!r} version={version!r}")
    if commit != expected_commit:
        fail(f"BUILD-METADATA commit {commit!r} does not match source run {expected_commit!r}")
    if public_files != str(EXPECTED_RELEASE_FILES):
        fail(f"BUILD-METADATA PUBLIC_RELEASE_FILES={public_files!r}; expected {EXPECTED_RELEASE_FILES}")

    local_digests = {path.name: sha256_file(path) for path in files}
    manifest = parse_manifest(bundle_dir / "SHA256.txt")
    expected_manifest_names = local_names - {"SHA256.txt"}
    if set(manifest) != expected_manifest_names:
        fail("SHA256.txt asset set does not match the release bundle")
    for name, digest in manifest.items():
        if local_digests[name] != digest:
            fail(f"SHA256.txt digest mismatch for {name}")

    try:
        release = json.loads(release_json_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        fail(f"unable to read GitHub Release metadata: {exc}")
    if not isinstance(release, dict):
        fail("GitHub Release metadata is not an object")
    if release.get("tag_name") != tag:
        fail(f"GitHub Release tag mismatch: {release.get('tag_name')!r} != {tag!r}")
    if release.get("target_commitish") != expected_commit:
        fail(
            "GitHub Release target commit mismatch: "
            f"{release.get('target_commitish')!r} != {expected_commit!r}"
        )
    if release.get("draft") is not False or release.get("prerelease") is not False:
        fail("GitHub Release must be a published non-prerelease release")

    assets = release.get("assets")
    if not isinstance(assets, list):
        fail("GitHub Release assets are unavailable")
    remote_digests: dict[str, str] = {}
    for asset in assets:
        if not isinstance(asset, dict):
            fail("GitHub Release contains invalid asset metadata")
        name = asset.get("name")
        digest = asset.get("digest")
        if not isinstance(name, str) or not name or name in remote_digests:
            fail(f"invalid or duplicate GitHub Release asset name: {name!r}")
        if not isinstance(digest, str) or not digest.startswith("sha256:"):
            fail(f"GitHub Release asset {name!r} has no SHA-256 digest")
        hex_digest = digest.removeprefix("sha256:")
        if not SHA256_RE.fullmatch(hex_digest):
            fail(f"GitHub Release asset {name!r} has malformed digest {digest!r}")
        remote_digests[name] = hex_digest

    if set(remote_digests) != local_names:
        fail("GitHub Release asset set does not match the exact source workflow bundle")
    for name, digest in local_digests.items():
        if remote_digests[name] != digest:
            fail(f"GitHub Release digest mismatch for {name}")

    immutable = "YES" if release.get("immutable") is True else "NO"
    return {
        "version": version,
        "tag": tag,
        "commit": commit,
        "immutable": immutable,
        "files": str(len(files)),
    }


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("bundle_dir", type=Path)
    parser.add_argument("release_json", type=Path)
    parser.add_argument("--expected-commit", required=True)
    args = parser.parse_args()

    result = verify_release(args.bundle_dir, args.release_json, args.expected_commit)
    print(f"RELEASE_ASSET_DIGEST_READBACK=PASS ({result['files']} files)")
    print(f"RELEASE_TAG={result['tag']}")
    print(f"RELEASE_COMMIT={result['commit']}")
    print(f"RELEASE_IMMUTABLE={result['immutable']}")


if __name__ == "__main__":
    main()
