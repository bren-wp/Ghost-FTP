# Ghost FTP language status — 0.99.10 Preview

The active source currently defines 24 advertised language codes:
`en, hr, cs, sk, hu, ro, bg, el, tr, uk, da, sv, no, de, fr, es, it, pt, nl, pl, sl, sr, bs, mk`.
Do not advertise Albanian (`sq`), which has no approved dictionary.

## Current implementation and verified scope

- **Windows and Linux shared desktop:** the existing `src/lib/i18n.ts`, reference completion and support dictionaries cover the same 24 language codes. `node ghostftp-desktop/scripts/check-i18n-parity.mjs` validates 186 canonical core keys plus 77 reference UI keys; that test does **not** prove every visible desktop value is translated. Runtime textual translation uses the existing desktop localization mechanism.
- **Android:** all 24 `values-*/strings.xml` sets contain the same 153 preexisting base keys. Release 0.99.9 added ten translated action labels. Release 0.99.10 adds 15 accessibility and workflow labels in `strings_ui_accessibility.xml` for all 24 locales, migrating another 21 inline UI references in `MainActivity.kt` to `getString(R.string...)`. The Android checker validates parity for both resource catalogs (25 extra keys across 24 locales). Other direct English strings in Kotlin still require migration.
- **Native macOS Swift package:** a new resource bundle `ghostftp-translations.json` contains 269 entries in each of 23 non-English dictionaries; English is the canonical key language. The `L10n.text` runtime selects the OS-preferred language and accepts Norwegian `nb/nn` as `no`. Native files, sites, cloud, sync and help navigation and many SwiftUI controls now use those lookups. Untranslated macOS-specific copy falls back to English; this is explicitly **not** full native translation certification.

## Mandatory continuing work

1. Extract all remaining user-visible Swift and Kotlin text, including dynamic messages and accessibility labels, from code into localized catalogs; never translate server paths, saved credentials or protocol keywords.
2. Provide approved idiomatic translations for each remaining string in every advertised locale; enforce placeholders, plurals, Unicode, accessibility and truncation checks. Every missing translation must remain tracked and visible rather than silently marketed as complete.
3. Test language switching/system-language restoration, persisted profile compatibility and narrow-screen content on installed app builds.
4. Keep native macOS explicit FTPS/SFTP disabled pending verified protocol implementation and E2E tests in issue #223; translation of security warnings must **not** imply implementation.

### Quantified source debt (regression baseline)

The CI inventory counts **123** direct Android Activity UI literals, **0** directly hard-coded static SwiftUI view labels in the three migrated native view files, and **67** native Swift view keys not yet present in all translation dictionaries. The 24-language dictionaries themselves are not a substitute for translating 100% of runtime UI and errors. `tools/audits/check-localization-inventory.mjs` blocks increases in these baseline counts. Reduce these numbers to **zero**, then remove the temporary debt budgets before full-localization acceptance. SF Symbol identifiers, protocol commands, filenames, hostnames and URLs are intentionally not treated as translatable UI strings.

**Status:** localization infrastructure and a targeted cross-platform tranche delivered; full all-platform zero-hardcoded-string acceptance is not yet complete.
