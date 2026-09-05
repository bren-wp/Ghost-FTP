#!/usr/bin/env python3
from __future__ import annotations

from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(rel: str) -> str:
    return (ROOT / rel).read_text(encoding="utf-8")


def write(rel: str, text: str) -> None:
    (ROOT / rel).write_text(text, encoding="utf-8", newline="\n")


def replace_once(rel: str, old: str, new: str) -> None:
    text = read(rel)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"{rel}: expected one marker, found {count}: {old[:100]!r}")
    write(rel, text.replace(old, new, 1))


# Wire the selected setup language into the main installer flow.
replace_once(
    "cmd/installer/main.go",
    '''\tinstallLanguage, ok := selectInstallerLanguage()\n\tif !ok {\n\t\treturn 0\n\t}\n\n\tcontent := brand.Website + "\\n" + brand.Support + "\\n\\n" +\n\t\tbrand.ProductName + " will be installed for your Windows user account and will be available from the Start menu."\n''',
    '''\tinstallLanguage, ok := selectInstallerLanguage()\n\tif !ok {\n\t\treturn 0\n\t}\n\tsetupCopy := installerCopyFor(installLanguage)\n\n\tcontent := brand.Website + "\\n" + brand.Support + "\\n\\n" + setupCopy.ConfirmBody\n''',
)
replace_once(
    "cmd/installer/main.go",
    '''\t\t"Install "+brand.ProductFull+" "+version+"?",\n''',
    '''\t\tinstallerConfirmTitle(installLanguage, version),\n''',
)
replace_once(
    "cmd/installer/main.go",
    '''\t\tlanguageWarning = "\\n\\nThe selected language could not be saved. Ghost FTP will start in English; you can change the language in Settings."\n''',
    '''\t\tlanguageWarning = "\\n\\n" + setupCopy.LanguageWarning\n''',
)
replace_once(
    "cmd/installer/main.go",
    '''\t\tshortcutWarning = "\\n\\nA shortcut could not be created. You can start Ghost FTP from its installation folder."\n''',
    '''\t\tshortcutWarning = "\\n\\n" + setupCopy.ShortcutWarning\n''',
)
replace_once(
    "cmd/installer/main.go",
    '''\t\t"Setup completed successfully",\n\t\t"Ghost FTP is ready to use."+legacyCleanupWarning+languageWarning+shortcutWarning+"\\n\\nLaunch Ghost FTP now?",\n''',
    '''\t\tsetupCopy.CompletedTitle,\n\t\tsetupCopy.ReadyBody+legacyCleanupWarning+languageWarning+shortcutWarning+"\\n\\n"+setupCopy.LaunchQuestion,\n''',
)
replace_once(
    "cmd/installer/main.go",
    '''\t\t\t\t"Ghost FTP is installed",\n\t\t\t\t"Ghost FTP could not be launched automatically. Start it from the Windows Start menu.",\n''',
    '''\t\t\t\tsetupCopy.InstalledTitle,\n\t\t\t\tsetupCopy.LaunchFailed,\n''',
)

# Make the first language picker explicitly English-first and clearer about privacy.
replace_once(
    "cmd/installer/language.go",
    '''\t\t"Choose the language to use in Ghost FTP. You can change it later in Settings.",\n''',
    '''\t\t"Choose the setup language. English is the default.\\nNo telemetry · No ads · FTP · FTPS · SFTP",\n''',
)

