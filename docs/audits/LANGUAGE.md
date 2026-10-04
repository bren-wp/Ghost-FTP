# Ghost FTP Language Audit — 0.30.6

The desktop UI advertises 24 locales: English, Hrvatski, Čeština, Slovenčina, Magyar, Română, Български, Ελληνικά, Türkçe, Українська, Dansk, Svenska, Norsk, Deutsch, Français, Español, Italiano, Português, Nederlands, Polski, Slovenščina, Srpski, Bosanski and Македонски.

Albanian is not an advertised Ghost FTP locale and the i18n parity contract rejects it if it is reintroduced accidentally.

The desktop `npm run check:i18n` contract keeps every advertised non-English core dictionary aligned with the Croatian canonical key set. The reference UI also has parity coverage across all 24 advertised locales; completion dictionaries cover the locale groups that were previously incomplete.

Key parity is not the same as linguistic acceptance. Target-OS QA must still review terminology, grammar, clipping, narrow-window layouts, protocol/security terminology, placeholders and accessibility labels.

Android currently has production-safe English UI copy in Kotlin and does not yet advertise the desktop's 24-language set. Android must not claim full multilingual parity until user-facing strings are migrated to locale resources and automated parity checks cover them.

Status: desktop source dictionary/reference-UI parity is CI-gated for the 0.30.6 line. Android localization remains a tracked product gap rather than a false completed claim.
