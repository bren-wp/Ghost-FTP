#!/usr/bin/env python3
"""Audit Ghost FTP desktop dependency provenance and tracking surfaces."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
FORBIDDEN_TRACKING = (
    "firebase-analytics",
    "firebase-crashlytics",
    "appcenter",
    "sentry",
    "getsentry",
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
    lines = [line.strip() for line in gomod.splitlines() if line.strip()]
    if not lines or lines[0] != "module github.com/bren-wp/Ghost-FTP":
        fail("unexpected Go module identity")
    if any(re.match(r"^(require|replace|exclude|retract)\b", line) for line in lines):
        fail("desktop/core Go module must remain free of external module requirements")
    if (ROOT / "go.sum").exists() or (ROOT / "vendor").exists():
        fail("desktop/core must remain standard-library-only at the Go module level")


def audit_transport_contract() -> None:
    source = "\n".join(
        path.read_text(encoding="utf-8", errors="strict")
        for path in sorted((ROOT / "internal/remote").glob("*.go"))
        if path.is_file()
    ).lower()
    docs = read("docs/DEPENDENCIES.md").lower()
    for executable in ("curl", "ssh", "sftp"):
        if executable not in source:
            fail(f"OS transport executable is no longer visible in internal/remote: {executable}")
        if f"`{executable}`" not in docs:
            fail(f"docs/DEPENDENCIES.md must disclose OS transport executable: {executable}")


def marker_pattern(marker: str) -> re.Pattern[str]:
    # Detect SDK/service identifiers as tokens rather than arbitrary substrings.
    # This still matches sentry, sentry-go, sentry.io, /sentry/ and getsentry,
    # while avoiding false positives from unrelated identifiers ending in
    # "sEntry" after case normalization.
    return re.compile(r"(?<![a-z0-9])" + re.escape(marker) + r"(?![a-z0-9])", re.IGNORECASE)


def desktop_source_files() -> list[Path]:
    files = [ROOT / "go.mod"]
    for surface in (ROOT / "cmd", ROOT / "internal"):
        if surface.is_dir():
            files.extend(sorted(path for path in surface.rglob("*.go") if path.is_file()))
    return files


def audit_tracking_markers() -> None:
    patterns = [(marker, marker_pattern(marker)) for marker in FORBIDDEN_TRACKING]
    for path in desktop_source_files():
        text = path.read_text(encoding="utf-8", errors="strict")
        for marker, pattern in patterns:
            match = pattern.search(text)
            if match is not None:
                rel = path.relative_to(ROOT).as_posix()
                line = text.count("\n", 0, match.start()) + 1
                fail(f"forbidden tracking/advertising marker {marker!r} present in {rel}:{line}")


def main() -> int:
    audit_go()
    audit_transport_contract()
    audit_tracking_markers()
    print("DEPENDENCY_AUDIT=PASS")
    print("DESKTOP_GO_EXTERNAL_MODULES=0")
    print("DESKTOP_BUNDLED_THIRD_PARTY_LIBRARIES=0")
    print("DESKTOP_OS_TRANSPORT_TOOLS=curl,ssh,sftp")
    print("TRACKING_ADS_CRASH_SDKS=FORBIDDEN")
    return 0


if __name__ == "__main__":
    sys.exit(main())