# Improve the native setup language dialog layout without adding a UI framework.
replace_once(
    "internal/platform/language_windows.go",
    '''\t\t420, 250, 560, 230,\n''',
    '''\t\t360, 210, 640, 300,\n''',
)
replace_once(
    "internal/platform/language_windows.go",
    '''\tmakeControl("STATIC", instruction, 0, 24, 22, 500, 42, 0)\n\tstate.combo = makeControl("COMBOBOX", "", wsTabStop|wsVScroll|cbsDropdown, 24, 70, 500, 260, languageIDCombo)\n''',
    '''\tmakeControl("STATIC", instruction, 0, 32, 28, 576, 54, 0)\n\tstate.combo = makeControl("COMBOBOX", "", wsTabStop|wsVScroll|cbsDropdown, 32, 92, 576, 260, languageIDCombo)\n''',
)
replace_once(
    "internal/platform/language_windows.go",
    '''\tmakeControl("BUTTON", "Install", wsTabStop, 338, 132, 88, 32, languageIDInstall)\n\tmakeControl("BUTTON", "Cancel", wsTabStop, 436, 132, 88, 32, languageIDCancel)\n''',
    '''\tmakeControl("STATIC", "Ghost FTP Setup · private by design · local-first settings", 0, 32, 140, 576, 24, 0)\n\tmakeControl("BUTTON", "Continue", wsTabStop, 420, 182, 90, 34, languageIDInstall)\n\tmakeControl("BUTTON", "Cancel", wsTabStop, 518, 182, 90, 34, languageIDCancel)\n''',
)

# Strengthen localization audit around 24 languages, real coverage and setup parity.
audit = read("scripts/audit_localization.py")
audit = audit.replace(
    'SUPPORTED = ("en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja", "it", "pl", "nl", "cs", "uk", "sv")',
    'SUPPORTED = ("en", "hr", "de", "fr", "es", "tr", "el", "pt", "zh", "ru", "hi", "ja", "it", "pl", "nl", "cs", "uk", "sv", "ro", "hu", "da", "fi", "no", "ko")',
)
audit = audit.replace(
    'for marker in ("ValidateCatalogs()", "expected 18 supported languages", "format verbs differ"):',
    'for marker in ("ValidateCatalogs()", "expected 24 supported languages", "format verbs differ", "supplemental catalog", "TranslationCoverage"):',
)
needle = '''    settings = read("internal/model/types.go")\n'''
insert = '''    if len(SUPPORTED) < 21 or SUPPORTED[0] != "en":\n        fail("localization registry must remain English-first with at least 21 languages")\n\n    installer_messages = read("cmd/installer/messages.go")\n    installer_language = read("cmd/installer/language.go")\n    installer_main = read("cmd/installer/main.go")\n    for code in SUPPORTED:\n        if f'\\"{code}\\": {{' not in installer_messages:\n            fail(f"installer primary copy is missing language: {code}")\n    for marker in ("installerCopyFor", "installerConfirmTitle", "setupCopy.CompletedTitle", "setupCopy.LaunchQuestion"):\n        if marker not in installer_main and marker not in installer_messages:\n            fail(f"installer localization flow is missing marker: {marker}")\n    for marker in ("i18n.Languages()", "You can change it later in Settings"):\n        if marker not in installer_language and marker == "i18n.Languages()":\n            fail("installer language list is not derived from the canonical registry")\n\n    settings = read("internal/model/types.go")\n'''
if needle not in audit:
    raise SystemExit("localization audit insertion marker missing")
audit = audit.replace(needle, insert, 1)
audit = audit.replace(
    'print("SUPPORTED_LANGUAGES=" + ",".join(SUPPORTED))',
    'print("SUPPORTED_LANGUAGES=" + ",".join(SUPPORTED))\n    print(f"SUPPORTED_LANGUAGE_COUNT={len(SUPPORTED)}")\n    print("INSTALLER_PRIMARY_FLOW_LOCALIZED=YES")',
)
write("scripts/audit_localization.py", audit)

# Run dependency provenance as a permanent CI gate.
replace_once(
    ".github/workflows/ci.yml",
    '''          python scripts/audit_repository.py\n          python scripts/audit_version.py\n''',
    '''          python scripts/audit_repository.py\n          python scripts/audit_dependencies.py\n          python scripts/audit_version.py\n''',
)

