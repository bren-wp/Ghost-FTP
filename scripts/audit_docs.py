#!/usr/bin/env python3
"""Validate active Ghost FTP documentation after the macOS retirement."""
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = (
    "README.md",
    "docs/README.md",
    "docs/REFERENCE-UI.md",
    "docs/PLATFORM-PARITY.md",
    "docs/INSTALLATION.md",
    "docs/SECURITY.md",
    "docs/PRIVACY.md",
    "docs/PACKAGES.md",
    "docs/RELEASE-VERIFICATION.md",
    "docs/TESTING.md",
)

def fail(message: str) -> None:
    raise SystemExit("DOCS_AUDIT_FAILED: " + message)

def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required document: {rel}")
    return path.read_text(encoding="utf-8")

def main() -> int:
    for rel in REQUIRED:
        read(rel)

    readme = read("README.md")
    for marker in ("Windows", "Linux", "Android",
                   "docs/images/0.0.8/ghost-ftp-main-workspace.png",
                   "docs/images/0.0.8/ghost-ftp-linux-main-workspace.png",
                   "docs/images/0.0.8/ghost-ftp-android-files.png"):
        if marker not in readme:
            fail(f"README missing {marker}")

    for rel in ("docs/PACKAGES.md", "docs/PLATFORM-PARITY.md", "docs/RELEASE-VERIFICATION.md"):
        text = read(rel)
        if "ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID,MACOS" in text:
            fail(f"{rel} still declares macOS active")

    if (ROOT / "macos").exists():
        fail("retired macos/ source tree still exists")
    if (ROOT / ".github/workflows/macos-app.yml").exists() or (ROOT / ".github/workflows/macos-production.yml").exists():
        fail("retired macOS workflow still exists")

    print("DOCS_AUDIT=PASS")
    print("ACTIVE_SOURCE_PLATFORMS=WINDOWS,LINUX,ANDROID")
    return 0

if __name__ == "__main__":
    sys.exit(main())
