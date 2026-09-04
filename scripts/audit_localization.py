#!/usr/bin/env python3
"""Validate Ghost FTP's English-first localization contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SUPPORTED = ("en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja", "it", "pl", "nl", "cs", "uk", "sv")


def fail(message: str) -> None:
    raise SystemExit("LOCALIZATION_AUDIT_FAILED: " + message)


def read(rel: str) -> str:
    path = ROOT / rel
    if not path.is_file():
        fail(f"missing required file: {rel}")
    return path.read_text(encoding="utf-8")


def main() -> int:
    version = read("VERSION").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")

    runtime = read("internal/i18n/i18n.go")
    if 'const DefaultLanguage = "en"' not in runtime:
        fail("English must remain the default runtime language")
    for code in SUPPORTED:
        if f'Code: "{code}"' not in runtime:
            fail(f"missing supported runtime language: {code}")

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
    for marker in ("ValidateCatalogs()", "expected 18 supported languages", "format verbs differ"):
        if marker not in test_text:
            fail(f"localization regression test is missing marker: {marker}")

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

    entrypoint = read("cmd/ghostftp/main.go")
    for marker in (
        "credential is not available",
        "invalid authentication request",
        "Ghost FTP could not start.",
        "The Ghost FTP window could not be opened.",
    ):
        if marker not in entrypoint:
            fail(f"Windows English fallback marker is missing: {marker}")

    # User-facing fallback text must not regress to the retired product name.
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
    return 0


if __name__ == "__main__":
    sys.exit(main())
