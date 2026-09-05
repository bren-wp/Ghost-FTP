#!/usr/bin/env python3
"""Audit Ghost FTP dependency provenance and forbidden tracking surfaces."""

from __future__ import annotations

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

ALLOWED_WEB_PLATFORM_REQUIREMENTS = {"php": ">=8.1"}
ALLOWED_WEB_SUGGESTED_EXTENSIONS = {
    "ext-ftp": "Required for FTP/FTPS support",
    "ext-ssh2": "Required for SFTP support",
    "ext-zip": "Required for ZIP create/extract and folder downloads",
    "ext-sodium": "Preferred for credential encryption",
    "ext-openssl": "Encryption fallback when sodium is unavailable",
}

FORBIDDEN_TRACKING = (
    "firebase-analytics",
    "firebase-crashlytics",
    "appcenter",
    "sentry",
    "bugsnag",
    "newrelic",
    "datadog",
    "amplitude",
    "mixpanel",
    "segment-analytics",
    "facebook-sdk",
    "google-mobile-ads",
    "telemetry endpoint",
)


def fail(message: str) -> None:
    raise SystemExit("DEPENDENCY_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def audit_go() -> None:
    gomod = read("go.mod")
    module_lines = [line.strip() for line in gomod.splitlines() if line.strip()]
    if not module_lines or module_lines[0] != "module github.com/bren-wp/Ghost-FTP":
        fail("unexpected Go module identity")
    if any(re.match(r"^(require|replace|exclude|retract)\b", line) for line in module_lines):
        fail("desktop/core Go module must not introduce external module requirements or replacements")
    if (ROOT / "go.sum").exists() or (ROOT / "vendor").exists():
        fail("desktop/core must remain standard-library-only; go.sum/vendor is unexpected")


def audit_web_companion() -> None:
    composer = ROOT / "GhostFTP WEB/composer.json"
    if not composer.is_file():
        return
    try:
        data = json.loads(composer.read_text(encoding="utf-8"))
    except json.JSONDecodeError as exc:
        fail(f"Web companion composer.json is invalid JSON: {exc}")

    require = data.get("require")
    if require != ALLOWED_WEB_PLATFORM_REQUIREMENTS:
        fail(
            "Web companion Composer require must contain only the PHP platform requirement "
            f"{ALLOWED_WEB_PLATFORM_REQUIREMENTS!r}; found={require!r}"
        )
    third_party = [name for name in require if name != "php" and not str(name).startswith("ext-")]
    if third_party:
        fail("Web companion third-party Composer packages are forbidden: " + ", ".join(sorted(third_party)))

    suggest = data.get("suggest")
    if suggest != ALLOWED_WEB_SUGGESTED_EXTENSIONS:
        fail("Web companion suggested-extension capability list drifted")


def audit_transport_contract() -> None:
    remote_root = ROOT / "internal/remote"
    source = "\n".join(
        p.read_text(encoding="utf-8", errors="strict")
        for p in sorted(remote_root.glob("*.go"))
        if p.is_file()
    ).lower()
    for executable in ("curl", "ssh", "sftp"):
        if executable not in source:
            fail(f"documented OS transport executable is no longer visible in internal/remote: {executable}")

    dependencies = read("docs/DEPENDENCIES.md").lower()
    for executable in ("curl", "ssh", "sftp"):
        if f"`{executable}`" not in dependencies:
            fail(f"docs/DEPENDENCIES.md must disclose OS transport executable: {executable}")


def audit_tracking_markers() -> None:
    candidates = [ROOT / "go.mod", ROOT / "GhostFTP WEB/composer.json"]
    combined = "\n".join(
        path.read_text(encoding="utf-8", errors="strict").lower()
        for path in candidates
        if path.is_file()
    )
    for marker in FORBIDDEN_TRACKING:
        if marker in combined:
            fail(f"forbidden tracking/advertising marker present in dependency surfaces: {marker}")


def main() -> int:
    audit_go()
    audit_web_companion()
    audit_transport_contract()
    audit_tracking_markers()
    print("DEPENDENCY_AUDIT=PASS")
    print("DESKTOP_GO_EXTERNAL_MODULES=0")
    print("DESKTOP_BUNDLED_THIRD_PARTY_LIBRARIES=0")
    print("DESKTOP_OS_TRANSPORT_TOOLS=curl,ssh,sftp")
    print("WEB_COMPANION_THIRD_PARTY_COMPOSER_PACKAGES=0")
    print("TRACKING_ADS_CRASH_SDKS=FORBIDDEN")
    return 0


if __name__ == "__main__":
    sys.exit(main())
