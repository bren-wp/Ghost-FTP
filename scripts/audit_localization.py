#!/usr/bin/env python3
"""Validate ByFTP's English-first localization and responsive-UI contract."""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SUPPORTED = ("en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja")
SECONDARY = ("de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja")


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
        "internal/i18n/action_catalogs.go",
        "internal/i18n/action_locale_registry.go",
        "internal/i18n/action_locale_de_fr.go",
        "internal/i18n/action_locale_es_tr.go",
        "internal/i18n/action_locale_el_pt.go",
        "internal/i18n/action_locale_zh_ru.go",
        "internal/i18n/action_locale_hi_ja.go",
        "internal/i18n/i18n_test.go",
    )
    for rel in required_catalog_files:
        read(rel)

    catalog = read("internal/i18n/catalogs.go")
    if '"en": {' not in catalog or '"hr": {' not in catalog:
        fail("canonical English and Croatian catalogs must be explicit")
    registry = read("internal/i18n/action_locale_registry.go")
    for code in SECONDARY:
        if f'"{code}": actionLocale(' not in registry:
            fail(f"missing complete action workflow catalog registration for {code}")
    for marker in ("actionLocaleKeys", "key/value count mismatch", "registerSecondaryActionCatalogs"):
        if marker not in registry:
            fail(f"action-localization registry is missing invariant: {marker}")

    test_text = read("internal/i18n/i18n_test.go")
    for marker in ("ValidateCatalogs()", "expected 12 supported languages", "format verbs differ", "fall back to English too often"):
        if marker not in test_text:
            fail(f"localization regression test is missing marker: {marker}")

    settings = read("internal/model/types.go")
    if 'Language' not in settings or 'json:"language,omitempty"' not in settings:
        fail("language must be persisted in Settings")
    settings_store = read("internal/config/settings.go")
    for marker in ("i18n.DefaultLanguage", "i18n.Normalize", "i18n.IsSupported"):
        if marker not in settings_store:
            fail(f"settings language migration/validation is missing: {marker}")

    windows = read("internal/desktop/localization_windows.go")
    for marker in ("changeLanguageFromUI", "applyLanguage", "setButtonLabel", "reloadProtocolLabels", "applyColumnLanguage", "layoutResponsive"):
        if marker not in windows:
            fail(f"Windows live localization is missing: {marker}")

    measured = read("internal/desktop/button_measure_windows.go")
    flow = read("internal/desktop/button_flow_windows.go")
    responsive = read("internal/desktop/layout_responsive_windows.go")
    renderer = read("internal/desktop/button_draw_windows.go")
    for marker in ("GetTextExtentPoint32W", "preferredButtonWidth"):
        if marker not in measured:
            fail(f"localized button measurement is missing: {marker}")
    for marker in ("placeButtonFlow", "buttonFlowRows"):
        if marker not in flow:
            fail(f"responsive button-flow layout is missing: {marker}")
    for marker in ("preferredButtonWidth", "placeButtonFlow", "resizeListColumns"):
        if marker not in responsive:
            fail(f"responsive multilingual layout is missing: {marker}")
    if "dtEndEllipsis" in renderer:
        fail("owner-drawn buttons must show the complete localized label; ellipsis is forbidden")

    readme = read("README.md")
    for marker in (f"Current release: {version}", "## Languages", "English"):
        if marker not in readme:
            fail(f"README is missing English-first marker: {marker}")

    primary_docs = (
        "docs/README.md", "docs/INSTALACIJA.md", "docs/ARHITEKTURA.md", "docs/SIGURNOST.md",
        "docs/PRIVATNOST.md", "docs/TESTIRANJE.md", "docs/PODRSKA.md", "docs/PROVJERA-IZDANJA.md",
        "docs/POTPISIVANJE.md", "docs/DOPRINOS.md", "docs/IZDAVANJE-NA-GITHUBU.md",
        "docs/PLAN-RAZVOJA.md", "docs/OBAVIJESTI-TRECIH-STRANA.md", "docs/SHARED-HOSTING.md",
    )
    croatian_only_markers = ("# Instalacija", "# Sigurnost", "# Privatnost", "# Podrška", "## Preuzimanje", "Trenutačno izdanje")
    for rel in primary_docs:
        text = read(rel)
        for phrase in croatian_only_markers:
            if phrase in text:
                fail(f"canonical documentation is not English-first: {rel} contains {phrase!r}")

    forbidden_primary = {
        "README.md": ("Trenutačno izdanje:", "## Preuzimanje", "## Dokumentacija"),
        ".github/pull_request_template.md": ("## Ovlaštenje", "## Sažetak"),
        ".github/workflows/release.yml": ("Objavi ByFTP", "audit_croatian.py", "Dohvati izvorni kod", "Dokumentacija"),
    }
    for rel, phrases in forbidden_primary.items():
        text = read(rel)
        for phrase in phrases:
            if phrase in text:
                fail(f"primary English surface still contains legacy Croatian-only marker {phrase!r}: {rel}")

    print(f"LOCALIZATION_AUDIT=PASS ({version})")
    print("PRIMARY_LANGUAGE=en")
    print("SUPPORTED_LANGUAGES=" + ",".join(SUPPORTED))
    print("FULL_BUTTON_LABELS=ENFORCED")
    return 0


if __name__ == "__main__":
    sys.exit(main())