# Add the development line without changing the published VERSION marker.
changelog = read("CHANGELOG.md")
if "## 1.1.0 - Unreleased" not in changelog:
    changelog = changelog.replace(
        "# Changelog\n\n",
        "# Changelog\n\n## 1.1.0 - Unreleased\n\n"
        "- Expanded the canonical English-first desktop/setup language registry from 18 to 24 languages, adding Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.\n"
        "- Added regional locale normalization including Norwegian `nb`/`nn` and Simplified Chinese aliases.\n"
        "- Replaced the weak eight-string supplemental-locale threshold with measured translation coverage and a 30-string translated-core floor for supplemental catalogs.\n"
        "- Added localized primary Windows Setup confirmation/completion/launch/warning copy for all 24 canonical languages while retaining English as the safe fallback.\n"
        "- Improved the dependency provenance contract: no external Go modules, zero Web Composer runtime dependencies, exactly pinned Android dependencies, and fail-closed rejection of tracking/ads/crash SDKs or dynamic versions.\n"
        "- Added detailed localization, dependency, settings and immutable release-history documentation.\n"
        "- 1.1.0 remains unreleased until Android, iOS and Web/PWA localization plus protocol/option work pass the full cross-platform release gate.\n\n",
        1,
    )
write("CHANGELOG.md", changelog)

# Root documentation: accurate current release plus explicit 1.1.0 development scope.
readme = read("README.md")
readme = readme.replace(
    "English is the default runtime language. Ghost FTP currently includes localization catalogs for English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian and Swedish.\n\nLanguage selection is persisted in application settings. New canonical user-facing text is maintained English-first and translated through the localization system.",
    "English is the primary and fallback runtime language. The 1.1.0 development line defines 24 canonical desktop/setup languages: English, Croatian, German, French, Spanish, Turkish, Greek, Portuguese, Simplified Chinese, Russian, Hindi, Japanese, Italian, Polish, Dutch, Czech, Ukrainian, Swedish, Romanian, Hungarian, Danish, Finnish, Norwegian and Korean.\n\nLanguage selection is persisted in application settings. Regional locale aliases normalize to canonical codes, and CI measures real translation coverage instead of counting an English-filled catalog as fully localized. Android, iOS and Web/PWA are being aligned to the same contract before 1.1.0 can be released. See [Localization](docs/LOCALIZATION.md).",
)
readme = readme.replace(
    "- [Installation](docs/INSTALLATION.md)\n",
    "- [Installation](docs/INSTALLATION.md)\n- [Localization](docs/LOCALIZATION.md)\n- [Settings](docs/SETTINGS.md)\n- [Dependencies](docs/DEPENDENCIES.md)\n- [Release history](docs/RELEASE-HISTORY.md)\n",
)
readme = readme.replace(
    "python scripts/audit_repository.py\npython scripts/audit_security.py",
    "python scripts/audit_repository.py\npython scripts/audit_dependencies.py\npython scripts/audit_localization.py\npython scripts/audit_security.py",
)
write("README.md", readme)

index = read("docs/README.md")
anchor = "- [Installation](INSTALLATION.md) — platform packages, installation expectations and prerequisites.\n"
addition = (
    anchor
    + "- [Localization](LOCALIZATION.md) — English-first language registry, translation coverage and platform rules.\n"
    + "- [Settings](SETTINGS.md) — persisted options, safe defaults and migration semantics.\n"
    + "- [Dependencies](DEPENDENCIES.md) — dependency allowlist, provenance and no-tracking policy.\n"
    + "- [Release history](RELEASE-HISTORY.md) — detailed immutable history of published Ghost FTP releases and unreleased development scope.\n"
)
if anchor not in index:
    raise SystemExit("docs index anchor missing")
index = index.replace(anchor, addition, 1)
write("docs/README.md", index)

print("ONE_SHOT_110_FOUNDATION_PATCH=PASS")
