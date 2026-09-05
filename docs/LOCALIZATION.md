# Localization

Ghost FTP uses an English-first localization model. English (`en`) is the canonical source language, the first safe setup/runtime fallback, and the language used whenever a translated string is unavailable or cannot be loaded safely.

## Supported language registry

Ghost FTP 1.1.0 defines 24 canonical language codes:

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

Platform resource identifiers are allowed to use the native platform convention while preserving the same canonical meaning. iOS therefore maps canonical `zh` to `zh-Hans` and canonical `no` to `nb` in its localized resource directories.

## Runtime rules

`internal/i18n/i18n.go` owns the canonical desktop/setup registry, normalization, fallback and catalog validation. Every advertised desktop/setup language must have the exact English key set with compatible formatting verbs. Empty translations, unknown keys, duplicate language codes and invalid aliases are release-blocking.

Full legacy catalogs must remain substantially localized. Supplemental catalogs introduced later must contain a meaningful translated core instead of merely copying the English catalog. `TranslationCoverage` and localization tests measure this directly.

Language selection is persisted in normal application settings. Unknown or unsupported persisted language values normalize safely to English.

## Windows application

The Windows desktop application persists the selected language and supports live label, column and protocol refresh without requiring the user to recreate a profile.

Startup and catastrophic-error text remains English-first because those paths may run before settings or localization state can be loaded safely.

## Windows Setup

The first setup language selector is intentionally English-first so a damaged or unknown locale never makes setup unusable. After the user selects a language, the primary confirmation, completion, launch and warning flow uses `cmd/installer/messages.go` and the same canonical locale normalization as the application.

Installer payload verification, rollback and error containment are not weakened by localization. Technical failure detail may fall back to English where no reviewed localized recovery message exists; an English fallback is preferable to an empty or misleading security message.

## Android

Android ships native resource catalogs for the same 24-language registry. User-visible resource keys are validated for exact parity with the canonical Android English resource set, including format placeholders and locale-directory drift.

`scripts/audit_android_localization.py` is fail-closed: a missing language, missing/extra key, incompatible formatting placeholder, dynamic dependency drift or effectively untranslated locale blocks the release gate.

Android follows platform locale selection. Passwords/passphrases remain session-only and are not persisted as part of localization or remembered connection state.

## iOS

iOS ships native `Localizable.strings` resources for all 24 canonical languages. The Xcode project includes them through a `PBXVariantGroup` in the application Resources build phase rather than leaving translation files disconnected from the packaged app.

Canonical platform mappings are explicit:

- `zh` → `zh-Hans.lproj`
- `no` → `nb.lproj`

The localization audit verifies the exact locale set, key parity, non-empty values, meaningful translated coverage, Xcode resource wiring and the dynamic runtime lookup used by the connection status control. The public iOS artifact remains an unsigned arm64 IPA; localization does not change its signing/provisioning status.

## Web/PWA

Web/PWA uses `GhostFTP\\I18n` as its server-side language registry and persists the canonical language in the existing per-user `client_state`. No localization cookie, telemetry identifier, external translation service or third-party i18n framework is required.

The authenticated shell receives only the normalized language code and an HTML-escaped JSON core catalog through same-origin page metadata. JavaScript uses English fallback text when a core key is unavailable. Changing language uses the existing CSRF-protected `me` / `save_preferences` preference API and then reloads the shell so server-rendered document language and runtime catalog remain synchronized.

The Web/PWA 1.1.0 core localization covers the connection-oriented catalog and high-frequency dynamic file-browser statuses/actions. The remaining static administrative/support shell copy is intentionally English-first until reviewed translations are added; Ghost FTP does not represent untranslated English fallback copy as a completed translation.

Locale-sensitive client operations, including file-name filtering/sorting, use the persisted canonical locale rather than a hard-coded Croatian locale. The PWA continues to exclude authenticated navigation, API, account, setup, diagnostics, download and preview responses from offline cache.

Runtime Web tests validate:

- exactly 24 canonical codes;
- English as the default/fallback;
- regional alias normalization;
- equal core key sets;
- non-empty translations and minimum real translation coverage;
- persisted language sanitization;
- server-to-JavaScript language/catalog wiring;
- absence of the previous hard-coded Croatian file-filter locale;
- English-first network/PWA fallback behavior.

## Adding or changing a language

1. Add or update the canonical language entry in `internal/i18n/i18n.go`.
2. Add a complete desktop/setup catalog or a reviewed supplemental catalog with sufficient real translation coverage.
3. Preserve all `%s`, `%d` and other formatting verbs required by the English source string.
4. Update Windows Setup copy where the language is setup-supported.
5. Add or update Android native resources with exact key/placeholder parity.
6. Add or update the corresponding iOS localized resource and Xcode resource membership.
7. Add or update the Web/PWA core catalog where the user-facing surface is supported.
8. Add normalization/affirmative-answer tests when the locale has relevant regional forms.
9. Run `go test ./...`, `python scripts/audit_localization.py`, `python scripts/audit_android_localization.py` and `python scripts/audit_web.py`.
10. Run the complete multi-platform CI matrix before publication.

Machine translation may be used as a draft. Security, credential, destructive-action, recovery and overwrite messages require review before a release is described as translation-complete for that surface.
