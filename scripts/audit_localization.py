#!/usr/bin/env python3
"""Validate Ghost FTP's English-first Windows/Linux localization contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SUPPORTED = (
    "en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja",
    "it", "pl", "nl", "cs", "uk", "sv", "ro", "hu", "da", "fi", "no", "ko",
)


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

    if len(SUPPORTED) < 21 or SUPPORTED[0] != "en":
        fail("localization registry must remain English-first with at least 21 languages")

    runtime = read("internal/i18n/i18n.go")
    if 'const DefaultLanguage = "en"' not in runtime:
        fail("English must remain the default runtime language")
    for code in SUPPORTED:
        if f'Code: "{code}"' not in runtime:
            fail(f"missing supported runtime language: {code}")
    for marker in (
        "TranslationCoverage",
        "Ghost FTP must expose at least 21 supported languages",
        "English must be the first",
        "return DefaultLanguage",
    ):
        if marker not in runtime:
            fail(f"runtime localization invariant is missing: {marker}")

    for rel in (
        "internal/i18n/catalogs.go",
        "internal/i18n/locale_de_fr.go",
        "internal/i18n/locale_es_tr.go",
        "internal/i18n/locale_el_pt.go",
        "internal/i18n/locale_zh_ru.go",
        "internal/i18n/locale_hi_ja.go",
        "internal/i18n/locale_additional.go",
        "internal/i18n/i18n_test.go",
    ):
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
            fail(f"Windows Setup primary copy is missing language: {code}")
    for marker in ("installerCopyFor", "installerConfirmTitle"):
        if marker not in installer_messages:
            fail(f"Windows Setup localization helper is missing: {marker}")
    for marker in ("setupCopy.CompletedTitle", "setupCopy.LaunchQuestion"):
        if marker not in installer_main:
            fail(f"Windows Setup localized primary flow is missing marker: {marker}")
    if "i18n.Languages()" not in installer_language:
        fail("Windows Setup language list is not derived from the canonical registry")

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

    linux = read("internal/desktop/other.go")
    for marker in (
        "//go:build linux",
        "terminalLanguage(engine)",
        "setTerminalLanguage(engine",
        'i18n.T(language, "terminal.commands")',
        "i18n.LanguageByCode(language).NativeName",
    ):
        if marker not in linux:
            fail(f"Linux localization surface is missing: {marker}")

    entrypoint = read("cmd/ghostftp/main.go")
    for marker in (
        "credential is not available",
        "invalid authentication request",
        "Ghost FTP could not start.",
        "The Ghost FTP window could not be opened.",
    ):
        if marker not in entrypoint:
            fail(f"Windows English fallback marker is missing: {marker}")

    readme = read("README.md")
    for marker in (f"Current Ghost FTP version: **{version}**", "## Languages", "English", "24 languages"):
        if marker not in readme:
            fail(f"README is missing English-first marker: {marker}")

    print(f"LOCALIZATION_AUDIT=PASS ({version})")
    print("PRIMARY_LANGUAGE=en")
    print("PUBLIC_BRAND=Ghost FTP")
    print("SUPPORTED_LANGUAGES=" + ",".join(SUPPORTED))
    print(f"SUPPORTED_LANGUAGE_COUNT={len(SUPPORTED)}")
    print("WINDOWS_SETUP_LOCALIZED=YES")
    print("WINDOWS_LIVE_LOCALIZATION=YES")
    print("LINUX_RUNTIME_LOCALIZATION=YES")
    return 0


if __name__ == "__main__":
    sys.exit(main())
