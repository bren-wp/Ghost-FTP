# Ghost FTP localization

Ghost FTP **1.0.0 Stable** uses an English-first, entirely local localization model. English (`en`) is the canonical source language, default locale and safe fallback when a translated value is unavailable or invalid.

## Supported languages

The maintained desktop catalog contains exactly **24 selectable languages**:

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

Common regional forms normalize to the canonical registry where defined, for example `pt-BR` → `pt`, `de-DE` → `de`, Simplified Chinese regional aliases → `zh`, and Norwegian variants → `no`.

## Canonical registry

`internal/i18n/i18n.go` owns supported-language metadata, normalization/aliases, English fallback, catalog validation, affirmative-answer matching and translation coverage measurement.

Every advertised locale must preserve the English key/format contract. Empty strings, incompatible formatting verbs, duplicate codes and invalid aliases are release-blocking defects.

## Offline privacy boundary

Localization requires no online translation service. Ghost FTP does not send filenames, hostnames, credentials, server diagnostics or selected language to a translation provider.

Language selection is local settings state, not analytics/profile-segmentation data.

## Windows

The native Windows frontend supports live language switching for catalog-backed UI. A language change refreshes visible product text without changing protocol/security state.

Startup/catastrophic fallback copy can remain English because those paths may execute before persisted settings are safely available.

## Windows Setup

Setup consumes the same canonical language registry and maintains localized primary install/maintenance copy. Install integrity, rollback, signing and file validation remain independent of translated text.

A missing translation must fall back to English rather than weakening a security decision or rendering an empty warning.

## Linux

The native Linux frontend consumes the same stored locale and translation catalog. Runtime language switching and normalization use the same canonical codes/fallback rules.

Linux UI localization must not create a second set of protocol strings with different security semantics.

## Security-sensitive text

Credential prompts, destructive-operation confirmations, recovery/overwrite warnings and host-trust decisions require careful review. Translation code must never infer transport or trust state from human-readable labels; typed protocol/security state remains authoritative.

## Adding or improving a language

1. update the canonical language/catalog source;
2. preserve every required English key;
3. preserve compatible format verbs/placeholders;
4. update Setup primary copy where relevant;
5. verify Windows live language switching;
6. verify Linux runtime language switching;
7. add alias/affirmative tests where needed;
8. run `scripts/audit_localization.py` and the full regression suite.

## CI contract

Localization verification checks:

- English is first/default/fallback;
- exactly 24 canonical languages remain registered;
- catalog keys and format verbs are valid;
- minimum real translation coverage remains satisfied;
- Windows localization wiring remains active;
- Setup primary copy remains covered;
- Linux runtime switching remains active;
- README/docs advertise the same 24-language contract.

Localization drift is a release failure, not a documentation-only issue.
