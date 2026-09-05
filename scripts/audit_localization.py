#!/usr/bin/env python3
"""Validate Ghost FTP's English-first localization contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SUPPORTED = (
    "en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja",
    "it", "pl", "nl", "cs", "uk", "sv", "ro", "hu", "da", "fi", "no", "ko",
)
IOS_LOCALE = {"zh": "zh-Hans", "no": "nb"}
IOS_CORE_MIN_KEYS = 12
IOS_CORE_MIN_TRANSLATED = 8
STRINGS_LINE = re.compile(r'^\s*"((?:\\.|[^"\\])*)"\s*=\s*"((?:\\.|[^"\\])*)";\s*$')


def fail(message: str) -> None:
    raise SystemExit("LOCALIZATION_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def parse_apple_strings(rel: str) -> dict[str, str]:
    text = read(rel)
    result: dict[str, str] = {}
    for line_number, line in enumerate(text.splitlines(), 1):
        stripped = line.strip()
        if not stripped or stripped.startswith("/*") or stripped.startswith("*") or stripped.endswith("*/"):
            continue
        match = STRINGS_LINE.fullmatch(line)
        if match is None:
            fail(f"invalid Apple strings syntax in {rel}:{line_number}")
        key, value = match.groups()
        if key in result:
            fail(f"duplicate Apple localization key {key!r} in {rel}")
        result[key] = value
    return result


def validate_ios_localization() -> None:
    english_rel = "ios/GhostFTP/en.lproj/Localizable.strings"
    english = parse_apple_strings(english_rel)
    if len(english) < IOS_CORE_MIN_KEYS:
        fail(f"iOS English core catalog has {len(english)} keys; expected at least {IOS_CORE_MIN_KEYS}")
    for key, value in english.items():
        if not key.strip() or not value.strip():
            fail(f"empty iOS English key/value: {key!r}")

    for code in SUPPORTED:
        apple_code = IOS_LOCALE.get(code, code)
        rel = f"ios/GhostFTP/{apple_code}.lproj/Localizable.strings"
        catalog = parse_apple_strings(rel)
        if set(catalog) != set(english):
            missing = sorted(set(english) - set(catalog))
            extra = sorted(set(catalog) - set(english))
            fail(f"iOS locale {code} key drift; missing={missing}, extra={extra}")
        for key, value in catalog.items():
            if not value.strip():
                fail(f"empty iOS localization value for {code}:{key}")
        if code != "en":
            translated = sum(1 for key, value in catalog.items() if value != english[key])
            if translated < IOS_CORE_MIN_TRANSLATED:
                fail(
                    f"iOS locale {code} only translates {translated}/{len(english)} core strings; "
                    f"minimum is {IOS_CORE_MIN_TRANSLATED}"
                )

    project = read("ios/GhostFTP.xcodeproj/project.pbxproj")
    for marker in (
        "PBXVariantGroup",
        "Localizable.strings in Resources",
        "name = Localizable.strings;",
    ):
        if marker not in project:
            fail(f"iOS Xcode localization resource wiring is missing: {marker}")
    for code in SUPPORTED:
        apple_code = IOS_LOCALE.get(code, code)
        path_marker = f"{apple_code}.lproj/Localizable.strings"
        if path_marker not in project:
            fail(f"iOS Xcode project is missing locale resource path: {apple_code}")
        region_marker = f'\"{apple_code}\"' if "-" in apple_code else f"\t\t\t\t{apple_code},"
        if region_marker not in project:
            fail(f"iOS Xcode knownRegions is missing locale: {apple_code}")

    connection_view = read("ios/GhostFTP/ConnectionView.swift")
    for marker in (
        'Text("Ghost FTP")',
        'Text("Private, direct file transfer")',
        'Section("Saved connection")',
        'Section("Connection")',
        'String(localized: "Connecting…")',
        'String(localized: "Connect")',
    ):
        if marker not in connection_view:
            fail(f"iOS localized connection surface is missing marker: {marker}")
    if 'Text("GhostFTP")' in connection_view:
        fail("legacy public GhostFTP spelling remains in iOS connection UI")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    if len(SUPPORTED) < 21 or SUPPORTED[0] != "en":
        fail("localization registry must remain English-first with at least 21 languages")

    runtime = read("internal/i18n/i18n.go")
    if 'const DefaultLanguage = "en"' not in runtime:
        fail("English must remain the default runtime language")
    for code in SUPPORTED:
        if f'Code: "{code}"' not in runtime:
            fail(f"missing supported runtime language: {code}")
    for marker in ("TranslationCoverage", "Ghost FTP must expose at least 21 supported languages", "English must be the first"):
        if marker not in runtime:
            fail(f"runtime localization invariant is missing: {marker}")

    required_catalog_files = (
        "internal/i18n/catalogs.go",
        "internal/i18n/locale_de_fr.go",
        "internal/i18n/locale_es_tr.go",
        "internal/i18n/locale_el_pt.go",
        "internal/i18n/locale_zh_ru.go",
        "internal/i18n/locale_hi_ja.go",
        "internal/i18n/locale_additional.go",
        "internal/i18n/i18n_test.go",
    )
    for rel in required_catalog_files:
        read(rel)

    catalog = read("internal/i18n/catalogs.go")
    if '"en": {' not in catalog or '"hr": {' not in catalog:
        fail("canonical English and Croatian catalogs must be explicit")

    test_text = read("internal/i18n/i18n_test.go")
    for marker in (
        "ValidateCatalogs()",
        "expected 24 supported languages",
        "format verbs differ",
        "supplemental catalog",
        "TranslationCoverage",
    ):
        if marker not in test_text:
            fail(f"localization regression test is missing marker: {marker}")

    installer_messages = read("cmd/installer/messages.go")
    installer_language = read("cmd/installer/language.go")
    installer_main = read("cmd/installer/main.go")
    for code in SUPPORTED:
        if f'\"{code}\": {{' not in installer_messages:
            fail(f"installer primary copy is missing language: {code}")
    for marker in ("installerCopyFor", "installerConfirmTitle"):
        if marker not in installer_messages:
            fail(f"installer localization helper is missing: {marker}")
    for marker in ("setupCopy.CompletedTitle", "setupCopy.LaunchQuestion"):
        if marker not in installer_main:
            fail(f"installer localized primary flow is missing marker: {marker}")
    if "i18n.Languages()" not in installer_language:
        fail("installer language list is not derived from the canonical registry")

    settings = read("internal/model/types.go")
    if 'Language' not in settings or 'json:\"language,omitempty\"' not in settings:
        fail("language must be persisted in Settings")
    settings_store = read("internal/config/settings.go")
    for marker in ("i18n.DefaultLanguage", "i18n.Normalize", "i18n.IsSupported"):
        if marker not in settings_store:
            fail(f"settings language migration/validation is missing: {marker}")

    windows = read("internal/desktop/localization_windows.go")
    for marker in ("changeLanguageFromUI", "applyLanguage", "setButtonLabel", "reloadProtocolLabels", "applyColumnLanguage"):
        if marker not in windows:
            fail(f"Windows live localization is missing: {marker}")

    validate_ios_localization()

    entrypoint = read("cmd/ghostftp/main.go")
    for marker in (
        "credential is not available",
        "invalid authentication request",
        "Ghost FTP could not start.",
        "The Ghost FTP window could not be opened.",
    ):
        if marker not in entrypoint:
            fail(f"Windows English fallback marker is missing: {marker}")

    for legacy in (
        "GhostFTP closed unexpectedly.",
        "GhostFTP could not start.",
        "GhostFTP could not access the local application-data folder.",
        "The GhostFTP data folder is not safe to use.",
        "The GhostFTP window could not be opened.",
    ):
        if legacy in entrypoint:
            fail(f"legacy public brand remains in Windows fallback text: {legacy}")

    readme = read("README.md")
    for marker in (f"Current Ghost FTP version: **{version}**", "## Languages", "English"):
        if marker not in readme:
            fail(f"README is missing English-first marker: {marker}")

    forbidden_primary = {
        "README.md": ("Trenutačno izdanje:", "## Preuzimanje", "## Dokumentacija"),
        ".github/pull_request_template.md": ("## Ovlaštenje", "## Sažetak"),
        "cmd/ghostftp/main.go": (
            "vjerodajnica nije dostupna",
            "neispravan zahtjev za prijavu",
            "nepouzdan nadređeni proces",
            "Ghost FTP je neočekivano zatvoren",
            "Ghost FTP je već pokrenut",
            "Ghost FTP se ne može pokrenuti",
            "Ghost FTP podatkovna mapa nije sigurna",
            "Ghost FTP prozor nije moguće otvoriti",
        ),
    }
    for rel, phrases in forbidden_primary.items():
        text = read(rel)
        for phrase in phrases:
            if phrase in text:
                fail(f"primary English surface still contains legacy Croatian-only marker {phrase!r}: {rel}")

    print(f"LOCALIZATION_AUDIT=PASS ({version})")
    print("PRIMARY_LANGUAGE=en")
    print("WINDOWS_STARTUP_FALLBACKS=en")
    print("PUBLIC_BRAND=Ghost FTP")
    print("SUPPORTED_LANGUAGES=" + ",".join(SUPPORTED))
    print(f"SUPPORTED_LANGUAGE_COUNT={len(SUPPORTED)}")
    print("INSTALLER_PRIMARY_FLOW_LOCALIZED=YES")
    print("IOS_CORE_LOCALIZATION=PASS")
    print(f"IOS_LOCALIZATION_LANGUAGES={len(SUPPORTED)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
