#!/usr/bin/env python3
"""Block production changes to an already published version unless VERSION changes."""

from __future__ import annotations

import argparse
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

# Files that change runtime behavior, installation packages, or public release
# content. CI/audit scripts and workflows are intentionally excluded so their
# safety checks can be strengthened without inventing a new product version.
PRODUCTION_PREFIXES = (
    "cmd/",
    "internal/",
    "build/",
    "docs/",
)
PRODUCTION_FILES = {
    "README.md",
    "CHANGELOG.md",
    "LICENSE",
    "go.mod",
    "BUILD-WINDOWS.cmd",
    "BUILD-WINDOWS.ps1",
    "scripts/BUILD-LOCAL.sh",
    "scripts/BUILD-LINUX.sh",
    "scripts/BUILD-MACOS.sh",
    "scripts/release_notes.py",
    "scripts/publish_release.ps1",
}


def git(*args: str, root: Path = ROOT, check: bool = True) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["git", *args],
        cwd=root,
        check=check,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )


def is_production_path(path: str) -> bool:
    normalized = path.replace("\\", "/").lstrip("./")
    return normalized in PRODUCTION_FILES or normalized.startswith(PRODUCTION_PREFIXES)


def tag_commit(version: str, root: Path = ROOT) -> str | None:
    result = git("rev-parse", "-q", "--verify", f"refs/tags/v{version}^{{commit}}", root=root, check=False)
    if result.returncode != 0:
        return None
    value = result.stdout.strip()
    return value or None


def changed_paths(base: str, head: str, root: Path = ROOT) -> list[str]:
    result = git("diff", "--name-only", "--diff-filter=ACMRTUXB", base, head, root=root)
    return [line.strip() for line in result.stdout.splitlines() if line.strip()]


def validate_release_version(base: str, head: str, root: Path = ROOT) -> tuple[bool, str]:
    version_path = root / "VERSION"
    if not version_path.is_file():
        return False, "VERSION is missing"
    version = version_path.read_text(encoding="utf-8").strip()
    if not version:
        return False, "VERSION is empty"

    released = tag_commit(version, root)
    if released is None:
        return True, f"v{version} has not been tagged yet; production changes are allowed before the first release"

    if not base or not head or set(base) == {"0"}:
        return True, "no reliable event base SHA is available; the version guard is not applied to this manual/initial run"

    try:
        paths = changed_paths(base, head, root)
    except subprocess.CalledProcessError as exc:
        return False, f"unable to calculate diff {base}..{head}: {exc.stderr.strip()}"

    if "VERSION" in paths:
        return True, "VERSION changed in this change set"

    production = sorted(path for path in paths if is_production_path(path))
    if not production:
        return True, "changes do not modify runtime, packages, or public release content"

    preview = ", ".join(production[:12])
    if len(production) > 12:
        preview += f", ... (+{len(production) - 12})"
    return False, (
        f"v{version} already exists at {released[:12]}, but this change set modifies production content without changing VERSION: "
        f"{preview}. Published versions are immutable; bump VERSION before merging new production changes."
    )


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", default="")
    parser.add_argument("--head", default="HEAD")
    args = parser.parse_args()

    ok, message = validate_release_version(args.base.strip(), args.head.strip())
    if not ok:
        print("RELEASE_VERSION_GUARD=FAILED", file=sys.stderr)
        print(message, file=sys.stderr)
        return 1
    print("RELEASE_VERSION_GUARD=PASS")
    print(message)
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
