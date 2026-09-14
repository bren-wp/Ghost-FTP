#!/usr/bin/env python3
"""Build deterministic official Ghost FTP browser extension packages."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
import sys
import zipfile

ROOT = Path(__file__).resolve().parents[1]
EXT_ROOT = ROOT / "extensions"
SHARED = EXT_ROOT / "shared"

OFFICIAL_PRODUCT = "Ghost FTP"
OFFICIAL_EXTENSION = "Ghost FTP Connection Helper"
OFFICIAL_SHORT_NAME = "Ghost FTP"
OFFICIAL_HOMEPAGE = "https://ghostftp.com"
OFFICIAL_FIREFOX_ID = "ghostftp-connection-helper@ghostftp.com"
PACKAGES = ("chrome", "edge", "firefox", "opera")
RUNTIME_FILES = ("core.js", "popup.js", "popup.css", "popup.html", "icons/icon.png")
ZIP_TIMESTAMP = (1980, 1, 1, 0, 0, 0)
FORBIDDEN_MANIFEST_KEYS = {
    "background",
    "content_scripts",
    "externally_connectable",
    "host_permissions",
    "optional_host_permissions",
    "optional_permissions",
    "web_accessible_resources",
}
FORBIDDEN_RUNTIME_MARKERS = (
    "fetch(",
    "XMLHttpRequest",
    "WebSocket",
    "sendBeacon",
    "localStorage",
    "sessionStorage",
    "indexedDB",
    "analytics",
    "telemetry",
)


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8")


def load_json(path: Path) -> dict:
    value = json.loads(read_text(path))
    if not isinstance(value, dict):
        raise ValueError(f"expected JSON object: {path.relative_to(ROOT)}")
    return value


def manifest_path(package: str) -> Path:
    return EXT_ROOT / package / "manifest.json"


def validate_brand() -> None:
    brand = load_json(EXT_ROOT / "BRAND.json")
    expected = {
        "product": OFFICIAL_PRODUCT,
        "extension": OFFICIAL_EXTENSION,
        "short_name": OFFICIAL_SHORT_NAME,
        "technical_id": "ghostftp-connection-helper",
        "homepage": OFFICIAL_HOMEPAGE,
        "firefox_id": OFFICIAL_FIREFOX_ID,
        "official_packages": list(PACKAGES),
    }
    if brand != expected:
        raise ValueError("extensions/BRAND.json does not match the official Ghost FTP brand contract")


def validate_manifest(package: str, manifest: dict, version: str) -> None:
    if manifest.get("manifest_version") != 3:
        raise ValueError(f"{package}: Manifest V3 is required")
    if manifest.get("name") != OFFICIAL_EXTENSION:
        raise ValueError(f"{package}: official extension name changed")
    if manifest.get("short_name") != OFFICIAL_SHORT_NAME:
        raise ValueError(f"{package}: official short name changed")
    if manifest.get("version") != version:
        raise ValueError(f"{package}: manifest version does not match VERSION")
    if manifest.get("homepage_url") != OFFICIAL_HOMEPAGE:
        raise ValueError(f"{package}: official homepage changed")
    if manifest.get("permissions", []) != []:
        raise ValueError(f"{package}: extension must request zero browser permissions")
    forbidden = FORBIDDEN_MANIFEST_KEYS.intersection(manifest)
    if forbidden:
        raise ValueError(f"{package}: forbidden manifest capabilities: {sorted(forbidden)}")

    action = manifest.get("action")
    if not isinstance(action, dict):
        raise ValueError(f"{package}: missing action")
    if action.get("default_title") != OFFICIAL_EXTENSION or action.get("default_popup") != "popup.html":
        raise ValueError(f"{package}: action branding/popup contract changed")

    firefox_settings = manifest.get("browser_specific_settings")
    if package == "firefox":
        expected = {
            "gecko": {
                "id": OFFICIAL_FIREFOX_ID,
                "data_collection_permissions": {"required": ["none"]},
            }
        }
        if firefox_settings != expected:
            raise ValueError("firefox: signing ID or data-collection declaration changed")
    elif firefox_settings is not None:
        raise ValueError(f"{package}: Firefox-only browser_specific_settings leaked into Chromium package")


def validate_runtime() -> None:
    for relative in RUNTIME_FILES:
        path = SHARED / relative
        if not path.is_file() or path.stat().st_size <= 0:
            raise ValueError(f"missing shared extension runtime file: {relative}")

    icon = (SHARED / "icons/icon.png").read_bytes()
    if not icon.startswith(b"\x89PNG\r\n\x1a\n"):
        raise ValueError("shared extension icon is not a PNG")

    html = read_text(SHARED / "popup.html")
    if OFFICIAL_PRODUCT not in html or "Connection Helper" not in html:
        raise ValueError("popup Ghost FTP branding changed")
    if "<style" in html.lower():
        raise ValueError("popup must not contain inline styles")
    if '<script src="core.js"></script>' not in html or '<script src="popup.js"></script>' not in html:
        raise ValueError("popup must load only the packaged runtime scripts")
    if "http://" in html or "https://" in html:
        raise ValueError("popup HTML must not reference remote resources")

    runtime = read_text(SHARED / "core.js") + "\n" + read_text(SHARED / "popup.js")
    for marker in FORBIDDEN_RUNTIME_MARKERS:
        if marker in runtime:
            raise ValueError(f"extension runtime contains forbidden capability marker: {marker}")
    for scheme in ("'ftp:'", "'ftps:'", "'sftp:'"):
        if scheme not in runtime:
            raise ValueError(f"connection parser is missing {scheme}")
    for required in ("MAX_INPUT_LENGTH", "CONTROL_CHARACTERS", "passwordDetected", "safeTarget", "new URL("):
        if required not in runtime:
            raise ValueError(f"connection parser is missing safety marker: {required}")


def zip_info(name: str) -> zipfile.ZipInfo:
    info = zipfile.ZipInfo(name, ZIP_TIMESTAMP)
    info.compress_type = zipfile.ZIP_DEFLATED
    info.create_system = 3
    info.external_attr = 0o100644 << 16
    return info


def write_package(package: str, output_dir: Path, version: str) -> Path:
    manifest = load_json(manifest_path(package))
    validate_manifest(package, manifest, version)

    destination = output_dir / f"Ghost-FTP-{version}-{package.capitalize()}-Extension.zip"
    with zipfile.ZipFile(destination, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        manifest_bytes = (json.dumps(manifest, indent=2, ensure_ascii=False) + "\n").encode("utf-8")
        archive.writestr(zip_info("manifest.json"), manifest_bytes)
        for relative in RUNTIME_FILES:
            archive.writestr(zip_info(relative), (SHARED / relative).read_bytes())

    expected_names = ["manifest.json", *RUNTIME_FILES]
    with zipfile.ZipFile(destination, "r") as archive:
        if archive.namelist() != expected_names:
            raise ValueError(f"{package}: package contains an unexpected file set")
        bad = archive.testzip()
        if bad is not None:
            raise ValueError(f"{package}: corrupt ZIP entry: {bad}")
    return destination


def build(output_dir: Path) -> list[Path]:
    version = read_text(ROOT / "VERSION").strip()
    if not version or any(not part.isdigit() for part in version.split(".")) or len(version.split(".")) != 3:
        raise ValueError("VERSION must be a numeric semantic version")

    validate_brand()
    validate_runtime()
    for package in PACKAGES:
        validate_manifest(package, load_json(manifest_path(package)), version)

    output_dir.mkdir(parents=True, exist_ok=True)
    for stale in output_dir.glob("Ghost-FTP-*-Extension.zip"):
        stale.unlink()
    return [write_package(package, output_dir, version) for package in PACKAGES]


def main() -> int:
    parser = argparse.ArgumentParser(description="Build official Ghost FTP browser extensions")
    parser.add_argument("--output", type=Path, default=ROOT / "dist" / "browser")
    parser.add_argument("--check", action="store_true", help="validate source contract without writing packages")
    args = parser.parse_args()

    try:
        if args.check:
            version = read_text(ROOT / "VERSION").strip()
            validate_brand()
            validate_runtime()
            for package in PACKAGES:
                validate_manifest(package, load_json(manifest_path(package)), version)
            outputs: list[Path] = []
        else:
            outputs = build(args.output)
    except (OSError, UnicodeError, json.JSONDecodeError, ValueError, zipfile.BadZipFile) as exc:
        print(f"BROWSER_EXTENSION_BUILD=FAILED: {exc}", file=sys.stderr)
        return 1

    print("BROWSER_EXTENSION_BUILD=PASS")
    print(f"BROWSER_EXTENSION_BRAND={OFFICIAL_PRODUCT}")
    print("BROWSER_EXTENSION_PACKAGES=chrome,edge,firefox,opera")
    for path in outputs:
        print(f"BROWSER_EXTENSION_ARTIFACT={path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
