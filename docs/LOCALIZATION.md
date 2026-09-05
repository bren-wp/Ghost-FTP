# Localization

Ghost FTP uses an English-first localization model. English (`en`) is the canonical source language, the first language presented by setup, and the only runtime fallback when a translated string is unavailable.

## Supported language registry

The 1.1.0 development line defines 24 canonical desktop/setup language codes:

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

Common regional forms normalize to the canonical code. For example, `pt-BR` resolves to `pt`, `de-DE` to `de`, `zh-Hans`/`zh-CN` to `zh`, and `nb-NO`/`nn-NO` to `no`.

## Runtime rules

`internal/i18n/i18n.go` owns the canonical registry, normalization, fallback and catalog validation. Every advertised language must have a catalog with the exact same key set as English and compatible formatting verbs. Empty translations, unknown keys and duplicate language codes are release-blocking.

Full legacy catalogs must remain substantially localized. Supplemental catalogs introduced later must contain a meaningful translated core instead of merely copying the English catalog. `TranslationCoverage` and the localization tests measure this directly.

The terminal `language` command derives its accepted language list from the same registry; hard-coded duplicate language lists are not allowed.

## Windows application

The Windows desktop application persists the selected language in normal application settings and supports live label/column/protocol refresh without requiring the user to recreate a profile.

Startup and catastrophic-error text stays English-first because those paths may run before settings or localization state can be safely loaded.

## Windows Setup

The first setup language selector is intentionally English-first so a damaged or unknown locale never makes setup unusable. After the user selects a language, the primary confirmation, completion, launch and warning flow uses `cmd/installer/messages.go` and the same canonical locale normalization as the application.

Installer payload verification, rollback and error containment are not weakened by localization. Technical failure detail may fall back to English until every installer recovery branch has a reviewed translation; an English fallback is preferable to an empty or misleading security message.

## Android, iOS and Web/PWA

The 1.1.0 release is not considered localization-complete until Android, iOS and Web/PWA expose the same canonical language choices or a documented platform-safe subset and their localization audits verify key coverage.

Android already centralizes user-visible copy in `res/values/strings.xml`; localized `values-*` resources must preserve all format placeholders. iOS user-visible SwiftUI strings must be moved into the localization resource layer rather than duplicated as hard-coded literals. Web/PWA localization must not put user-selected language or translations into unsafe HTML without normal escaping.

## Adding or changing a language

1. Add or update the canonical language entry in `internal/i18n/i18n.go`.
2. Add a complete catalog or reviewed supplemental catalog with sufficient real translation coverage.
3. Preserve all `%s`, `%d` and other formatting verbs exactly as required by the English source string.
4. Update installer copy where the language is setup-supported.
5. Add/adjust platform localization resources.
6. Add normalization/affirmative-answer tests when the locale has relevant regional codes.
7. Run `go test ./...` and `python scripts/audit_localization.py`.
8. Run the full multi-platform CI matrix before release.

Machine translation may be used only as a draft. Security, credential, destructive-action, recovery and overwrite messages require human review before a release is declared translation-complete.
