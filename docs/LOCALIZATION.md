# Localization

Ghost FTP uses an **English-first** localization model. English (`en`) is the canonical source language, default locale and safe fallback whenever a translated string is unavailable or invalid.

## Supported languages

Ghost FTP 2.x exposes 24 canonical language codes:

| Code | Language | Native name |
| --- | --- | --- |
| `en` | English | English |
| `hr` | Croatian | Hrvatski |
| `de` | German | Deutsch |
| `fr` | French | Français |
| `es` | Spanish | Español |
| `tr` | Turkish | Türkçe |
| `el` | Greek | Ελληνικά |
| `pt` | Portuguese | Português |
| `zh` | Chinese (Simplified) | 简体中文 |
| `ru` | Russian | Русский |
| `hi` | Hindi | हिन्दी |
| `ja` | Japanese | 日本語 |
| `it` | Italian | Italiano |
| `pl` | Polish | Polski |
| `nl` | Dutch | Nederlands |
| `cs` | Czech | Čeština |
| `uk` | Ukrainian | Українська |
| `sv` | Swedish | Svenska |
| `ro` | Romanian | Română |
| `hu` | Hungarian | Magyar |
| `da` | Danish | Dansk |
| `fi` | Finnish | Suomi |
| `no` | Norwegian | Norsk |
| `ko` | Korean | 한국어 |

Common regional forms normalize to the canonical registry. Examples include `pt-BR` → `pt`, `de-DE` → `de`, `zh-Hans`/`zh-CN` → `zh`, and `nb`/`nn` → `no`.

## Canonical runtime registry

`internal/i18n/i18n.go` owns:

- supported language metadata;
- normalization/aliases;
- English fallback;
- catalog validation;
- affirmative-answer matching;
- translation coverage measurement.

Every advertised runtime language must preserve the English key/format contract. Empty strings, incompatible format verbs, duplicate language codes or invalid aliases are release-blocking.

## Translation coverage

Ghost FTP does not count an English copy as a completed translation. Supplemental catalogs must exceed the minimum translated-core coverage enforced by tests/audits.

Security-sensitive copy—credential prompts, destructive actions, recovery/overwrite warnings and trust decisions—requires human review before the corresponding surface is described as translation-complete.

Machine translation can be used as a drafting aid but does not override review requirements.

## Windows application

The native Windows frontend supports live language switching. A language change refreshes labels, protocol names, list columns and supported status/action text without requiring a new profile.

Startup/catastrophic fallback text remains English-first because those paths may execute before persisted localization state is safely available.

## Windows Setup

Setup uses the same canonical language registry. The language selector is English-first and localized primary confirmation/completion/launch/warning copy is maintained for all canonical languages.

Installer integrity, rollback and path validation remain independent of translated text. Technical recovery detail may safely fall back to English instead of rendering an empty or misleading localized security message.

## Linux

Linux reads the same persisted language setting and uses the same catalogs as the Windows core.

The terminal frontend can change language at runtime using the supported canonical code. Invalid language state normalizes to English.

The Linux localization contract covers connection prompts/status, remote operations, transfer status and other catalog-backed terminal messages. Additional hard-coded terminal helper text should continue moving into the canonical catalog as translations are reviewed.

## Web companion

The separate Web companion uses its own PHP/PWA registry synchronized to the same 24-language product set and English fallback principle.

The Web companion does not use an external translation service, localization tracking cookie or third-party i18n framework. User language preference is stored in existing authenticated client state and sensitive navigation/API responses remain excluded from PWA caching.

The Web companion is not a Windows/Linux desktop platform artifact.

## Adding or changing a language

For desktop/setup changes:

1. add/update the canonical language in `internal/i18n/i18n.go`;
2. add/update the catalog while preserving every required key;
3. preserve `%s`, `%d` and all other English format verbs;
4. update Windows Setup primary copy;
5. verify Windows live localization;
6. verify Linux runtime language switching;
7. add alias/affirmative tests when regional forms require them;
8. run the localization audit and complete CI.

For Web companion translation changes, update its registry/catalog separately and run `scripts/audit_web.py`.

## CI contract

`scripts/audit_localization.py` verifies:

- English is the first/default locale;
- exactly the canonical 24-language desktop registry is present;
- required catalog source/test files exist;
- translation coverage and format-contract tests remain wired;
- Windows live localization remains wired;
- Windows Setup copy covers every supported language;
- Linux runtime localization remains wired;
- root documentation advertises the same 24-language/English-first contract.

Localization drift is therefore a release failure, not a documentation-only defect.

## Retired platform history

Historical 1.x commits may contain Android/iOS localization resources because those applications were active at that time. They are not part of the 2.x localization gate and must not be treated as current platform requirements.
