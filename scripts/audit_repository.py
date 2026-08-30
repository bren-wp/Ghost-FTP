#!/usr/bin/env python3
"""Audit every tracked repository path and file for portable release hygiene."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
CURRENT_RELEASE_RE = re.compile(
    r"(?im)\b(?:Current release|Trenutačno izdanje)\s*:\s*(\d+\.\d+\.\d+)\b"
)
MERGE_CONFLICT_RE = re.compile(r"(?m)^(?:<<<<<<< .+|=======|>>>>>>> .+)$")
BINARY_EXTENSIONS = {
    ".a", ".apk", ".deb", ".dll", ".dylib", ".exe", ".gif", ".ico", ".icns",
    ".ipa", ".jar", ".jpeg", ".jpg", ".otf", ".pdf", ".pkg", ".png", ".so",
    ".ttf", ".webp", ".woff", ".woff2", ".zip",
}
FORBIDDEN_TRACKED_PREFIXES = (
    ".gradle/",
    ".idea/",
    ".vscode/",
    "android/.gradle/",
    "android/app/build/",
    "coverage/",
    "dist/",
    "ios/build/",
    "linux/out/",
    "macos/out/",
    "tmp/",
)
FORBIDDEN_BASENAMES = {".DS_Store", "Thumbs.db", "desktop.ini"}
WINDOWS_RESERVED = {
    "CON", "PRN", "AUX", "NUL",
    *(f"COM{i}" for i in range(1, 10)),
    *(f"LPT{i}" for i in range(1, 10)),
}


def fail(errors: list[str]) -> None:
    for error in errors:
        print(f"REPOSITORY_AUDIT_ERROR: {error}", file=sys.stderr)
    raise SystemExit(1)


def tracked_entries(root: Path = ROOT) -> list[tuple[str, str]]:
    try:
        raw = subprocess.check_output(
            ["git", "ls-files", "-s", "-z"], cwd=root, stderr=subprocess.STDOUT
        )
    except (OSError, subprocess.CalledProcessError) as exc:
        raise RuntimeError(f"git ls-files failed: {exc}") from exc

    entries: list[tuple[str, str]] = []
    for record in raw.split(b"\0"):
        if not record:
            continue
        try:
            meta, path_bytes = record.split(b"\t", 1)
            mode = meta.split(b" ", 1)[0].decode("ascii")
            path = path_bytes.decode("utf-8", "strict")
        except (ValueError, UnicodeDecodeError) as exc:
            raise RuntimeError("tracked path metadata is not canonical UTF-8") from exc
        entries.append((mode, path))
    return entries


def validate_path(path: str, mode: str, seen_casefold: dict[str, str]) -> list[str]:
    errors: list[str] = []
    if not path or path.startswith("/") or "\\" in path:
        errors.append(f"noncanonical tracked path: {path!r}")
        return errors
    if any(ord(ch) < 32 or ord(ch) == 127 for ch in path):
        errors.append(f"tracked path contains a control character: {path!r}")
    if len(path.encode("utf-8")) > 240:
        errors.append(f"tracked path exceeds 240 UTF-8 bytes: {path}")
    if mode == "120000":
        errors.append(f"tracked symlink is not allowed in release source: {path}")

    normalized = path.replace("\\", "/")
    for prefix in FORBIDDEN_TRACKED_PREFIXES:
        if normalized.startswith(prefix):
            errors.append(f"generated/cache path is tracked: {path}")
            break

    parts = normalized.split("/")
    for part in parts:
        if part in ("", ".", ".."):
            errors.append(f"tracked path contains an unsafe component: {path}")
            break
        if part.endswith((" ", ".")):
            errors.append(f"tracked path is not Windows-portable: {path}")
            break
        stem = part.split(".", 1)[0].upper()
        if stem in WINDOWS_RESERVED:
            errors.append(f"tracked path uses a Windows-reserved name: {path}")
            break

    if parts[-1] in FORBIDDEN_BASENAMES:
        errors.append(f"OS/editor artifact is tracked: {path}")

    folded = normalized.casefold()
    previous = seen_casefold.get(folded)
    if previous is not None and previous != normalized:
        errors.append(f"case-insensitive path collision: {previous} <-> {normalized}")
    else:
        seen_casefold[folded] = normalized
    return errors


def validate_text(path: str, data: bytes, version: str) -> list[str]:
    errors: list[str] = []
    if data.startswith(b"\xef\xbb\xbf"):
        errors.append(f"UTF-8 BOM is not allowed: {path}")
    try:
        text = data.decode("utf-8", "strict")
    except UnicodeDecodeError:
        return [f"non-binary tracked file is not valid UTF-8: {path}"]

    for line_no, line in enumerate(text.splitlines(), 1):
        if line.endswith((" ", "\t")):
            errors.append(f"trailing whitespace: {path}:{line_no}")
            if len(errors) >= 20:
                break

    if text and not text.endswith("\n"):
        errors.append(f"text file is missing final newline: {path}")

    if MERGE_CONFLICT_RE.search(text):
        errors.append(f"merge-conflict marker remains in {path}")

    for match in CURRENT_RELEASE_RE.finditer(text):
        if match.group(1) != version:
            errors.append(
                f"stale current-release reference in {path}: {match.group(1)} != {version}"
            )
    return errors


def main() -> int:
    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not VERSION_RE.fullmatch(version):
        fail([f"VERSION is not semantic: {version!r}"])

    required = {
        "VERSION", "README.md", "CHANGELOG.md", "LICENSE",
        ".github/workflows/ci.yml", ".github/workflows/release.yml",
    }
    errors: list[str] = []
    seen_casefold: dict[str, str] = {}
    tracked = tracked_entries()
    tracked_paths = {path for _, path in tracked}
    missing = sorted(required - tracked_paths)
    if missing:
        errors.append("required tracked files missing: " + ", ".join(missing))

    text_count = 0
    binary_count = 0
    for mode, path in tracked:
        errors.extend(validate_path(path, mode, seen_casefold))
        absolute = ROOT / path
        if not absolute.is_file():
            errors.append(f"tracked file is missing from checkout: {path}")
            continue
        data = absolute.read_bytes()
        suffix = absolute.suffix.lower()
        if suffix in BINARY_EXTENSIONS:
            binary_count += 1
            continue
        if b"\x00" in data:
            errors.append(f"unexpected NUL/binary content in non-binary tracked file: {path}")
            continue
        text_count += 1
        errors.extend(validate_text(path, data, version))

    if errors:
        fail(errors[:200])

    print(f"REPOSITORY_AUDIT=PASS ({version})")
    print(f"REPOSITORY_TRACKED_FILES={len(tracked)}")
    print(f"REPOSITORY_TEXT_FILES={text_count}")
    print(f"REPOSITORY_BINARY_FILES={binary_count}")
    print("REPOSITORY_PATH_COLLISIONS=BLOCKED")
    print("REPOSITORY_GENERATED_ARTIFACTS=BLOCKED")
    print("REPOSITORY_TEXT_UTF8_AND_WHITESPACE=ENFORCED")
    print("REPOSITORY_CURRENT_RELEASE_DRIFT=BLOCKED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
