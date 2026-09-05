#!/usr/bin/env python3
"""Audit Ghost FTP dependency provenance and forbidden tracking surfaces."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

ALLOWED_ANDROID = {
    "implementation": {
        "commons-net:commons-net:3.13.0",
        "com.hierynomus:sshj:0.40.0",
    },
    "testImplementation": {
        "junit:junit:4.13.2",
    },
}

FORBIDDEN_TRACKING = (
    "firebase-analytics",
    "firebase-crashlytics",
    "appcenter",
    "sentry-android",
    "bugsnag",
    "newrelic",
    "datadog",
    "amplitude",
    "mixpanel",
    "segment-analytics",
    "facebook-android-sdk",
    "play-services-ads",
    "google-mobile-ads",
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


def audit_android() -> None:
    gradle = read("android/app/build.gradle.kts")
    found: dict[str, set[str]] = {key: set() for key in ALLOWED_ANDROID}
    pattern = re.compile(r'^\s*(implementation|testImplementation)\("([^\"]+)"\)', re.MULTILINE)
    for kind, coordinate in pattern.findall(gradle):
        found.setdefault(kind, set()).add(coordinate)
        if "+" in coordinate or "SNAPSHOT" in coordinate.upper():
            fail(f"dynamic or snapshot Android dependency is forbidden: {coordinate}")
    if found != ALLOWED_ANDROID:
        fail(f"Android dependency allowlist drift: found={found!r} expected={ALLOWED_ANDROID!r}")

    lower = gradle.lower()
    for marker in FORBIDDEN_TRACKING:
        if marker in lower:
            fail(f"tracking/advertising SDK is forbidden: {marker}")


def audit_web() -> None:
    composer = read("GhostFTP WEB/composer.json")
    if '"require": {}' not in composer.replace("\n", " ").replace("  ", " "):
        # Parse without importing third-party tooling; exact empty require is the
        # contract for the shared-hosting package.
        import json
        data = json.loads(composer)
        if data.get("require") not in ({}, None):
            fail("Web/PWA Composer runtime dependencies must remain empty")


def audit_repository_tracking_markers() -> None:
    candidates = (
        "android/app/build.gradle.kts",
        "go.mod",
        "GhostFTP WEB/composer.json",
    )
    combined = "\n".join(read(rel).lower() for rel in candidates)
    for marker in FORBIDDEN_TRACKING:
        if marker in combined:
            fail(f"forbidden tracking marker present in dependency surfaces: {marker}")


def main() -> int:
    audit_go()
    audit_android()
    audit_web()
    audit_repository_tracking_markers()
    print("DEPENDENCY_AUDIT=PASS")
    print("GO_EXTERNAL_MODULES=0")
    print("ANDROID_RUNTIME_DEPENDENCIES=2_PINNED")
    print("WEB_COMPOSER_RUNTIME_DEPENDENCIES=0")
    print("TRACKING_ADS_CRASH_SDKS=FORBIDDEN")
    return 0


if __name__ == "__main__":
    sys.exit(main())
