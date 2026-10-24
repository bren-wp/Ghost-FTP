# Ghost FTP language status — 0.99.9 Preview

The active source currently defines 24 advertised language codes:
`en, hr, cs, sk, hu, ro, bg, el, tr, uk, da, sv, no, de, fr, es, it, pt, nl, pl, sl, sr, bs, mk`.
Do not advertise Albanian (`sq`), which has no approved dictionary.

## Current implementation and verified scope

- **Windows and Linux shared desktop:** the existing `src/lib/i18n.ts`, reference completion and support dictionaries cover the same 24 language codes. `node ghostftp-desktop/scripts/check-i18n-parity.mjs` validates 186 canonical core keys plus 77 reference UI keys; that test does **not** prove every visible desktop value is translated. Runtime textual translation uses the existing desktop localization mechanism.
- **Android:** all 24 `values-*/strings.xml` sets contain the same 153 preexisting base keys. Release 0.99.9 adds ten fully translated user-facing UI action labels in separate `strings_ui_extra.xml` resources for all 24 locales, and replaces 25 corresponding hard-coded literals in `MainActivity.kt` with `getString(R.string...)`. `android/scripts/check-multilingual-ui.mjs` checks parity. Other direct English strings in Kotlin still require migration.
- **Native macOS Swift package:** a new resource bundle `ghostftp-translations.json` contains 269 entries in each of 23 non-English dictionaries; English is the canonical key language. The `L10n.text` runtime selects the OS-preferred language and accepts Norwegian `nb/nn` as `no`. Native files, sites, cloud, sync and help navigation and many SwiftUI controls now use those lookups. Untranslated macOS-specific copy falls back to English; this is explicitly **not** full native translation certification.

## Mandatory continuing work

1. Extract all remaining user-visible Swift and Kotlin text, including dynamic messages and accessibility labels, from code into localized catalogs; never translate server paths, saved credentials or protocol keywords.
2. Provide approved idiomatic translations for each remaining string in every advertised locale; enforce placeholders, plurals, Unicode, accessibility and truncation checks. Every missing translation must remain tracked and visible rather than silently marketed as complete.
3. Test language switching/system-language restoration, persisted profile compatibility and narrow-screen content on installed app builds.
4. Keep native macOS explicit FTPS/SFTP disabled pending verified protocol implementation and E2E tests in issue #223; translation of security warnings must **not** imply implementation.

**Status:** localization infrastructure and a targeted cross-platform tranche delivered; full all-platform zero-hardcoded-string acceptance is not yet complete.
