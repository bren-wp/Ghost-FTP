#!/usr/bin/env python3
"""Fail-closed privacy guard for Ghost FTP Android and browser surfaces."""

from __future__ import annotations

import json
import re
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

ANDROID_MANIFEST = ROOT / "android/app/src/main/AndroidManifest.xml"
ANDROID_BUILD = ROOT / "android/app/build.gradle"
ANDROID_SOURCE_ROOT = ROOT / "android/app/src/main/java"
ANDROID_PROFILE_STORE = (
    ROOT / "android/app/src/main/java/app/ghostftp/client/SiteProfileStore.java"
)
ANDROID_NS = "{http://schemas.android.com/apk/res/android}"
ANDROID_ALLOWED_PERMISSIONS = {"android.permission.INTERNET"}

BROWSER_ROOT = ROOT / "ekstenzije"
BROWSER_TARGETS = ("chromium", "firefox")
BROWSER_ZERO_PRIVILEGE_KEYS = (
    "permissions",
    "optional_permissions",
    "host_permissions",
    "optional_host_permissions",
    "content_scripts",
)
BROWSER_FORBIDDEN_NETWORK_PATTERNS = {
    "fetch(": re.compile(r"\bfetch\s*\(", re.IGNORECASE),
    "XMLHttpRequest": re.compile(r"\bXMLHttpRequest\b"),
    "WebSocket": re.compile(r"\bWebSocket\s*\("),
    "EventSource": re.compile(r"\bEventSource\s*\("),
    "sendBeacon": re.compile(r"\bsendBeacon\s*\("),
}
BROWSER_FORBIDDEN_CODE_PATTERNS = {
    "eval": re.compile(r"\beval\s*\("),
    "Function constructor": re.compile(r"\bnew\s+Function\s*\("),
}
URL_RE = re.compile(r"https?://[^\s\"'`<>)]+", re.IGNORECASE)
DEPENDENCY_RE = re.compile(
    r"^\s*(?:implementation|api|runtimeOnly|compileOnly|kapt|ksp|annotationProcessor)\b",
    re.MULTILINE,
)
FORBIDDEN_VENDOR_MARKERS = {
    "sentry.io",
    "io.sentry",
    "google-analytics",
    "googletagmanager",
    "firebase-analytics",
    "com.google.firebase.analytics",
    "com.google.firebase.crashlytics",
    "segment.io",
    "com.segment.analytics",
    "mixpanel",
    "amplitude",
    "posthog",
    "datadog",
    "newrelic",
    "bugsnag",
    "crashlytics",
    "appcenter",
    "telemetrydeck",
}


def fail(message: str) -> None:
    raise SystemExit("CROSS_PLATFORM_PRIVACY_AUDIT_FAILED: " + message)


def read_text(path: Path) -> str:
    if not path.is_file():
        fail(f"missing {path.relative_to(ROOT)}")
    try:
        return path.read_text(encoding="utf-8")
    except UnicodeError as exc:
        fail(f"{path.relative_to(ROOT)} is not valid UTF-8: {exc}")


def reject_vendor_markers(path: Path, text: str) -> None:
    lower = text.lower()
    for marker in sorted(FORBIDDEN_VENDOR_MARKERS):
        if marker in lower:
            fail(
                f"telemetry/vendor marker {marker!r} found in "
                f"{path.relative_to(ROOT)}"
            )


