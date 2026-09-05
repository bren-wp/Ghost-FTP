#!/usr/bin/env python3
"""Fail closed when Android localization drifts from the 24-language contract."""

from __future__ import annotations

import re
import sys
import xml.etree.ElementTree as ET
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
RES = ROOT / "android/app/src/main/res"
LANGUAGES = (
    "en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja",
    "it", "pl", "nl", "cs", "uk", "sv", "ro", "hu", "da", "fi", "no", "ko",
)
PLACEHOLDER = re.compile(r"%(?:\d+\$[a-zA-Z]|%)")
NON_TRANSLATABLE = {"app_name", "root_path", "connected_with_diagnostics"}


def fail(message: str) -> None:
    raise SystemExit("ANDROID_LOCALIZATION_AUDIT_FAILED: " + message)


def catalog(path: Path) -> dict[str, str]:
    if not path.is_file():
        fail(f"missing Android catalog: {path.relative_to(ROOT)}")
    try:
        root = ET.parse(path).getroot()
    except ET.ParseError as exc:
        fail(f"invalid XML in {path.relative_to(ROOT)}: {exc}")
    if root.tag != "resources":
        fail(f"unexpected Android resource root in {path.relative_to(ROOT)}")
    values: dict[str, str] = {}
    for item in root.findall("string"):
        name = item.attrib.get("name", "").strip()
        if not name:
            fail(f"unnamed string in {path.relative_to(ROOT)}")
        if name in values:
            fail(f"duplicate Android string {name!r} in {path.relative_to(ROOT)}")
        text = "".join(item.itertext()).strip()
        if not text:
            fail(f"empty Android string {name!r} in {path.relative_to(ROOT)}")
        values[name] = text
    return values


def path_for(language: str) -> Path:
    if language == "en":
        return RES / "values/strings.xml"
    return RES / f"values-{language}/strings.xml"


def main() -> int:
    if LANGUAGES[0] != "en" or len(LANGUAGES) != 24:
        fail("Android language registry must contain exactly 24 languages with English first")

    english = catalog(path_for("en"))
    if len(english) < 50:
        fail(f"English Android catalog unexpectedly small: {len(english)} strings")
    expected_keys = set(english)

    for language in LANGUAGES[1:]:
        values = catalog(path_for(language))
        keys = set(values)
        missing = sorted(expected_keys - keys)
        extra = sorted(keys - expected_keys)
        if missing or extra:
            fail(f"{language} key drift; missing={missing}, extra={extra}")

        translated = 0
        eligible = 0
        for key in sorted(expected_keys):
            source = english[key]
            target = values[key]
            if sorted(PLACEHOLDER.findall(source)) != sorted(PLACEHOLDER.findall(target)):
                fail(f"{language}.{key} format placeholders differ from English")
            if key not in NON_TRANSLATABLE:
                eligible += 1
                if target != source:
                    translated += 1
        coverage = translated / eligible if eligible else 1.0
        if coverage < 0.80:
            fail(f"{language} Android translation coverage is only {coverage:.1%}; require >= 80%")

    discovered = sorted(
        path.parent.name.removeprefix("values-")
        for path in RES.glob("values-*/strings.xml")
        if path.parent.name.startswith("values-")
    )
    expected_localized = sorted(LANGUAGES[1:])
    if discovered != expected_localized:
        fail(f"Android locale directory drift; expected={expected_localized}, got={discovered}")

    print("ANDROID_LOCALIZATION_AUDIT=PASS")
    print("ANDROID_PRIMARY_LANGUAGE=en")
    print("ANDROID_SUPPORTED_LANGUAGE_COUNT=24")
    print("ANDROID_STRING_KEY_PARITY=100_PERCENT")
    print("ANDROID_FORMAT_PLACEHOLDERS=PARITY_ENFORCED")
    print("ANDROID_MIN_TRANSLATION_COVERAGE=80_PERCENT")
    return 0


if __name__ == "__main__":
    sys.exit(main())