def audit_android() -> None:
    manifest_text = read_text(ANDROID_MANIFEST)
    try:
        manifest = ET.fromstring(manifest_text)
    except ET.ParseError as exc:
        fail(f"invalid AndroidManifest.xml: {exc}")

    permissions: set[str] = set()
    for tag in ("uses-permission", "uses-permission-sdk-23"):
        for node in manifest.findall(tag):
            name = node.attrib.get(ANDROID_NS + "name", "").strip()
            if name:
                permissions.add(name)
    if permissions != ANDROID_ALLOWED_PERMISSIONS:
        fail(
            "Android permission surface changed; expected exactly "
            f"{sorted(ANDROID_ALLOWED_PERMISSIONS)}, got {sorted(permissions)}"
        )

    application = manifest.find("application")
    if application is None:
        fail("Android manifest is missing <application>")
    if application.attrib.get(ANDROID_NS + "allowBackup") != "false":
        fail("Android allowBackup must remain false")

    build_text = read_text(ANDROID_BUILD)
    if DEPENDENCY_RE.search(build_text):
        fail(
            "Android app gained an explicit dependency; local-first dependency "
            "changes require privacy/security review"
        )
    reject_vendor_markers(ANDROID_BUILD, build_text)

    source_files = sorted(
        path
        for suffix in ("*.java", "*.kt")
        for path in ANDROID_SOURCE_ROOT.rglob(suffix)
        if path.is_file()
    )
    if not source_files:
        fail("Android production source was not found")

    for path in source_files:
        text = read_text(path)
        reject_vendor_markers(path, text)
        urls = sorted(set(URL_RE.findall(text)))
        if urls:
            fail(
                f"fixed HTTP(S) URL found in Android runtime source "
                f"{path.relative_to(ROOT)}: {urls[0]}"
            )

    profile_store = read_text(ANDROID_PROFILE_STORE)
    for secret_key in (
        '"password"',
        '"passphrase"',
        '"privateKey"',
        '"private_key"',
        '"credential"',
        '"secret"',
    ):
        if secret_key.lower() in profile_store.lower():
            fail(
                "Android SharedPreferences profile storage must not persist "
                f"credential material: {secret_key}"
            )


def audit_browser_extensions() -> None:
    for target in BROWSER_TARGETS:
        root = BROWSER_ROOT / target
        manifest_path = root / "manifest.json"
        manifest_text = read_text(manifest_path)
        try:
            manifest = json.loads(manifest_text)
        except json.JSONDecodeError as exc:
            fail(f"invalid {manifest_path.relative_to(ROOT)}: {exc}")

        if manifest.get("manifest_version") != 3:
            fail(f"{manifest_path.relative_to(ROOT)} must remain Manifest V3")

        for key in BROWSER_ZERO_PRIVILEGE_KEYS:
            value = manifest.get(key, [])
            if value not in (None, [], {}):
                fail(
                    f"{manifest_path.relative_to(ROOT)} gained {key}; "
                    "browser privilege expansion requires privacy/security review"
                )
        if manifest.get("externally_connectable") not in (None, [], {}):
            fail(
                f"{manifest_path.relative_to(ROOT)} gained externally_connectable"
            )

        source_files = sorted(
            path
            for pattern in ("*.js", "*.html", "*.css")
            for path in root.rglob(pattern)
            if path.is_file()
        )
        if not source_files:
            fail(f"browser runtime source was not found for {target}")

        for path in source_files:
            text = read_text(path)
            reject_vendor_markers(path, text)
            urls = sorted(set(URL_RE.findall(text)))
            if urls:
                fail(
                    f"fixed HTTP(S) URL found in browser runtime source "
                    f"{path.relative_to(ROOT)}: {urls[0]}"
                )
            if path.suffix.lower() == ".js":
                for name, pattern in BROWSER_FORBIDDEN_NETWORK_PATTERNS.items():
                    if pattern.search(text):
                        fail(
                            f"browser network API {name!r} found in "
                            f"{path.relative_to(ROOT)}"
                        )
                for name, pattern in BROWSER_FORBIDDEN_CODE_PATTERNS.items():
                    if pattern.search(text):
                        fail(
                            f"dynamic code execution {name!r} found in "
                            f"{path.relative_to(ROOT)}"
                        )


def main() -> None:
    audit_android()
    audit_browser_extensions()
    print("CROSS_PLATFORM_PRIVACY_AUDIT=PASS")
    print("ANDROID_PERMISSION_SURFACE=INTERNET_ONLY")
    print("ANDROID_EXTERNAL_APP_DEPENDENCIES=BLOCKED")
    print("ANDROID_FIXED_HTTP_URLS=BLOCKED")
    print("ANDROID_PROFILE_SECRET_PERSISTENCE=BLOCKED")
    print("BROWSER_EXTENSION_PERMISSIONS=ZERO")
    print("BROWSER_EXTENSION_FIXED_HTTP_URLS=BLOCKED")
    print("BROWSER_EXTENSION_NETWORK_APIS=BLOCKED")
    print("BROWSER_REMOTE_CODE_EXECUTION=BLOCKED")
    print("CROSS_PLATFORM_TELEMETRY_VENDOR_MARKERS=BLOCKED")


if __name__ == "__main__":
    main()
